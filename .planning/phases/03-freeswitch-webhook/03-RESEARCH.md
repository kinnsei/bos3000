# Phase 3: FreeSWITCH 集成 + 录音 + Webhook - Research

**Researched:** 2026-03-10
**Domain:** FreeSWITCH ESL integration (eslgo), call recording pipeline (FS + ffmpeg + S3), Webhook delivery system (Encore Pub/Sub + HMAC), FS high availability
**Confidence:** HIGH

## Summary

Phase 3 replaces the Phase 2 Mock FSClient with a real eslgo-based implementation connecting to FreeSWITCH via ESL (Event Socket Layer, port 8021). The eslgo library (v1.5.0, github.com/percipia/eslgo) provides idiomatic Go bindings for inbound ESL connections with `Dial()`, `OriginateCall()`, `HangupCall()`, event listeners by UUID, and raw command execution via `SendCommand()`. The FSClient interface (5 methods + RegisterEventHandler) defined in Phase 2 maps cleanly to eslgo's API.

The recording pipeline uses FreeSWITCH's `uuid_record` command (sent via ESL) to record each leg separately. For split-leg recording, two `uuid_record start` commands are issued -- one on the A-leg UUID and one on the B-leg UUID -- each writing to separate WAV files on the FS host. After call termination, an Encore Pub/Sub message triggers an async worker that runs ffmpeg to merge the two mono WAV files into a stereo MP3, uploads the result to Encore Object Storage (S3-compatible), and generates presigned download URLs (15-minute TTL). Local files are cleaned up after 24 hours.

The Webhook system uses Encore Pub/Sub with a dedicated topic and subscription. On each call status change, a webhook_deliveries DB record is created and a message is published. The subscription handler sends HTTP POST with HMAC-SHA256 signature (X-Signature header). Encore's built-in retry policy with configurable MaxRetries and exponential backoff handles retries. Messages exceeding MaxRetries enter the dead letter queue. Admin APIs expose DLQ viewing and manual retry; client APIs allow webhook_url configuration and delivery log viewing.

**Primary recommendation:** Implement the real FSClient as a thin wrapper over eslgo `*Conn`, with a FSClientManager managing two connections (primary + standby). Use Encore Pub/Sub for both recording merge tasks and webhook delivery. Use Encore Object Storage for recording files with `SignedDownloadURL` for playback/download.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- 采用 Inbound 模式连接 FreeSWITCH ESL 端口（8021），应用主动连接 FS
- 断线后自动重连，指数退避（1s、2s、4s... 最大 30s），重连期间标记实例不健康，Pick() 自动跳过
- ESL 事件采用事件派发器模式驱动状态机：事件处理器解析 ESL 事件后通过回调/channel 通知状态机，状态机不直接依赖 ESL，保持与 Mock 相同的接口
- 只订阅业务相关事件：CHANNEL_ANSWER、CHANNEL_BRIDGE、CHANNEL_HANGUP，减少噪音
- 通话 UUID 由应用端预生成，originate 时传给 FS，ESL 事件通过 UUID 直接关联到 callback_calls 记录
- FSClient 接口不变，Phase 3 提供基于 eslgo 的真实实现替换 Phase 2 的 Mock，通过配置切换。Mock 继续用于单元测试
- 孤儿事件（找不到对应呼叫记录的 ESL 事件）记录警告日志（含 UUID 和事件详情），不崩溃不阻塞
- 录音在 CHANNEL_BRIDGE 后开始（已决策，PROJECT.md）
- A/B 路分轨录音（WAV），通话结束后异步 ffmpeg 合并为双声道 MP3
- ffmpeg 合并任务通过 Encore Pub/Sub 消息队列异步触发：通话结束时发布录音合并消息，独立 Worker 消费处理。可重试、可观测、不影响主流程
- S3 存储路径：`recordings/{customer_id}/{YYYY-MM-DD}/{call_id}_merged.mp3`（分轨文件类似命名 `_a.wav`、`_b.wav`）
- 用户通过 S3 预签名 URL 播放和下载录音（有效期 15 分钟），减轻后端带宽压力
- 录音文件保留 90 天，通过 S3 生命周期策略自动删除
- 本地录音文件 24h 后清理（REC-03 已定义）
- 呼叫状态变更时创建 webhook_deliveries 记录（独立表，已决策，PROJECT.md）
- Webhook Worker 指数退避重试 5 次，间隔 30s/1m/5m/15m/1h，超过最大次数进 DLQ
- Webhook 请求使用 HMAC-SHA256 签名验证：每个客户生成 webhook_secret，用 HMAC-SHA256 对 payload 签名，签名放在 X-Signature header 中
- Webhook payload 包含完整呼叫详情：event_type + call_id + 当前状态 + A/B 路详情 + custom_data + timestamp
- 全部状态变更触发 Webhook 推送：initiating、a_dialing、a_connected、b_dialing、bridged、finished、failed
- 管理员可查看 DLQ 并手动重试（HOOK-03）
- 客户可配置默认 webhook_url 并查看最近发送记录（HOOK-04）
- 采用主备模式：新通话优先走主 FS，主故障时自动切换到备 FS
- 健康探测每 10s 发送 status 命令，连续 3 次失败标记不健康（FS-03 已定义）
- 主 FS 恢复健康后不自动回切，避免频繁切换引起不稳定
- 主 FS 故障时，所有在途通话标记为 failed，触发 finalizeCall（退款 + 释放并发）。新通话路由到备 FS
- 开发环境 docker-compose 默认启动单台 FS，双机可选启动

### Claude's Discretion
- eslgo 连接的具体配置参数（超时、buffer 大小等）
- 事件派发器的具体实现方式（channel vs callback）
- 录音文件的临时存储路径和清理机制
- ffmpeg 合并的具体命令参数
- Webhook Worker 的具体实现（Encore Pub/Sub subscription 配置）
- FSClient Manager 的内部实现细节
- docker-compose 中 FreeSWITCH 的具体 SIP 配置

