# Phase 2: 呼叫引擎（Mock）- Research

**Researched:** 2026-03-10
**Domain:** Call engine state machine, FSClient interface/mock, CDR records, wastage classification (Encore.go + Go)
**Confidence:** HIGH

## Summary

Phase 2 builds the core call engine on top of Phase 1's platform services (auth, billing, routing, compliance). The central challenge is implementing a robust A/B leg state machine that drives a callback call through its lifecycle (a_dialing -> a_connected -> b_dialing -> b_connected -> bridged -> finished/failed), handles all failure modes with correct cleanup (balance refund, slot release, wastage marking), and persists all state changes to a unified `callback_calls` table that serves as both real-time state tracker and CDR history.

The key architectural decision from CONTEXT.md is that the Mock FSClient operates in **instant simulation mode** -- events fire immediately without actual delays, making tests fast. The FSClient is defined as a Go interface with 5 methods (OriginateALeg, OriginateBLegAndBridge, HangupCall, StartRecording, StopRecording), and the mock implementation uses injected `MockConfig` to control behavior per test case. Phase 2 also covers the callback service's API layer: initiate callback, query status, force hangup, and CDR listing with filtering.

**Primary recommendation:** Create a single `callback` Encore service with the FSClient interface defined in a sub-package (`callback/fsclient/`). The state machine runs as a goroutine per call, coordinated via channels for event delivery. The mock FSClient immediately invokes registered event handlers, simulating the full ESL event flow. All 5 failure scenarios from CONTEXT.md must have dedicated test cases.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- A 路外呼失败（无人接听/拒接/号码错误）直接结束呼叫，不重试，标记 failed，退还预扣、释放并发，不计损耗（A 未接通无实际成本）
- B 路 originate 失败不尝试备用网关，直接结束。A 路已接通在等待，重试会延长等待时间增加损耗。容灾切换由 Phase 1 路由引擎在选择网关时处理
- "桥接秒断"阈值可配置，全局默认 10 秒，管理员可在 system_configs 中调整
- 强制挂断（管理员或客户）视为正常结束，按实际通话时长计费
- A 路 park 超过 60s 自动挂断（PRD 已定义）
- 发起回拨参数：a_number + b_number 必填，可选参数包括 caller_id（指定外显号）、max_duration（最大时长，秒）、custom_data（透传 JSON）、preferred_gateway（优先网关）、callback_url（回调 URL）
- custom_data 接受任意 JSON 对象，系统透明存储，在 CDR 和 Webhook 中原样返回，限制 1KB 大小
- max_duration 可选，不传则使用全局默认值（如 3600 秒），超时自动挂断
- 查询状态返回完整详情：当前状态、A/B 路分别的状态、外显号、桥接时长、损耗类型、费用信息，一次调用获取所有信息
- 仅提供单个回拨 API，不提供批量接口。批量场景由前端循环调用，后端通过并发槽位控制
- 强制挂断为统一接口，后端根据认证角色自动判断权限：管理员可挂任何通话，客户只能挂自己的
- 发起回拨响应只返回 call_id 和初始状态，不返回预估费用。预扣是内部逻辑，实际费用通过 CDR 查看
- A/B 路信息作为 callback_calls 表的字段嵌入（a_number, a_answer_time, a_hangup_cause, b_number, b_answer_time, b_hangup_cause 等），一条记录 = 一次回拨
- callback_calls 表同时承载实时状态和历史话单职责，通话结束后状态变为 finished/failed，自然成为 CDR，不需要独立的 cdr_records 表
- CDR 记录在回拨发起时创建（状态 initiating），后续状态变更通过 UPDATE 更新同一条记录。保证进行中的呼叫可追踪，不会因崩溃丢失
- 损耗信息嵌入 callback_calls 表：wastage_type（损耗类型）、wastage_cost（损耗成本）、wastage_duration（损耗时长），无损耗时为 NULL
- 所有时间戳存储到毫秒级（timestamptz），时长计算精确到毫秒，计费时按规则向上取整
- CDR 查询 API 支持完整筛选：时间范围、状态、A/B 号码、损耗类型、客户 ID 等，分页返回
- Mock 采用即时模拟方式，立即触发事件回调（ANSWER/BRIDGE/HANGUP），不实际等待。测试运行快
- 核心覆盖 5 种场景：① A 路拒接/无人接听 ② A 路接通但 B 路失败 ③ 桥接成功但秒断 ④ 正常完成 ⑤ A 路 park 超时
- Mock 通过注入配置方式控制行为（如 MockConfig{ALegResult: "answer", BLegResult: "reject"}），每个测试用例可独立配置不同场景
- Mock 支持交互式 API 测试：encore run 时 Mock 也能工作，开发者可用 curl/Postman 调 API 测试完整流程
- PRD 已定义 FSClient 对外仅暴露 5 个方法：OriginateALeg / OriginateBLegAndBridge / HangupCall / StartRecording / StopRecording（Phase 2 中 StartRecording/StopRecording 为空实现，Phase 3 补充）
- 状态序列固定：a_dialing -> a_connected -> b_dialing -> b_connected -> bridged -> finished/failed
- 并发槽位由 finalizeCall 统一释放，不使用 defer（避免 ESL 事件异步场景下双重释放）