### Deferred Ideas (OUT OF SCOPE)
无
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FS-01 | FSClient 封装 eslgo，仅暴露 5 个方法 | eslgo v1.5.0 Conn API maps to: OriginateCall -> OriginateALeg/OriginateBLegAndBridge, HangupCall -> HangupCall, SendCommand -> StartRecording/StopRecording (uuid_record) |
| FS-02 | ESL 事件处理器正确响应 CHANNEL_ANSWER/CHANNEL_BRIDGE/CHANNEL_HANGUP | eslgo RegisterEventListener by UUID + EventListenAll; event.GetName() and event.GetHeader() for parsing |
| FS-03 | 健康探测每 10s 发送 status，连续 3 次失败标记不健康 | eslgo SendCommand with "status" API command; timer goroutine per instance |
| FS-04 | FreeSWITCH 双机热备，FSClient Manager 维护双连接，Pick() 跳过不健康实例 | FSClientManager struct with two ESLFSClient instances + health state + mutex |
| FS-05 | 新通话在任一 FS 实例故障时 < 5s 恢复到健康实例 | Health probe every 10s + 3 consecutive failures = 30s detection; but reconnect + instant failover on connection loss via onDisconnect callback achieves < 5s |
| REC-01 | 录音在 CHANNEL_BRIDGE 后开始，A/B 路时间戳对齐 | State machine triggers StartRecording after CHANNEL_BRIDGE event; both uuid_record commands issued in sequence |
| REC-02 | A/B 分轨 WAV，异步 ffmpeg 合并为双声道 MP3 | uuid_record per UUID writes separate WAV; ffmpeg amerge filter merges to stereo MP3 |
| REC-03 | 合并后上传 S3，本地 24h 清理 | Encore Object Storage Upload; cron job or cleanup goroutine for local files |
| REC-04 | 用户可在线播放和下载录音 | Encore Object Storage SignedDownloadURL with 15-min TTL |
| HOOK-01 | 状态变更创建 webhook_deliveries 记录 | New migration for webhook_deliveries table; INSERT on each status change in state machine |
| HOOK-02 | Worker 非阻塞延迟队列重试，指数退避，超限进 DLQ | Encore Pub/Sub subscription with RetryPolicy (MinBackoff, MaxBackoff, MaxRetries=5) |
| HOOK-03 | 管理员可查看 DLQ、手动重试 | Admin API endpoints reading webhook_deliveries WHERE status='dlq'; manual retry republishes message |
| HOOK-04 | 客户可配置 webhook_url、查看最近发送记录 | Client API endpoints for updating users.webhook_url + listing own webhook_deliveries |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| encore.dev | v1.52.1 | Framework (service, API, DB, cache, pubsub, objects) | Project constraint |
| encore.dev/pubsub | (bundled) | Recording merge queue + webhook delivery queue | Native async messaging with retry + DLQ |
| encore.dev/storage/objects | (bundled) | S3-compatible recording storage | Native object storage with presigned URLs |
| encore.dev/cron | (bundled) | Local file cleanup scheduling | Native cron for periodic tasks |
| github.com/percipia/eslgo | v1.5.0 | FreeSWITCH ESL inbound client | Production-tested Go ESL library, idiomatic API |
| crypto/hmac + crypto/sha256 (stdlib) | Go 1.25 | Webhook HMAC-SHA256 signing | Standard library, no external dependency |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| os/exec (stdlib) | Go 1.25 | Invoke ffmpeg for recording merge | Post-call recording merge worker |
| net/http (stdlib) | Go 1.25 | Webhook HTTP POST delivery | Webhook worker sending notifications |
| encoding/hex (stdlib) | Go 1.25 | HMAC signature hex encoding | Webhook signature header |
| time (stdlib) | Go 1.25 | Health probe timer, reconnect backoff | FSClient Manager internals |
| sync (stdlib) | Go 1.25 | Mutex for health state, connection management | FSClient Manager concurrency |
| context (stdlib) | Go 1.25 | Cancellation for ESL operations | All ESL calls |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| eslgo | vma/esl or 0x19/goesl | eslgo is most actively maintained (Dec 2025), production-tested for thousands of calls/sec |
| os/exec ffmpeg | Go audio library | ffmpeg is the standard for audio processing; Go audio libs lack stereo merge capability |
| Encore Pub/Sub for webhooks | Custom goroutine pool | Pub/Sub provides built-in retry, DLQ, observability; hand-rolled loses all of that |
| Encore Object Storage | Raw S3 SDK | Encore provides presigned URLs, auto-provisioning, testing support natively |

**Installation:**
```bash
go get github.com/percipia/eslgo@v1.5.0
```

## Architecture Patterns

### Recommended Project Structure
```
callback/
  fsclient/
    interface.go           # FSClient interface (from Phase 2, unchanged)
    types.go               # CallEvent, OriginateParams (from Phase 2)
    mock.go                # MockFSClient (from Phase 2, unchanged)
    esl.go                 # ESLFSClient - real eslgo implementation [NEW]
    manager.go             # FSClientManager - HA with health probes [NEW]
    manager_test.go        # Manager health/failover tests [NEW]
  recording/               # Sub-package for recording pipeline
    merge.go               # ffmpeg merge logic
    merge_test.go          # Merge unit tests
  statemachine.go          # Updated: add recording + webhook triggers
  finalize.go              # Updated: add recording merge publish + webhook publish
webhook/                   # WEBHOOK service [NEW]
  webhook.go               # Service struct + init
  deliver.go               # Webhook delivery worker (Pub/Sub subscription handler)
  dlq.go                   # DLQ admin APIs
  client.go                # Client webhook config + delivery log APIs
  types.go                 # WebhookEvent, WebhookDelivery types
  migrations/
    1_create_webhook_deliveries.up.sql
recording/                 # RECORDING service [NEW]
  recording.go             # Service struct + init
  worker.go                # Recording merge worker (Pub/Sub subscription handler)
  api.go                   # Recording URL APIs (presigned download)
  types.go                 # RecordingEvent types
docker-compose.dev.yml     # Updated: add FreeSWITCH container(s)
```

### Pattern 1: ESLFSClient Wrapping eslgo
**What:** A thin adapter implementing the FSClient interface from Phase 2. Uses eslgo.Conn for all FreeSWITCH operations. Translates between FSClient method signatures and eslgo API calls.
**When to use:** Production runtime (switched via config; mock used for tests).
**Example:**
```go
// callback/fsclient/esl.go
package fsclient

import (
    "context"
    "fmt"
    "github.com/percipia/eslgo"
    eslcommand "github.com/percipia/eslgo/command"
)

type ESLFSClient struct {
    conn     *eslgo.Conn
    handlers map[string][]func(CallEvent)
    mu       sync.RWMutex
}

func NewESLFSClient(address, password string) (*ESLFSClient, error) {
    client := &ESLFSClient{
        handlers: make(map[string][]func(CallEvent)),
    }
    conn, err := eslgo.Dial(address, password, func() {
        // onDisconnect callback - triggers reconnection in Manager
    })
    if err != nil {
        return nil, fmt.Errorf("ESL dial failed: %w", err)
    }
    client.conn = conn

    // Subscribe only to business events
    // Use SendCommand to execute: "event plain CHANNEL_ANSWER CHANNEL_BRIDGE CHANNEL_HANGUP"
    // Register global event listener to dispatch to handlers
    conn.RegisterEventListener(eslgo.EventListenAll, func(event *eslgo.Event) {
        client.dispatchEvent(event)
    })

    return client, nil
}

func (c *ESLFSClient) OriginateALeg(ctx context.Context, params OriginateParams) (string, error) {
    // Pre-generate UUID for tracking
    // originate {origination_uuid=<params.CallID>}sofia/gateway/<gateway>/<number> &park()
    // The &park() application parks the channel waiting for bridge
    vars := map[string]string{
        "origination_uuid":    params.CallID + "-a",
        "origination_caller_id_number": params.CallerID,
    }
    resp, err := c.conn.OriginateCall(ctx, true, // background=true for async
        eslgo.Leg{CallURL: fmt.Sprintf("sofia/gateway/%s/%s", params.GatewayIP, params.Number)},
        eslgo.Leg{CallURL: "&park()"},
        vars,
    )
    if err != nil {
        return "", fmt.Errorf("originate A-leg failed: %w", err)
    }
    if !resp.IsOk() {
        return "", fmt.Errorf("originate A-leg rejected: %s", resp.GetReply())
    }
    return params.CallID + "-a", nil
}

func (c *ESLFSClient) OriginateBLegAndBridge(ctx context.Context, aUUID string, params OriginateParams) (string, error) {
    // uuid_bridge <a_uuid> <b_dial_string>
    // Or: originate {origination_uuid=<uuid>}sofia/gateway/<gw>/<num> &bridge(<a_uuid>)
    bUUID := params.CallID + "-b"
    vars := map[string]string{
        "origination_uuid": bUUID,
    }
    resp, err := c.conn.OriginateCall(ctx, true,
        eslgo.Leg{CallURL: fmt.Sprintf("sofia/gateway/%s/%s", params.GatewayIP, params.Number)},
        eslgo.Leg{CallURL: fmt.Sprintf("&bridge(%s)", aUUID)},
        vars,
    )
    if err != nil {
        return "", fmt.Errorf("originate B-leg failed: %w", err)
    }
    if !resp.IsOk() {
        return "", fmt.Errorf("originate B-leg rejected: %s", resp.GetReply())
    }
    return bUUID, nil
}

func (c *ESLFSClient) StartRecording(ctx context.Context, uuid string, callID string, leg string) error {
    // uuid_record <uuid> start <path>
    path := fmt.Sprintf("/var/lib/freeswitch/recordings/%s_%s.wav", callID, leg)
    resp, err := c.conn.SendCommand(ctx, &RawAPICommand{
        Command: fmt.Sprintf("uuid_record %s start %s", uuid, path),
    })
    if err != nil {
        return fmt.Errorf("start recording failed: %w", err)
    }
    if !resp.IsOk() {
        return fmt.Errorf("start recording rejected: %s", resp.GetReply())
    }
    return nil
}

func (c *ESLFSClient) StopRecording(ctx context.Context, uuid string, callID string, leg string) error {
    path := fmt.Sprintf("/var/lib/freeswitch/recordings/%s_%s.wav", callID, leg)
    resp, err := c.conn.SendCommand(ctx, &RawAPICommand{
        Command: fmt.Sprintf("uuid_record %s stop %s", uuid, path),
    })
    if err != nil {
        return err
    }
    _ = resp
    return nil
}

func (c *ESLFSClient) HangupCall(ctx context.Context, uuid string, cause string) error {
    return c.conn.HangupCall(ctx, uuid, cause)
}

// RawAPICommand implements eslgo command.Command for raw API commands
type RawAPICommand struct {
    Command string
}

func (c *RawAPICommand) BuildMessage() string {
    return fmt.Sprintf("api %s", c.Command)
}

func (c *ESLFSClient) dispatchEvent(event *eslgo.Event) {
    eventName := event.GetName()
    // Only process subscribed events
    if eventName != "CHANNEL_ANSWER" && eventName != "CHANNEL_BRIDGE" && eventName != "CHANNEL_HANGUP" {
        return
    }

    uuid := event.GetHeader("Unique-ID")
    callID := event.GetHeader("variable_origination_uuid")
    leg := "A"
    if strings.HasSuffix(uuid, "-b") {
        leg = "B"
    }
    hangupCause := ""
    if eventName == "CHANNEL_HANGUP" {
        hangupCause = event.GetHeader("Hangup-Cause")
    }

    ce := CallEvent{
        CallID:      callID,
        UUID:        uuid,
        Leg:         leg,
        EventName:   eventName,
        HangupCause: hangupCause,
        Timestamp:   time.Now(),
    }

    c.mu.RLock()
    handlers := c.handlers[eventName]
    c.mu.RUnlock()
    for _, h := range handlers {
        h(ce)
    }
}
```