### Claude's Discretion
- FSClient 接口的具体方法签名
- 状态机的内部实现方式（goroutine + channel 还是其他模式）
- Mock 的具体代码组织（文件结构、包划分）
- CDR 表的完整字段列表和索引设计
- 并发控制的具体实现细节

### Deferred Ideas (OUT OF SCOPE)
无
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CALL-01 | 用户可通过 API 发起 A/B 双外呼回拨，系统先呼 A 路，A 接通后呼 B 路并桥接 | FSClient interface with OriginateALeg + OriginateBLegAndBridge; state machine drives sequence; callback service API endpoint |
| CALL-02 | A 路外呼后进入 park 状态等待桥接，park 超时 60s 自动挂断防止通道泄漏 | Mock simulates park timeout via goroutine timer; state machine handles timeout as A-leg hangup with cleanup |
| CALL-03 | B 路 originate 命令级失败时，系统显式标记损耗、挂断 A 路、退款、释放并发 | handleBLegOriginateFailure pattern from PRD; Mock can return error from OriginateBLegAndBridge |
| CALL-04 | 呼叫状态机完整覆盖 a_dialing->a_connected->b_dialing->b_connected->bridged->finished/failed 全流程 | State machine with explicit transitions; callback_calls.status column tracks current state |
| CALL-05 | 用户可查询呼叫实时状态（含 A/B 路详情、桥接时长、损耗类型） | GetCallStatus API reads from callback_calls; returns all embedded A/B leg fields |
| CALL-06 | 用户可强制挂断进行中的通话（管理员全局 / 客户自有） | ForceHangup API with role-based permission check; calls FSClient.HangupCall |
| CALL-07 | 系统通过 Mock FSClient 支持全状态机单测（不依赖真实 FreeSWITCH） | FSClient as Go interface; MockFSClient with configurable behavior per scenario |
| WAST-01 | 系统自动分类损耗类型：A通B不通（a_connected_b_failed）、桥接秒断（bridge_broken_early） | Wastage classification in finalizeCall based on state + bridge duration vs threshold |
| WAST-02 | 损耗成本精确计算并记录到呼叫记录（wastage_cost 字段） | A-leg cost during B-leg failure = wastage_cost; bridge_broken_early uses actual A-leg billable time |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| encore.dev | v1.52.1 | Framework (service, API, DB, cache) | Project constraint, already in go.mod |
| encore.dev/storage/sqldb | (bundled) | callback_calls table management | Encore native DB with auto migrations |
| encore.dev/beta/errs | (bundled) | Structured error responses | Consistent error codes across services |
| encore.dev/beta/auth | (bundled) | Auth context for role-based access | Admin/client permission checks |
| github.com/google/uuid | v1.6+ | Call ID generation | Standard UUID library, lightweight |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/json (stdlib) | Go 1.25 | custom_data JSON handling | Parsing/validating custom_data field (1KB limit) |
| sync (stdlib) | Go 1.25 | In-memory call state tracking | Mutex for active call map |
| time (stdlib) | Go 1.25 | Timer for park timeout, duration calculation | Park timeout goroutine, billing duration |
| context (stdlib) | Go 1.25 | Cancellation propagation | State machine lifecycle control |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| goroutine per call | Event-driven loop | Goroutine per call is simpler to reason about, natural for Go, each call is independent |
| In-memory active call map | Redis call state | In-memory is faster, single-process design for now; Redis needed only for multi-instance in production |
| google/uuid | crypto/rand + fmt | UUID lib handles formatting, version selection correctly |

**Installation:**
```bash
go get github.com/google/uuid
```

## Architecture Patterns