### Pattern 2: FSClientManager for High Availability
**What:** Manages two ESLFSClient instances (primary + standby). Runs health probes every 10 seconds. Pick() returns the current healthy instance for new calls.
**When to use:** Always in production. Dev environment can run with single instance.
**Example:**
```go
// callback/fsclient/manager.go
type FSClientManager struct {
    instances []*managedInstance
    mu        sync.RWMutex
    activeIdx int // index of currently active (primary) instance
}

type managedInstance struct {
    client      *ESLFSClient
    address     string
    password    string
    healthy     bool
    failCount   int
    reconnecting bool
}

func (m *FSClientManager) Pick() (FSClient, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    // Try active instance first
    if m.instances[m.activeIdx].healthy {
        return m.instances[m.activeIdx].client, nil
    }
    // Failover to other instance
    other := 1 - m.activeIdx
    if m.instances[other].healthy {
        return m.instances[other].client, nil
    }
    return nil, fmt.Errorf("no healthy FreeSWITCH instance available")
}

func (m *FSClientManager) startHealthProbe(idx int) {
    ticker := time.NewTicker(10 * time.Second)
    for range ticker.C {
        inst := m.instances[idx]
        // Send "status" command via ESL
        resp, err := inst.client.conn.SendCommand(context.Background(), &RawAPICommand{Command: "status"})
        if err != nil || !resp.IsOk() {
            inst.failCount++
            if inst.failCount >= 3 {
                m.mu.Lock()
                inst.healthy = false
                if idx == m.activeIdx {
                    // Fail all in-flight calls on this instance
                    m.failInFlightCalls(idx)
                    // Switch active to other instance if healthy
                    other := 1 - idx
                    if m.instances[other].healthy {
                        m.activeIdx = other
                    }
                }
                m.mu.Unlock()
            }
        } else {
            m.mu.Lock()
            inst.failCount = 0
            inst.healthy = true
            // Do NOT auto-switch back to recovered instance (per CONTEXT.md)
            m.mu.Unlock()
        }
    }
}
```

### Pattern 3: Recording Merge via Pub/Sub Worker
**What:** After call finalization, publish a RecordingMergeEvent to a Pub/Sub topic. A subscription handler runs ffmpeg merge and uploads to S3.
**When to use:** Every call that has recordings (bridged calls).
**Example:**
```go
// recording/recording.go
var RecordingMergeTopic = pubsub.NewTopic[*RecordingMergeEvent]("recording-merge", pubsub.TopicConfig{
    DeliveryGuarantee: pubsub.AtLeastOnce,
})

type RecordingMergeEvent struct {
    CallID     string `json:"call_id"`
    CustomerID int64  `json:"customer_id"`
    AFilePath  string `json:"a_file_path"` // /var/lib/freeswitch/recordings/{call_id}_a.wav
    BFilePath  string `json:"b_file_path"`
    Date       string `json:"date"` // YYYY-MM-DD
}

// recording/worker.go
var _ = pubsub.NewSubscription(
    RecordingMergeTopic, "recording-merge-worker",
    pubsub.SubscriptionConfig[*RecordingMergeEvent]{
        Handler:   pubsub.MethodHandler((*Service).HandleMerge),
        RetryPolicy: &pubsub.RetryPolicy{
            MinBackoff: 30 * time.Second,
            MaxBackoff: 5 * time.Minute,
            MaxRetries: 3,
        },
    },
)

func (s *Service) HandleMerge(ctx context.Context, event *RecordingMergeEvent) error {
    // 1. Merge A + B WAV into stereo MP3 via ffmpeg
    outputPath := fmt.Sprintf("/tmp/%s_merged.mp3", event.CallID)
    cmd := exec.CommandContext(ctx, "ffmpeg",
        "-i", event.AFilePath,
        "-i", event.BFilePath,
        "-filter_complex", "[0:a][1:a]amerge=inputs=2[a]",
        "-map", "[a]",
        "-ac", "2",
        "-codec:a", "libmp3lame",
        "-q:a", "2",
        outputPath,
    )
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("ffmpeg merge failed: %w", err)
    }

    // 2. Upload merged file to S3
    s3Key := fmt.Sprintf("recordings/%d/%s/%s_merged.mp3", event.CustomerID, event.Date, event.CallID)
    w, err := s.bucket.Upload(ctx, s3Key, nil)
    if err != nil {
        return err
    }
    f, _ := os.Open(outputPath)
    defer f.Close()
    io.Copy(w, f)
    if err := w.Close(); err != nil {
        return fmt.Errorf("S3 upload failed: %w", err)
    }

    // 3. Also upload individual legs
    // Upload A and B WAV files with _a.wav / _b.wav suffixes

    // 4. Update callback_calls with recording URLs
    // UPDATE callback_calls SET recording_key = s3Key WHERE call_id = event.CallID

    // 5. Clean up local merged file (originals cleaned by 24h cron)
    os.Remove(outputPath)

    return nil
}
```