### Recommended Project Structure
```
callback/                    # CALLBACK service - call engine
  callback.go                # Service struct, init, FSClient injection
  initiate.go                # POST /callbacks - initiate callback API
  status.go                  # GET /callbacks/:id - query status API
  hangup.go                  # POST /callbacks/:id/hangup - force hangup API
  list.go                    # GET /callbacks - CDR list with filters
  statemachine.go            # State machine: runCall goroutine
  finalize.go                # finalizeCall: cleanup, billing, wastage
  wastage.go                 # Wastage classification logic
  fsclient/                  # Sub-package: FSClient interface + types
    interface.go             # FSClient interface definition
    types.go                 # CallEvent, OriginateParams, MockConfig
    mock.go                  # MockFSClient implementation
  migrations/
    1_create_callback_calls.up.sql
    2_create_system_configs.up.sql
  callback_test.go           # Integration tests covering all 5 scenarios
```

### Pattern 1: FSClient Interface with Dependency Injection
**What:** Define FSClient as a Go interface on the callback service. Service struct holds the interface, initService() creates mock or real based on environment.
**When to use:** Always -- this is the core abstraction enabling Phase 2 mock and Phase 3 real implementation.
**Example:**
```go
// callback/fsclient/interface.go
package fsclient

import "context"

// FSClient abstracts FreeSWITCH ESL operations.
// Phase 2: MockFSClient. Phase 3: real eslgo-based implementation.
type FSClient interface {
    // OriginateALeg initiates A-leg outbound call, parks the channel.
    // Returns the channel UUID (or mock UUID).
    OriginateALeg(ctx context.Context, params OriginateParams) (uuid string, err error)

    // OriginateBLegAndBridge initiates B-leg and bridges to parked A-leg.
    OriginateBLegAndBridge(ctx context.Context, aUUID string, params OriginateParams) (bUUID string, err error)

    // HangupCall terminates a channel by UUID.
    HangupCall(ctx context.Context, uuid string, cause string) error

    // StartRecording begins recording on a channel (Phase 2: no-op).
    StartRecording(ctx context.Context, uuid string, callID string, leg string) error

    // StopRecording stops recording on a channel (Phase 2: no-op).
    StopRecording(ctx context.Context, uuid string, callID string, leg string) error

    // RegisterEventHandler registers a callback for ESL events.
    // The mock fires events immediately; real impl receives from ESL connection.
    RegisterEventHandler(eventName string, handler func(CallEvent))
}
```

### Pattern 2: State Machine as Goroutine per Call
**What:** Each callback initiation spawns a goroutine that drives the state machine. The goroutine calls FSClient methods and receives events via channels or direct callback invocation (mock). A `finalizeCall` function handles all cleanup paths.
**When to use:** Every callback call.
**Example:**
```go
// callback/statemachine.go
func (s *Service) runCall(ctx context.Context, call *CallbackCall) {
    // Phase 1: Originate A-leg
    aUUID, err := s.fsClient.OriginateALeg(ctx, OriginateParams{...})
    if err != nil {
        s.finalizeCall(call, "a_originate_failed", err)
        return
    }
    call.AfsUUID = aUUID
    s.updateStatus(call, "a_dialing")

    // Wait for A-leg answer or timeout/hangup
    event := s.waitForEvent(call.CallID, "A", parkTimeout)
    if event == nil || event.EventName == "CHANNEL_HANGUP" {
        s.finalizeCall(call, "a_failed", nil)
        return
    }
    // A-leg answered
    s.updateStatus(call, "a_connected")
    call.AConnectAt = event.Timestamp

    // Phase 2: Originate B-leg and bridge
    bUUID, err := s.fsClient.OriginateBLegAndBridge(ctx, aUUID, OriginateParams{...})
    if err != nil {
        // Command-level failure (CALL-03)
        s.handleBLegOriginateFailure(call, aUUID, err)
        return
    }
    s.updateStatus(call, "b_dialing")

    // Wait for B-leg events...
    // Continue through b_connected -> bridged -> finished/failed
}
```