### Pattern 4: Webhook Delivery via Pub/Sub with HMAC Signing
**What:** Publish webhook events on status changes. Subscription handler delivers with HMAC-SHA256 signature and exponential backoff retry.
**When to use:** Every call status change for customers with webhook_url configured.
**Example:**
```go
// webhook/webhook.go
var WebhookTopic = pubsub.NewTopic[*WebhookEvent]("webhook-delivery", pubsub.TopicConfig{
    DeliveryGuarantee: pubsub.AtLeastOnce,
})

type WebhookEvent struct {
    DeliveryID int64           `json:"delivery_id"`
    WebhookURL string          `json:"webhook_url"`
    Secret     string          `json:"secret"`
    Payload    json.RawMessage `json:"payload"`
}

// webhook/deliver.go
var _ = pubsub.NewSubscription(
    WebhookTopic, "webhook-delivery-worker",
    pubsub.SubscriptionConfig[*WebhookEvent]{
        Handler: pubsub.MethodHandler((*Service).DeliverWebhook),
        RetryPolicy: &pubsub.RetryPolicy{
            MinBackoff: 30 * time.Second,
            MaxBackoff: 1 * time.Hour,
            MaxRetries: 5,
        },
    },
)

func (s *Service) DeliverWebhook(ctx context.Context, event *WebhookEvent) error {
    // 1. Compute HMAC-SHA256 signature
    mac := hmac.New(sha256.New, []byte(event.Secret))
    mac.Write(event.Payload)
    signature := hex.EncodeToString(mac.Sum(nil))

    // 2. Send HTTP POST
    req, _ := http.NewRequestWithContext(ctx, "POST", event.WebhookURL, bytes.NewReader(event.Payload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Signature", signature)
    req.Header.Set("X-Delivery-ID", fmt.Sprintf("%d", event.DeliveryID))

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        // Update delivery status to "retrying"
        s.updateDeliveryStatus(ctx, event.DeliveryID, "retrying", err.Error())
        return err // triggers Pub/Sub retry
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        s.updateDeliveryStatus(ctx, event.DeliveryID, "delivered", "")
        return nil
    }

    // Non-2xx = failure, trigger retry
    body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
    errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
    s.updateDeliveryStatus(ctx, event.DeliveryID, "retrying", errMsg)
    return fmt.Errorf("webhook delivery failed: %s", errMsg)
}
```

### Pattern 5: Webhook Payload Structure
**What:** Standardized webhook payload with complete call details.
**Example:**
```go
type WebhookPayload struct {
    EventType string          `json:"event_type"` // "status_changed"
    CallID    string          `json:"call_id"`
    Status    string          `json:"status"`
    ALeg      *LegDetail      `json:"a_leg"`
    BLeg      *LegDetail      `json:"b_leg,omitempty"`
    Bridge    *BridgeDetail   `json:"bridge,omitempty"`
    CustomData json.RawMessage `json:"custom_data,omitempty"`
    Timestamp  time.Time       `json:"timestamp"`
}

type LegDetail struct {
    Number      string     `json:"number"`
    Status      string     `json:"status"`
    DialAt      *time.Time `json:"dial_at,omitempty"`
    AnswerAt    *time.Time `json:"answer_at,omitempty"`
    HangupAt    *time.Time `json:"hangup_at,omitempty"`
    HangupCause string     `json:"hangup_cause,omitempty"`
}
```

### Anti-Patterns to Avoid
- **Direct eslgo usage outside ESLFSClient:** All FreeSWITCH interaction must go through the FSClient interface. Never import eslgo in the callback service main package.
- **Synchronous recording merge:** Never wait for ffmpeg in the call finalization path. Always publish to Pub/Sub and return immediately.
- **Synchronous webhook delivery:** Never send webhooks in the state machine goroutine. Always publish to Pub/Sub.
- **Polling for recordings:** Do not poll S3 for recording availability. The merge worker updates the DB record when complete.
- **Storing webhook_secret in plaintext in logs:** The secret must not appear in any log output. The WebhookEvent struct carries it for the worker, but logging should redact it.
- **Re-implementing Pub/Sub retry logic:** Encore Pub/Sub handles retry + DLQ natively. Do not build a custom retry mechanism on top.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ESL connection management | Raw TCP socket + protocol parser | eslgo v1.5.0 `Dial()` | ESL protocol is complex (auth, events, async replies); eslgo is production-tested |
| Audio merge | Go audio library | ffmpeg via os/exec | ffmpeg handles codec conversion, stereo merge, MP3 encoding reliably |
| Object storage + presigned URLs | Raw S3 SDK | Encore Object Storage | Auto-provisioned, presigned URLs, testing support built-in |
| Async task queue with retry | Custom goroutine pool + DB queue | Encore Pub/Sub | Built-in retry policy, DLQ, observability, cloud-native |
| HMAC signing | Custom hash scheme | crypto/hmac + crypto/sha256 | Stdlib is correct and timing-safe with hmac.Equal |
| Exponential backoff | Custom backoff logic | Encore Pub/Sub RetryPolicy | MinBackoff/MaxBackoff/MaxRetries configured declaratively |
| Webhook delivery tracking | Custom state machine | webhook_deliveries table + Pub/Sub status | Simple DB record + message-driven updates |

**Key insight:** Phase 3 adds three infrastructure concerns (ESL, recordings, webhooks) but Encore's primitives (Pub/Sub, Object Storage, Cron) handle the heavy lifting. The custom code is primarily thin adapters and business logic.

## Common Pitfalls

### Pitfall 1: ESL Connection Loss During Active Calls
**What goes wrong:** FreeSWITCH disconnects (network issue, restart). Active calls on that instance have no event delivery. State machines hang waiting for events.
**Why it happens:** ESL is a persistent TCP connection; network issues cause silent failures.
**How to avoid:** eslgo's `onDisconnect` callback fires immediately. On disconnect: (1) mark instance unhealthy, (2) iterate all active calls on that instance and finalize them as "failed" with reason "fs_connection_lost", (3) start reconnection goroutine with exponential backoff. State machine goroutines must have context cancellation tied to instance health.
**Warning signs:** Growing goroutine count, calls stuck in intermediate states.

### Pitfall 2: UUID Collision Between Application and FreeSWITCH
**What goes wrong:** Application pre-generates UUID but FreeSWITCH assigns a different one, causing event mismatches.
**Why it happens:** Not setting `origination_uuid` channel variable correctly.
**How to avoid:** Always set `origination_uuid` in the originate command variables map. Verify in ESL event that `variable_origination_uuid` matches. Use the pattern `{call_id}-a` and `{call_id}-b` for A/B leg UUIDs to enable easy parsing.
**Warning signs:** Orphan events (UUID not found in active calls map).

### Pitfall 3: ffmpeg Hanging on Missing Input File
**What goes wrong:** ffmpeg blocks indefinitely waiting for input if the WAV file was never created (recording failed silently).
**Why it happens:** FreeSWITCH may fail to create the recording file but not report an error via ESL.
**How to avoid:** Before invoking ffmpeg, check that both input files exist and have non-zero size. If a file is missing, skip merge and mark the recording as "partial" or "failed" in the DB. Use `context.WithTimeout` on exec.CommandContext (e.g., 5-minute timeout).
**Warning signs:** Merge worker stuck, Pub/Sub retries piling up.

### Pitfall 4: Webhook Secret Rotation During Active Deliveries
**What goes wrong:** Customer rotates webhook_secret while deliveries are in the Pub/Sub queue. Queued messages have the old secret.
**Why it happens:** Secret is snapshot at publish time.
**How to avoid:** Store the secret in the WebhookEvent message at publish time. The webhook is signed with the secret that was active when the event was created. Document this behavior for customers -- signature verification uses the secret at event-creation time.
**Warning signs:** Customer reports signature verification failures after rotating secret.

### Pitfall 5: Recording File Path Mismatch Between FS Host and Application
**What goes wrong:** Application tells FS to record to path X, but the recording merge worker runs on a different host/container and cannot access path X.
**Why it happens:** FreeSWITCH records to its local filesystem. If FS runs in Docker, the path is inside the container.
**How to avoid:** Use Docker volume mounts to share the recording directory between FS container and the application container. In docker-compose, mount `/var/lib/freeswitch/recordings` as a shared volume. For production, consider FS writing directly to a shared filesystem or the application pulling files via SCP/NFS.
**Warning signs:** "File not found" errors in merge worker.

### Pitfall 6: Health Probe False Positive After Network Partition Heals
**What goes wrong:** Network partition heals, health probe succeeds, but in-flight calls were already finalized as failed. Duplicate finalization if FS sends late hangup events.
**Why it happens:** ESL events from before the partition may arrive after reconnection.
**How to avoid:** On reconnection, do NOT re-register event listeners for old call UUIDs. Only register listeners for new calls. The sync.Once protection in finalizeCall (from Phase 2) prevents double finalization at the DB level.
**Warning signs:** Duplicate refund transactions, negative concurrent slot counts.

## Code Examples

### webhook_deliveries Migration
```sql
-- webhook/migrations/1_create_webhook_deliveries.up.sql
CREATE TABLE webhook_deliveries (
    id BIGSERIAL PRIMARY KEY,
    call_id VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    event_type VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'delivering', 'delivered', 'retrying', 'failed', 'dlq'
    )),
    webhook_url TEXT NOT NULL,
    payload JSONB NOT NULL,

    -- Delivery tracking
    attempt_count INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    last_attempt_at TIMESTAMPTZ,
    last_error TEXT,
    delivered_at TIMESTAMPTZ,

    -- DLQ management
    dlq_at TIMESTAMPTZ,
    dlq_retry_count INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_user_created ON webhook_deliveries(user_id, created_at DESC);
CREATE INDEX idx_webhook_status ON webhook_deliveries(status) WHERE status IN ('pending', 'retrying', 'dlq');
CREATE INDEX idx_webhook_call ON webhook_deliveries(call_id);
```

### callback_calls Recording Columns Migration
```sql
-- callback/migrations/3_add_recording_columns.up.sql
ALTER TABLE callback_calls
    ADD COLUMN recording_key TEXT,           -- S3 object key for merged recording
    ADD COLUMN recording_a_key TEXT,         -- S3 key for A-leg WAV
    ADD COLUMN recording_b_key TEXT,         -- S3 key for B-leg WAV
    ADD COLUMN recording_status VARCHAR(20)  -- 'recording', 'merging', 'ready', 'failed'
        CHECK (recording_status IN ('recording', 'merging', 'ready', 'failed'));
```

### users Table Webhook Columns Migration
```sql
-- auth/migrations/3_add_webhook_columns.up.sql
ALTER TABLE users
    ADD COLUMN webhook_url TEXT,
    ADD COLUMN webhook_secret VARCHAR(128);
```

### Encore Object Storage Bucket Declaration
```go
// recording/recording.go
var RecordingsBucket = objects.NewBucket("recordings", objects.BucketConfig{
    Versioned: false,
})
```

### Presigned Download URL API
```go
// recording/api.go
type GetRecordingURLParams struct {
    CallID string `query:"call_id"`
    Type   string `query:"type"` // "merged", "a", "b"
}

type GetRecordingURLResponse struct {
    URL       string    `json:"url"`
    ExpiresAt time.Time `json:"expires_at"`
}

//encore:api auth method=GET path=/recordings/url
func (s *Service) GetRecordingURL(ctx context.Context, p *GetRecordingURLParams) (*GetRecordingURLResponse, error) {
    // 1. Verify auth (client sees own, admin sees all)
    // 2. Look up recording_key from callback_calls
    // 3. Generate presigned URL
    ttl := 15 * time.Minute
    url, err := RecordingsBucket.SignedDownloadURL(ctx, recordingKey, objects.WithTTL(ttl))
    if err != nil {
        return nil, err
    }
    return &GetRecordingURLResponse{
        URL:       url,
        ExpiresAt: time.Now().Add(ttl),
    }, nil
}
```