### Pattern 3: Event-Driven Mock with Immediate Callback
**What:** MockFSClient stores registered event handlers. When OriginateALeg is called, it immediately fires CHANNEL_ANSWER or CHANNEL_HANGUP based on MockConfig. This makes tests deterministic and fast.
**When to use:** All Phase 2 testing and interactive development.
**Example:**
```go
// callback/fsclient/mock.go
type MockConfig struct {
    ALegResult    string // "answer", "no_answer", "reject", "error"
    BLegResult    string // "answer", "reject", "error", "busy"
    BridgeResult  string // "stable", "early_hangup"
    BridgeDuration time.Duration // simulated bridge time (for billing calc)
    ALegHangupCause string
    BLegHangupCause string
}

type MockFSClient struct {
    config   MockConfig
    handlers map[string][]func(CallEvent)
    mu       sync.RWMutex
    calls    map[string]*mockCallState // track active mock calls
}

func (m *MockFSClient) OriginateALeg(ctx context.Context, params OriginateParams) (string, error) {
    if m.config.ALegResult == "error" {
        return "", fmt.Errorf("originate command failed")
    }
    uuid := "mock-a-" + params.CallID

    // Fire events based on config
    go func() {
        if m.config.ALegResult == "answer" {
            m.fireEvent(CallEvent{
                CallID: params.CallID, UUID: uuid,
                Leg: "A", EventName: "CHANNEL_ANSWER",
                Timestamp: time.Now(),
            })
        } else {
            m.fireEvent(CallEvent{
                CallID: params.CallID, UUID: uuid,
                Leg: "A", EventName: "CHANNEL_HANGUP",
                HangupCause: mapResultToHangupCause(m.config.ALegResult),
                Timestamp: time.Now(),
            })
        }
    }()
    return uuid, nil
}
```

### Pattern 4: finalizeCall Centralized Cleanup
**What:** A single function handles ALL call termination paths: refund excess pre-deduction, release concurrent slot, classify wastage, update DB status, mark CDR complete.
**When to use:** Every call termination, regardless of success or failure.
**Example:**
```go
// callback/finalize.go
func (s *Service) finalizeCall(call *CallbackCall, reason string, originErr error) {
    ctx := context.Background()

    // 1. Determine wastage type
    wastageType := s.classifyWastage(call, reason)

    // 2. Calculate actual cost and wastage cost
    actualCost, wastageCost := s.calculateCosts(call, wastageType)

    // 3. Finalize billing (refund pre-deduction excess)
    billing.Finalize(ctx, &billing.FinalizeParams{
        UserID:          call.UserID,
        CallID:          call.CallID,
        ALegDurationSec: call.ALegDurationSec(),
        BLegDurationSec: call.BLegDurationSec(),
        ALegRate:        call.ALegRate,
        BLegRate:        call.BLegRate,
        PreDeductAmount: call.PreDeductAmount,
    })

    // 4. Release concurrent slot (NOT using defer - per PROJECT.md decision)
    billing.ReleaseSlot(ctx, &billing.ReleaseSlotParams{UserID: call.UserID})

    // 5. Update DB with final state
    s.updateCallFinal(call, wastageType, wastageCost, actualCost, reason)
}
```

### Pattern 5: Active Call Map for Status Queries and Force Hangup
**What:** Maintain a `sync.Map` or mutex-protected map of active call IDs to their state machine references. This enables real-time status queries and force hangup on in-progress calls.
**When to use:** Status queries and force hangup need to find active calls quickly.
**Example:**
```go
type Service struct {
    fsClient    fsclient.FSClient
    activeCalls sync.Map // map[string]*activeCall
    db          *sqldb.Database
}

type activeCall struct {
    call     *CallbackCall
    cancel   context.CancelFunc // cancels the state machine goroutine
    eventCh  chan fsclient.CallEvent // receives events from FSClient
}
```

### Anti-Patterns to Avoid
- **Using `defer` for concurrent slot release:** The CONTEXT.md and PROJECT.md explicitly forbid this. In ESL async scenarios, defer can trigger double-release. Use `finalizeCall` as the single release point.
- **Separate CDR table:** CONTEXT.md says callback_calls IS the CDR. Do not create a separate cdr_records table.
- **Blocking state machine on DB writes:** State transitions should update DB asynchronously or use non-blocking patterns. The state machine goroutine should not stall on slow DB writes.
- **Float for money/duration:** All monetary values are int64 (fen). Duration stored as milliseconds in int64 for precision, converted to seconds for billing.
- **Retrying B-leg originate:** CONTEXT.md explicitly says no retry -- direct failure. Retry would extend A-leg park time and increase wastage.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| UUID generation | Random string concatenation | `github.com/google/uuid` | Proper UUID v4 format, collision avoidance |
| Call state persistence | Custom file-based state | `encore.dev/storage/sqldb` (callback_calls table) | Transactional, crash-recoverable |
| Concurrent slot tracking | Manual counter with mutex | Phase 1 billing service Redis INCR via API call | Already built, atomic, shared across instances |
| Rate resolution | Inline rate lookup | Phase 1 billing service ResolveRate API | Multi-tier resolution already implemented |
| Blacklist check | Inline DB query | Phase 1 compliance service CheckBlacklist API | Global + client scope already handled |
| Daily limit check | Manual Redis counter | Phase 1 compliance service CheckDailyLimit API | Fail-open policy already implemented |
| Timer-based timeout | Manual `time.After` with goroutine leak | `context.WithTimeout` or `time.AfterFunc` with proper cleanup | Context cancellation propagates correctly |

**Key insight:** Phase 2 heavily depends on Phase 1 services via internal API calls. The callback service orchestrates, it does not re-implement billing, routing, or compliance logic.

## Common Pitfalls

### Pitfall 1: Goroutine Leak on Call Cancellation
**What goes wrong:** State machine goroutine blocks forever waiting for an event that never arrives (e.g., mock doesn't fire hangup after force hangup).
**Why it happens:** Missing timeout on event wait channels.
**How to avoid:** Always use `select` with `case <-ctx.Done()` when waiting for events. The state machine goroutine must have a parent context that gets cancelled on force hangup or service shutdown.
**Warning signs:** Goroutine count growing indefinitely during tests.

### Pitfall 2: Race Condition Between Event Handler and State Machine
**What goes wrong:** Event handler fires before the state machine goroutine is ready to receive, losing the event.
**Why it happens:** Mock fires events synchronously/immediately in the same goroutine that calls Originate.
**How to avoid:** Use buffered channels (capacity 1-2) for event delivery, or fire mock events in a separate goroutine with a tiny delay. The mock's `go func()` in Pattern 3 addresses this.
**Warning signs:** Tests passing intermittently, events appearing "lost".

### Pitfall 3: Double Finalization
**What goes wrong:** Both a timeout handler and an event handler try to finalize the same call, causing double refund or double slot release.
**Why it happens:** Park timeout fires at the same moment as a hangup event.
**How to avoid:** Use `sync.Once` or an atomic flag per call to ensure `finalizeCall` runs exactly once. Check-and-set pattern on the call status in DB (WHERE status NOT IN ('finished', 'failed')).
**Warning signs:** Negative concurrent slot counts, balance inconsistencies.

### Pitfall 4: Mock Not Simulating Event Sequence Correctly
**What goes wrong:** Mock fires CHANNEL_BRIDGE without first firing CHANNEL_ANSWER for B-leg, causing state machine to skip states.
**Why it happens:** Incomplete mock event sequence.
**How to avoid:** Mock must follow the exact event sequence that real FreeSWITCH produces: A CHANNEL_ANSWER -> (B CHANNEL_ANSWER) -> CHANNEL_BRIDGE -> CHANNEL_HANGUP. Document the expected sequence and verify in tests.
**Warning signs:** Status jumps (e.g., a_connected directly to finished without b_dialing).

### Pitfall 5: Millisecond Precision Loss
**What goes wrong:** Duration calculations lose precision when converting between time.Time and int64 seconds.
**Why it happens:** Using `time.Since().Seconds()` which returns float64, or truncating to seconds too early.
**How to avoid:** Store all timestamps as `timestamptz` in PostgreSQL (preserves microseconds). Calculate durations using `time.Duration.Milliseconds()`. Convert to billing seconds only at billing time using ceiling division.
**Warning signs:** Billing amounts off by 1 block unit.

### Pitfall 6: custom_data Validation
**What goes wrong:** Large custom_data payloads or invalid JSON cause DB storage issues or downstream Webhook payload bloat.
**Why it happens:** No size validation on input.
**How to avoid:** Validate custom_data is valid JSON and <= 1KB at API input. Store as JSONB in PostgreSQL (automatic validation). Reject with InvalidArgument if too large.
**Warning signs:** Slow CDR queries due to large JSONB fields.

## Code Examples

### callback_calls Migration Schema
```sql
-- callback/migrations/1_create_callback_calls.up.sql
CREATE TABLE callback_calls (
    id BIGSERIAL PRIMARY KEY,
    call_id VARCHAR(64) UNIQUE NOT NULL,
    user_id BIGINT NOT NULL,

    -- API input
    a_number VARCHAR(32) NOT NULL,
    b_number VARCHAR(32) NOT NULL,
    caller_id VARCHAR(32),
    max_duration INT,
    custom_data JSONB,
    callback_url TEXT,

    -- Status (real-time + CDR)
    status VARCHAR(20) NOT NULL DEFAULT 'initiating' CHECK (status IN (
        'initiating', 'a_dialing', 'a_connected',
        'b_dialing', 'b_connected', 'bridged',
        'finished', 'failed'
    )),

    -- A-Leg details
    a_fs_uuid VARCHAR(64),
    a_gateway_id BIGINT,
    a_gateway_name VARCHAR(100),
    a_dial_at TIMESTAMPTZ,
    a_answer_at TIMESTAMPTZ,
    a_hangup_at TIMESTAMPTZ,
    a_hangup_cause VARCHAR(64),
    a_duration_ms BIGINT DEFAULT 0,

    -- B-Leg details
    b_fs_uuid VARCHAR(64),
    b_gateway_id BIGINT,
    b_gateway_name VARCHAR(100),
    b_dial_at TIMESTAMPTZ,
    b_answer_at TIMESTAMPTZ,
    b_hangup_at TIMESTAMPTZ,
    b_hangup_cause VARCHAR(64),
    b_duration_ms BIGINT DEFAULT 0,

    -- Bridge
    bridge_at TIMESTAMPTZ,
    bridge_end_at TIMESTAMPTZ,
    bridge_duration_ms BIGINT DEFAULT 0,

    -- Billing
    a_leg_rate BIGINT,
    b_leg_rate BIGINT,
    pre_deduct_amount BIGINT DEFAULT 0,
    a_leg_cost BIGINT DEFAULT 0,
    b_leg_cost BIGINT DEFAULT 0,
    total_cost BIGINT DEFAULT 0,

    -- Wastage
    wastage_type VARCHAR(30) CHECK (wastage_type IN (
        'a_connected_b_failed', 'bridge_broken_early'
    )),
    wastage_cost BIGINT,
    wastage_duration_ms BIGINT,

    -- Termination
    hangup_by VARCHAR(10),  -- 'a', 'b', 'system', 'admin', 'client'
    failure_reason VARCHAR(100),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Performance indexes
CREATE INDEX idx_calls_user_created ON callback_calls(user_id, created_at DESC);
CREATE INDEX idx_calls_status ON callback_calls(status) WHERE status NOT IN ('finished', 'failed');
CREATE INDEX idx_calls_wastage ON callback_calls(wastage_type, created_at) WHERE wastage_type IS NOT NULL;
CREATE INDEX idx_calls_a_number ON callback_calls(a_number, created_at DESC);
CREATE INDEX idx_calls_b_number ON callback_calls(b_number, created_at DESC);
CREATE INDEX idx_calls_call_id ON callback_calls(call_id);
```

### Wastage Classification Logic
```go
// callback/wastage.go
func classifyWastage(call *CallbackCall, bridgeBrokenThresholdSec int64) (wastageType string, wastageCost int64) {
    // Case 1: A connected but B never connected
    if call.AAnswerAt != nil && call.BAnswerAt == nil {
        aWaitDurationMs := call.AHangupAt.Sub(*call.AAnswerAt).Milliseconds()
        // A-leg cost during the wait = wastage cost
        aWaitSec := (aWaitDurationMs + 999) / 1000 // ceil to seconds
        wastageCost = billing.CalculateCost(aWaitSec, call.ALegRate, 6) // 6s blocks
        return "a_connected_b_failed", wastageCost
    }

    // Case 2: Bridge established but broken too early
    if call.BridgeAt != nil && call.BridgeEndAt != nil {
        bridgeDurationSec := call.BridgeEndAt.Sub(*call.BridgeAt).Milliseconds() / 1000
        if bridgeDurationSec < bridgeBrokenThresholdSec {
            // Both legs cost during short bridge = wastage cost
            totalCost := call.ALegCost + call.BLegCost
            return "bridge_broken_early", totalCost
        }
    }

    return "", 0 // no wastage
}
```

### Initiate Callback API
```go
// callback/initiate.go

type InitiateCallbackParams struct {
    ANumber          string          `json:"a_number"`
    BNumber          string          `json:"b_number"`
    CallerID         *string         `json:"caller_id,omitempty"`
    MaxDuration      *int            `json:"max_duration,omitempty"`
    CustomData       json.RawMessage `json:"custom_data,omitempty"`
    PreferredGateway *string         `json:"preferred_gateway,omitempty"`
    CallbackURL      *string         `json:"callback_url,omitempty"`
}

func (p *InitiateCallbackParams) Validate() error {
    if p.ANumber == "" || p.BNumber == "" {
        return &errs.Error{Code: errs.InvalidArgument, Message: "a_number and b_number are required"}
    }
    if len(p.CustomData) > 1024 {
        return &errs.Error{Code: errs.InvalidArgument, Message: "custom_data must be <= 1KB"}
    }
    return nil
}

type InitiateCallbackResponse struct {
    CallID string `json:"call_id"`
    Status string `json:"status"`
}

//encore:api auth method=POST path=/callbacks
func (s *Service) InitiateCallback(ctx context.Context, p *InitiateCallbackParams) (*InitiateCallbackResponse, error) {
    userData := auth.Data().(*authpkg.AuthData)

    // 1. Compliance checks (calls Phase 1 services)
    if err := compliance.CheckBlacklist(ctx, &compliance.CheckBlacklistParams{
        CalledNumber: p.BNumber, UserID: userData.UserID,
    }); err != nil {
        return nil, err
    }
    if err := compliance.CheckDailyLimit(ctx, &compliance.CheckDailyLimitParams{
        UserID: userData.UserID, DailyLimit: userData.DailyLimit,
    }); err != nil {
        return nil, err
    }

    // 2. Resolve rates
    rates, err := billing.ResolveRate(ctx, &billing.ResolveRateParams{
        UserID: userData.UserID, CalledPrefix: p.BNumber[:3],
    })
    if err != nil {
        return nil, err
    }

    // 3. Pre-deduct balance
    callID := uuid.New().String()
    preDeduct, err := billing.PreDeduct(ctx, &billing.PreDeductParams{
        UserID: userData.UserID, CallID: callID,
        ALegRate: rates.ALegRate, BLegRate: rates.BLegRate,
    })
    if err != nil {
        return nil, err
    }

    // 4. Acquire concurrent slot
    _, err = billing.AcquireSlot(ctx, &billing.AcquireSlotParams{
        UserID: userData.UserID, MaxConcurrent: userData.MaxConcurrent,
    })
    if err != nil {
        // Refund pre-deduction on slot failure
        billing.Finalize(ctx, &billing.FinalizeParams{...zero duration...})
        return nil, err
    }

    // 5. Route selection
    aGateway, _ := routing.PickALeg(ctx, &routing.PickALegParams{})
    bGateway, _ := routing.PickBLeg(ctx, &routing.PickBLegParams{CalledNumber: p.BNumber})
    did, _ := routing.SelectDID(ctx, &routing.SelectDIDParams{UserID: userData.UserID})

    // 6. Create DB record
    call := &CallbackCall{
        CallID: callID, UserID: userData.UserID,
        ANumber: p.ANumber, BNumber: p.BNumber,
        // ... all fields
        Status: "a_dialing",
    }
    s.insertCall(ctx, call)

    // 7. Launch state machine
    go s.runCall(context.Background(), call)

    return &InitiateCallbackResponse{CallID: callID, Status: "a_dialing"}, nil
}
```

### Event Delivery Pattern for Mock
```go
// callback/fsclient/mock.go

// The mock uses channels to deliver events to the state machine.
// The state machine registers per-call event channels via RegisterEventHandler.

func (m *MockFSClient) RegisterEventHandler(eventName string, handler func(CallEvent)) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.handlers[eventName] = append(m.handlers[eventName], handler)
}

func (m *MockFSClient) fireEvent(event CallEvent) {
    m.mu.RLock()
    handlers := m.handlers[event.EventName]
    m.mu.RUnlock()
    for _, h := range handlers {
        h(event)
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Separate CDR table + callback_calls | Unified callback_calls as CDR | CONTEXT.md decision | Simpler schema, no sync needed |
| DECIMAL for money | BIGINT (fen/cents) | Phase 1 convention | No float precision issues |
| defer for cleanup | finalizeCall centralized | PROJECT.md decision | Prevents double-release in async ESL |
| Blocking mock (sleep) | Instant mock with goroutine events | CONTEXT.md decision | Tests run in milliseconds |

## Open Questions

1. **system_configs table ownership**
   - What we know: bridge_broken_early_threshold_sec needs to be configurable (default 10s). The PRD shows system_configs as a standalone table.
   - What's unclear: Should the callback service own this table, or should it be a shared config service? Phase 1 doesn't define a system_configs service.
   - Recommendation: Create system_configs as part of the callback service migration. It's the primary consumer. Other services can read via `sqldb.Named("callback")` if needed, or a simple config API can be added later.

2. **Event routing from mock to specific call**
   - What we know: The mock fires events globally. The state machine needs events for a specific call.
   - What's unclear: How to route events to the correct call's goroutine.
   - Recommendation: Include `CallID` in every event. The state machine goroutine filters on its own CallID. Use a per-call event channel registered in the active call map. The mock fires to all handlers, each handler checks CallID and routes to the correct channel.

3. **Encore service calling Phase 1 service APIs**
   - What we know: Encore allows cross-service calls as regular function calls (import and call).
   - What's unclear: Whether Phase 1 services expose the right private API signatures for the callback service to consume.
   - Recommendation: Phase 1 services define private APIs (billing.PreDeduct, billing.AcquireSlot, etc.) that the callback service calls. The planner should verify these API signatures match during planning. If mismatches exist, add adapter tasks.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + encore test |
| Config file | none -- Encore uses standard Go test conventions |
| Quick run command | `encore test ./callback/... -v` |
| Full suite command | `encore test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CALL-01 | Full callback flow: A dials, answers, B dials, answers, bridge, finish | integration | `encore test ./callback/... -run TestFullCallbackFlow -x` | No -- Wave 0 |
| CALL-02 | A-leg park timeout after 60s triggers hangup + cleanup | unit | `encore test ./callback/... -run TestParkTimeout -x` | No -- Wave 0 |
| CALL-03 | B-leg originate failure triggers wastage + hangup A + refund + slot release | unit | `encore test ./callback/... -run TestBLegOriginateFailure -x` | No -- Wave 0 |
| CALL-04 | State transitions cover full sequence a_dialing through finished/failed | integration | `encore test ./callback/... -run TestStateTransitions -x` | No -- Wave 0 |
| CALL-05 | GetCallStatus returns all fields including A/B leg details and wastage | unit | `encore test ./callback/... -run TestGetCallStatus -x` | No -- Wave 0 |
| CALL-06 | Force hangup: admin can hangup any, client only own calls | unit | `encore test ./callback/... -run TestForceHangup -x` | No -- Wave 0 |
| CALL-07 | Mock FSClient covers all 5 scenarios without real FS | integration | `encore test ./callback/... -run TestMock -x` | No -- Wave 0 |
| WAST-01 | Wastage classified as a_connected_b_failed or bridge_broken_early | unit | `encore test ./callback/... -run TestWastageClassification -x` | No -- Wave 0 |
| WAST-02 | Wastage cost correctly calculated and persisted | unit | `encore test ./callback/... -run TestWastageCost -x` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `encore test ./callback/...`
- **Per wave merge:** `encore test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `callback/callback_test.go` -- covers CALL-01 through CALL-07, WAST-01, WAST-02
- [ ] `callback/fsclient/mock.go` -- MockFSClient implementation
- [ ] `callback/fsclient/mock_test.go` -- Mock behavior verification
- [ ] `callback/migrations/1_create_callback_calls.up.sql` -- table schema
- [ ] `callback/migrations/2_create_system_configs.up.sql` -- bridge threshold config
- [ ] Phase 1 services must be complete (auth, billing, routing, compliance) -- callback service calls their APIs

## Sources

### Primary (HIGH confidence)
- CONTEXT.md (Phase 2 decisions) -- all locked decisions on state machine, mock, CDR design
- PROJECT.md -- FSClient 5-method interface, finalizeCall design, architecture decisions
- PRD v3.1 sections 4.2, 5.1, 6.1, 6.2 -- FSClient code, callback_calls schema, call flow sequence
- Phase 1 RESEARCH.md and PLANs -- billing, routing, compliance API signatures

### Secondary (MEDIUM confidence)
- Encore.go documentation (from system prompt) -- service patterns, sqldb, auth, errs, testing
- Go stdlib documentation -- sync.Map, context, time patterns

### Tertiary (LOW confidence)
- None -- this phase is well-constrained by PRD and CONTEXT.md decisions

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries already established in Phase 1, no new external dependencies
- Architecture: HIGH - state machine pattern well-defined in PRD and CONTEXT.md, clear precedent in telecom
- Pitfalls: HIGH - common goroutine/concurrency patterns in Go are well-understood
- Mock design: HIGH - CONTEXT.md provides detailed mock specification
- Integration with Phase 1: MEDIUM - API signatures assumed from Phase 1 PLANs, may need adjustment

**Research date:** 2026-03-10
**Valid until:** 2026-04-10 (stable ecosystem, locked decisions from CONTEXT.md)