### HMAC-SHA256 Webhook Signing
```go
// webhook/sign.go
func signPayload(secret string, payload []byte) string {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    return hex.EncodeToString(mac.Sum(nil))
}

func verifySignature(secret string, payload []byte, signature string) bool {
    expected := signPayload(secret, payload)
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

### FreeSWITCH Docker Compose Configuration
```yaml
# docker-compose.dev.yml (updated for Phase 3)
services:
  freeswitch-primary:
    image: signalwire/freeswitch:1.10
    container_name: bos3000-fs-primary
    ports:
      - "8021:8021"     # ESL
      - "5080:5080/udp" # SIP
      - "5080:5080/tcp"
    volumes:
      - ./docker/freeswitch/conf:/usr/local/freeswitch/etc/freeswitch
      - fs-recordings:/var/lib/freeswitch/recordings
    environment:
      - ESL_PASSWORD=ClueCon
    networks:
      - bos3000

  freeswitch-standby:
    image: signalwire/freeswitch:1.10
    container_name: bos3000-fs-standby
    ports:
      - "8022:8021"     # ESL (different host port)
      - "5081:5080/udp"
    volumes:
      - ./docker/freeswitch/conf:/usr/local/freeswitch/etc/freeswitch
      - fs-recordings:/var/lib/freeswitch/recordings
    environment:
      - ESL_PASSWORD=ClueCon
    profiles:
      - ha  # Only started with: docker compose --profile ha up
    networks:
      - bos3000

volumes:
  fs-recordings:  # Shared volume for recording files

networks:
  bos3000:
```

### ESL Event Socket Configuration
```xml
<!-- docker/freeswitch/conf/autoload_configs/event_socket.conf.xml -->
<configuration name="event_socket.conf" description="Socket Client">
  <settings>
    <param name="nat-map" value="false"/>
    <param name="listen-ip" value="::"/>
    <param name="listen-port" value="8021"/>
    <param name="password" value="ClueCon"/>
  </settings>
</configuration>
```

### ffmpeg Merge Command
```bash
# Merge two mono WAV files into stereo MP3
# A-leg on left channel, B-leg on right channel
ffmpeg -y \
  -i /var/lib/freeswitch/recordings/{call_id}_a.wav \
  -i /var/lib/freeswitch/recordings/{call_id}_b.wav \
  -filter_complex "[0:a][1:a]amerge=inputs=2[a]" \
  -map "[a]" \
  -ac 2 \
  -codec:a libmp3lame \
  -q:a 2 \
  /tmp/{call_id}_merged.mp3
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Mock FSClient (Phase 2) | Real eslgo ESLFSClient | Phase 3 | Production FreeSWITCH integration |
| StartRecording/StopRecording no-ops | Real uuid_record commands via ESL | Phase 3 | Actual call recording |
| No webhooks | Encore Pub/Sub driven webhook delivery | Phase 3 | Async customer notifications |
| No recording storage | Encore Object Storage with presigned URLs | Phase 3 | S3-compatible recording persistence |
| Single FS (dev) | Primary/Standby FS with health probes | Phase 3 | High availability for production |

## Open Questions

1. **eslgo OriginateCall with park application**
   - What we know: eslgo's `OriginateCall()` takes two `Leg` objects. The second leg can be `&park()` to park the A-leg.
   - What's unclear: Exact dial string format for sofia gateway with origination_uuid variable. Whether eslgo handles the channel variable injection correctly in the `vars` map parameter.
   - Recommendation: Verify during implementation with a real FS instance. The docker-compose setup enables quick iteration. Fallback: use `SendCommand` with raw `originate` API string if OriginateCall doesn't support all variables.

2. **Recording file access between containers**
   - What we know: FS writes recordings to its local filesystem. The merge worker needs access to these files.
   - What's unclear: In production (non-Docker), how recordings flow from FS to the merge worker.
   - Recommendation: Use Docker shared volume for dev. For production, document that FS must write to a shared filesystem (NFS, EFS) or the application should SCP files. This is a deployment concern, not a code concern.

3. **Encore Pub/Sub DLQ access API**
   - What we know: Encore Pub/Sub moves messages to DLQ after MaxRetries. The admin needs to view and retry DLQ messages.
   - What's unclear: Whether Encore provides a programmatic API to read from the DLQ, or if DLQ access is only via Encore Cloud dashboard.
   - Recommendation: Track DLQ state in the webhook_deliveries DB table (status='dlq'). When Pub/Sub exhausts retries, the handler's final failure updates the DB record to 'dlq'. Manual retry reads from DB and re-publishes to the topic. This gives full control without depending on Encore's DLQ API.

4. **eslgo reconnection handling**
   - What we know: eslgo provides an `onDisconnect` callback in `Dial()`. No built-in reconnection.
   - What's unclear: Whether a new `Dial()` call after disconnect cleanly replaces the old connection.
   - Recommendation: On disconnect, create a new `ESLFSClient` instance entirely. The Manager tracks the instance reference and swaps it atomically. Old instance's event listeners are abandoned (orphan events are safely ignored per CONTEXT.md).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + encore test |
| Config file | none -- Encore uses standard Go test conventions |
| Quick run command | `encore test ./callback/fsclient/... ./webhook/... ./recording/... -v` |
| Full suite command | `encore test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FS-01 | ESLFSClient implements FSClient interface (compile check) | unit | `encore test ./callback/fsclient/... -run TestESLFSClientInterface -x` | No -- Wave 0 |
| FS-02 | Event dispatcher routes CHANNEL_ANSWER/BRIDGE/HANGUP to handlers | unit | `encore test ./callback/fsclient/... -run TestEventDispatch -x` | No -- Wave 0 |
| FS-03 | Health probe marks unhealthy after 3 failures | unit | `encore test ./callback/fsclient/... -run TestHealthProbe -x` | No -- Wave 0 |
| FS-04 | Manager.Pick() skips unhealthy, fails over to standby | unit | `encore test ./callback/fsclient/... -run TestManagerFailover -x` | No -- Wave 0 |
| FS-05 | New call routes to healthy instance within 5s of failure | integration | `encore test ./callback/fsclient/... -run TestFailoverTiming -x` | No -- Wave 0 |
| REC-01 | StartRecording called after CHANNEL_BRIDGE event | unit | `encore test ./callback/... -run TestRecordingStartOnBridge -x` | No -- Wave 0 |
| REC-02 | ffmpeg merge produces stereo MP3 from two mono WAVs | unit | `encore test ./recording/... -run TestFFmpegMerge -x` | No -- Wave 0 |
| REC-03 | Merged file uploaded to S3, local cleanup after 24h | integration | `encore test ./recording/... -run TestUploadAndCleanup -x` | No -- Wave 0 |
| REC-04 | Presigned download URL generated with 15-min TTL | unit | `encore test ./recording/... -run TestPresignedURL -x` | No -- Wave 0 |
| HOOK-01 | Status change creates webhook_deliveries record | unit | `encore test ./webhook/... -run TestDeliveryCreation -x` | No -- Wave 0 |
| HOOK-02 | Webhook worker retries on failure, moves to DLQ after 5 | unit | `encore test ./webhook/... -run TestRetryAndDLQ -x` | No -- Wave 0 |
| HOOK-03 | Admin can view DLQ and manually retry | unit | `encore test ./webhook/... -run TestAdminDLQ -x` | No -- Wave 0 |
| HOOK-04 | Client can set webhook_url and view delivery log | unit | `encore test ./webhook/... -run TestClientWebhookConfig -x` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `encore test ./callback/fsclient/... ./webhook/... ./recording/...`
- **Per wave merge:** `encore test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `callback/fsclient/esl_test.go` -- covers FS-01, FS-02 (mock ESL conn for unit tests)
- [ ] `callback/fsclient/manager_test.go` -- covers FS-03, FS-04, FS-05
- [ ] `callback/recording/merge_test.go` -- covers REC-02 (requires ffmpeg in test env)
- [ ] `recording/recording_test.go` -- covers REC-03, REC-04
- [ ] `webhook/webhook_test.go` -- covers HOOK-01, HOOK-02, HOOK-03, HOOK-04
- [ ] `webhook/migrations/1_create_webhook_deliveries.up.sql` -- webhook delivery table
- [ ] `callback/migrations/3_add_recording_columns.up.sql` -- recording columns on callback_calls
- [ ] `auth/migrations/3_add_webhook_columns.up.sql` -- webhook_url/secret on users
- [ ] `docker-compose.dev.yml` -- FreeSWITCH container with ESL enabled
- [ ] `docker/freeswitch/conf/` -- Minimal FS config with event_socket enabled
- [ ] ffmpeg must be available in test/dev environment

## Sources

### Primary (HIGH confidence)
- [eslgo v1.5.0 pkg.go.dev](https://pkg.go.dev/github.com/percipia/eslgo) -- Conn API, Dial, OriginateCall, RegisterEventListener, HangupCall, SendCommand
- [eslgo GitHub README](https://github.com/percipia/eslgo) -- Version info, usage examples, Leg type
- [Encore Object Storage docs](https://encore.dev/docs/go/primitives/object-storage) -- NewBucket, Upload, SignedDownloadURL, BucketRef permissions
- [Encore Pub/Sub docs](https://encore.dev/docs/go/primitives/pubsub) -- NewTopic, NewSubscription, RetryPolicy, MaxRetries, DLQ behavior, MethodHandler
- [FreeSWITCH mod_commands](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Modules/mod_commands_1966741/) -- uuid_record, uuid_setvar syntax
- [FreeSWITCH record_session](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Modules/mod-dptools/6587110/) -- Recording application details
- [FreeSWITCH RECORD_STEREO](https://developer.signalwire.com/freeswitch/Channel-Variables-Catalog/RECORD_STEREO_16352883/) -- Per-channel stereo recording variable
- [FreeSWITCH origination_uuid](https://developer.signalwire.com/freeswitch/Channel-Variables-Catalog/origination_uuid_16353330/) -- Pre-assigning UUID to originated call

### Secondary (MEDIUM confidence)
- [HMAC webhook signature patterns](https://hookdeck.com/webhooks/guides/how-to-implement-sha256-webhook-signature-verification) -- Industry standard HMAC-SHA256 signing approach
- [ffmpeg amerge filter](https://forum.videohelp.com/threads/353278-What-is-the-ffmpeg-command-to-merge-two-mono-audio-streams-into-one-stereo) -- Merge two mono to stereo command syntax
- [FreeSWITCH Docker setup](https://medium.com/@er.anil.ec/installing-freeswitch-from-source-using-docker-and-docker-compose-ed2cc4fa992e) -- Docker compose patterns, volume mounts, port configuration

### Tertiary (LOW confidence)
- eslgo reconnection behavior -- No official documentation on reconnection; inferred from onDisconnect callback pattern
- FreeSWITCH uuid_record per-leg reliability -- Community reports mixed results; our approach (separate uuid_record per UUID) should be more reliable than record_session

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- eslgo is well-documented, Encore primitives are production-ready
- Architecture (ESLFSClient): HIGH -- eslgo API maps cleanly to FSClient interface from Phase 2
- Architecture (Recording pipeline): MEDIUM -- ffmpeg merge is well-understood, but FS recording path sharing between containers needs validation
- Architecture (Webhook): HIGH -- Encore Pub/Sub with retry/DLQ is a standard pattern
- Pitfalls: HIGH -- ESL disconnection, file path mismatch are well-known telecom system issues
- FS HA: MEDIUM -- Health probe pattern is solid, but real-world failover timing needs integration testing

**Research date:** 2026-03-10
**Valid until:** 2026-04-10 (stable ecosystem; eslgo v1.5.0 is latest stable)
