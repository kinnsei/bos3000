# BOS3000 API 回拨系统 PRD（FreeSWITCH 版）

**版本**: v3.1-FreeSWITCH
**架构**: Encore.go + FreeSWITCH ESL + sqlc + React/Vite
**模式**: 纯 API 发起（无接入号）
**日期**: 2026-03-09
**默认 Web 端口**: 12345（避开 80/443/8080 等常见受限端口）

---

## 目录

1. 项目概述与架构决策
2. 技术架构总览
3. FreeSWITCH 设计详解
4. eslgo 集成方案
5. 数据库设计
6. 呼叫流程与状态机
7. API 接口定义
8. 计费与损耗模型
9. 路由引擎
10. 双模式权限体系
11. Admin Dashboard 功能结构
12. Client Portal 功能结构
13. 前端技术栈
14. 非功能性需求与 SLA
15. 部署与运维
16. 错误码规范
17. 开发阶段与验收标准

---

## 1. 项目概述与架构决策

### 1.1 业务模型

API 回拨双呼（Click-to-Call）：管理员或客户通过 API/Web 发起呼叫，系统先外呼 A 路（主叫方），A 接通后再外呼 B 路（被叫方），桥接双方完成通话。

### 1.2 为什么从 OpenSIPS B2BUA 切换到纯 FreeSWITCH

| 维度 | OpenSIPS B2BUA 方案 | FreeSWITCH ESL 方案 |
|------|--------------------|--------------------|
| 双呼实现 | 需要 `b2b_logic.xml` 场景配置 + MI 命令序列 | `originate` + `bridge` 两条命令 |
| 录音 | 需要独立 rtpengine 双实例 | FS 原生 `uuid_record`，无需额外组件 |
| Go 控制接口 | MI HTTP JSON-RPC（语义复杂） | ESL TCP 文本协议（直观） |
| 调试 | OpenSIPS 日志难读，B2BUA 状态不透明 | FS `fs_cli` 实时查看通话，日志清晰 |
| 组件数量 | OpenSIPS + rtpengine-A + rtpengine-B | 仅 FreeSWITCH |
| 通话中能力 | 转接/咨询转等原生支持 | 通过 ESL 命令支持，略需编排 |

**结论**：本系统没有复杂的 SIP 拓扑隐藏需求，回拨双呼是 FreeSWITCH `originate` 的经典用例，去掉 OpenSIPS + rtpengine 后整个信令媒体层简化为单一组件。

### 1.3 核心架构特征

- **API-First**：纯 API 发起，无接入号，无呼入路由逻辑
- **ESL 控制**：Encore.go 通过 ESL（Event Socket）直接驱动 FreeSWITCH
- **A/B 双外呼**：A 路 `originate` + park（含超时保护），A 接通后 B 路 `originate` + bridge
- **原生录音**：FS `uuid_record` 在 CHANNEL_BRIDGE 后开始录制，保证录音与通话对齐
- **损耗分析**：精准统计"A 路已接通但 B 路未接通"的损耗成本
- **双模式权限**：管理员全平台 / 客户自助
- **默认端口 12345**：所有 Web 服务端口统一使用 12345，避免在受限环境下 80/443/8080 不可用的问题

---

## 2. 技术架构总览

```
┌──────────────────────────────────────────────────────────────────────────┐
│                            前端层 (Frontend)                              │
│  ┌──────────────────────────┐   ┌────────────────────────────────────┐   │
│  │    Admin Dashboard       │   │    Client Portal                   │   │
│  │    React 18 + Vite 5     │   │    React 18 + Vite 5               │   │
│  │    shadcn/ui + MagicUI   │   │    shadcn/ui + MagicUI             │   │
│  │    Tailwind CSS 4        │   │    Tailwind CSS 4                  │   │
│  │    port: 12345           │   │    port: 12346                     │   │
│  └──────────┬───────────────┘   └───────────────┬────────────────────┘   │
└─────────────┼─────────────────────────────────── ┼────────────────────────┘
              │ REST / WebSocket (:12345)          │
┌─────────────▼────────────────────────────────────▼────────────────────────┐
│                     Encore.go (Control Plane, port 12345)                  │
│  ┌───────────────────────────────────────────────────────────────────────┐│
│  │  callback service  │  routing service  │  billing service  │  webhook  ││
│  └────────────┬────────────────────────────────────────────────────┬──────┘│
│               │                                                    │       │
│  ┌────────────▼──────────────────────────────────────────────────┐ │       │
│  │              sqlc (PostgreSQL)        Redis                   │ │       │
│  │  callback_calls / users / gateways    · 并发计数 INCR/DECR    │ │       │
│  │  cdr_records / did_numbers            · 余额行锁              │ │       │
│  │  system_configs / rate_plans          · Call State Cache      │ │       │
│  │  webhook_deliveries                                           │ │       │
│  └────────────────────────────────────────────────────────────────┘ │       │
│               │ ESL TCP (port 8021)                                 │       │
└───────────────┼─────────────────────────────────────────────────────┘       │
                │                                                              │
┌───────────────▼──────────────────────────────────────────────────────────┐  │
│                         FreeSWITCH 1.10+                                  │  │
│                                                                            │  │
│  ┌─────────────────┐  ┌─────────────────┐  ┌────────────────────────┐    │  │
│  │  sofia (SIP)    │  │  mod_event_socket│  │  mod_dptools           │    │  │
│  │                 │  │  (ESL 控制接口)  │  │  · park (带超时保护)   │    │  │
│  │  Gateway Pool A │  └────────┬────────┘  │  · bridge              │    │  │
│  │  (A路外呼网关)  │           │            │  · uuid_record         │    │  │
│  │                 │    Events/Commands     │  · uuid_kill           │    │  │
│  │  Gateway Pool B │           │            └────────────────────────┘    │  │
│  │  (B路外呼网关)  │  ┌────────▼────────┐                                 │  │
│  └─────────────────┘  │  FSClient (Go)  │  ← eslgo 封装层                │  │
│                        └─────────────────┘                                │  │
│  录音存储: /recordings/{call_id}/a.wav  /recordings/{call_id}/b.wav       │  │
└──────────────────────────────────────────────────────────────────────────┘  │
                                                                               │
┌──────────────────────────────────────────────────────────────────────────┐  │
│  Webhook Worker (Encore PubSub)                                           │◄─┘
│  · 延迟队列重投递（非阻塞）  · DLQ  · 状态变更推送                        │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 3. FreeSWITCH 设计详解

### 3.1 模块配置

```xml
<!-- /etc/freeswitch/autoload_configs/modules.conf.xml（精简版） -->
<configuration name="modules.conf">
  <modules>
    <load module="mod_sofia"/>          <!-- SIP 栈 -->
    <load module="mod_event_socket"/>   <!-- ESL 接口 -->
    <load module="mod_dptools"/>        <!-- park, bridge, playback -->
    <load module="mod_commands"/>       <!-- uuid_record, uuid_kill 等 -->
    <load module="mod_native_file"/>    <!-- 录音格式支持 -->
    <load module="mod_sndfile"/>        <!-- WAV/MP3 支持 -->
    <load module="mod_logfile"/>        <!-- 日志 -->
    <!-- 不加载 mod_dialplan_xml 以外的 dialplan 模块，保持最简 -->
  </modules>
</configuration>
```

### 3.2 ESL 接口配置

```xml
<!-- /etc/freeswitch/autoload_configs/event_socket.conf.xml -->
<configuration name="event_socket.conf">
  <settings>
    <param name="nat-map" value="false"/>
    <param name="listen-ip" value="127.0.0.1"/>  <!-- 仅本地，不对外暴露 -->
    <param name="listen-port" value="8021"/>
    <param name="password" value="$${esl_password}"/>  <!-- 使用 FS 原生变量语法 -->
    <param name="apply-inbound-acl" value="loopback.auto"/>
  </settings>
</configuration>
```

> **部署说明**：FreeSWITCH 原生不支持 `${}` 格式的环境变量替换，须使用 `$${var}` 语法并在 `vars.xml` 中定义变量。容器部署时通过 entrypoint 脚本 `envsubst` 将环境变量 `ESL_PASSWORD` 写入 `vars.xml`：
>
> ```bash
> # docker-entrypoint.sh
> echo "<X-PRE-PROCESS cmd=\"set\" data=\"esl_password=${ESL_PASSWORD}\"/>" \
>   > /etc/freeswitch/vars_env.xml
> ```

### 3.3 SIP Profile（sofia）

系统仅做外呼，无呼入路由。使用一个 `egress` profile，网关按 A/B 路分组：

```xml
<!-- /etc/freeswitch/sip_profiles/egress.xml -->
<profile name="egress">
  <settings>
    <param name="context" value="egress"/>
    <param name="dialplan" value="XML"/>
    <param name="rtp-ip" value="$${local_ip_v4}"/>
    <param name="sip-ip" value="$${local_ip_v4}"/>
    <param name="sip-port" value="5080"/>   <!-- 专用外呼端口 -->
    <param name="codec-prefs" value="PCMA,PCMU,G729"/>
    <param name="inbound-codec-negotiation" value="generous"/>
    <param name="record-path" value="/recordings"/>
    <param name="record-template" value="${call_id}/${leg}.wav"/>
  </settings>

  <!-- A路网关池（成本相同，Encore 侧轮询） -->
  <gateways>
    <X-PRE-PROCESS cmd="include" data="../gateways/a-leg/*.xml"/>
    <X-PRE-PROCESS cmd="include" data="../gateways/b-leg/*.xml"/>
  </gateways>
</profile>
```

**A 路网关示例**（`gateways/a-leg/a-pool-01.xml`）：

```xml
<gateway name="a-pool-01">
  <param name="proxy"    value="sip-a01.bos3000.local"/>
  <param name="register" value="false"/>   <!-- 纯外呼，无需注册 -->
  <param name="ping"     value="30"/>      <!-- 健康检查间隔（秒） -->
  <param name="ping-max" value="3"/>       <!-- 连续3次失败标记 DOWN -->
</gateway>
```

**B 路网关示例**（`gateways/b-leg/b-mobile-01.xml`）：

```xml
<gateway name="b-mobile-01">
  <param name="proxy"    value="sip-mobile.bj.local"/>
  <param name="register" value="false"/>
  <param name="ping"     value="30"/>
  <param name="ping-max" value="3"/>
</gateway>
```

### 3.4 Dialplan

系统通过 ESL `originate` 发起外呼，dialplan 只需要一个极简的 egress context，**含 park 超时保护**：

```xml
<!-- /etc/freeswitch/dialplan/egress.xml -->
<context name="egress">
  <!-- 所有外呼通过 ESL originate 发起，直接走到 park 等待桥接 -->
  <!-- park_timeout 防止 Go 侧异常未能发起 B 路时 A 路通道泄漏 -->
  <extension name="catch-all">
    <condition>
      <action application="set" data="park_timeout=60"/>
      <action application="set" data="park_timeout_transfer=hangup:NORMAL_CLEARING"/>
      <action application="park"/>
    </condition>
  </extension>
</context>
```

> **设计说明**：将业务逻辑全部放在 Go 的 ESL 事件处理器里，而非 dialplan XML。这样状态机完全在 Encore.go 中维护，调试和测试都更容易。`park_timeout=60` 确保即使 Go 侧 `dialBLeg()` 因 bug 未被调用，A 路通道也会在 60 秒后自动挂断，避免资源泄漏和持续计费。

---

## 4. eslgo 集成方案

### 4.1 eslgo 使用决策

**结论：引用 eslgo，但浅度封装，不暴露其类型到业务层。**

| 维度 | 说明 |
|------|------|
| 协议复杂度 | ESL 是简单的 TCP 文本协议，eslgo 省掉的是事件解析和连接重连逻辑（约 300 行样板） |
| 维护活跃度 | 一般，不建议依赖其高层抽象 |
| 封装策略 | 在 eslgo 上包一层 `FSClient`，对外只暴露 5 个方法，隔离底层依赖 |
| 替换成本 | 封装后替换底层库改动 < 100 行 |

### 4.2 FSClient 封装设计

```go
// internal/fsclient/client.go
// 对业务层暴露的接口，完全隔离 eslgo 类型

package fsclient

import (
    "context"
    "fmt"
    "strings"
    "sync"
    "time"

    "github.com/0x19/eslgo"
)

// CallEvent 是业务层关心的事件类型
type CallEvent struct {
    CallID    string            // X-BOS-Call-ID 自定义头
    UUID      string            // FreeSWITCH channel UUID
    Leg       string            // "A" 或 "B"
    EventName string            // CHANNEL_ANSWER / CHANNEL_HANGUP / CHANNEL_BRIDGE
    HangupCause string          // NORMAL_CLEARING / NO_ANSWER / USER_BUSY 等
    Variables map[string]string // 透传的 channel variables
    Timestamp time.Time
}

// FSClient 是对 FreeSWITCH ESL 的封装，业务层只与此接口交互
type FSClient struct {
    conn     *eslgo.Conn
    host     string
    password string
    handlers map[string][]func(CallEvent)
    mu       sync.RWMutex
    healthy  bool        // 健康状态，由 healthCheck 维护
    lastPing time.Time   // 最近一次成功 ping 的时间
}

func New(host, password string) (*FSClient, error) {
    conn, err := eslgo.Dial(host, password, func(conn *eslgo.RawConn) {
        // 重连回调（eslgo 提供）
    })
    if err != nil {
        return nil, fmt.Errorf("FSClient connect failed: %w", err)
    }

    c := &FSClient{
        conn:     conn,
        host:     host,
        password: password,
        handlers: make(map[string][]func(CallEvent)),
        healthy:  true,
        lastPing: time.Now(),
    }
    c.subscribeEvents()
    go c.healthCheckLoop()  // 启动健康探测
    return c, nil
}

// IsHealthy 返回当前连接健康状态
func (c *FSClient) IsHealthy() bool {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.healthy
}

// healthCheckLoop 周期性发送 api status 探测连接是否可用
func (c *FSClient) healthCheckLoop() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    failCount := 0

    for range ticker.C {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        _, err := c.conn.SendCommand(ctx, eslgo.NewCommand("api", "status"))
        cancel()

        c.mu.Lock()
        if err != nil {
            failCount++
            if failCount >= 3 {
                c.healthy = false
            }
        } else {
            failCount = 0
            c.healthy = true
            c.lastPing = time.Now()
        }
        c.mu.Unlock()
    }
}

// OriginateALeg 发起 A 路外呼，将通道 park 等待后续桥接
// 返回 FreeSWITCH channel UUID
func (c *FSClient) OriginateALeg(ctx context.Context, params OriginateParams) (string, error) {
    // originate 命令格式：
    // originate {var1=val1,var2=val2}sofia/gateway/<gateway>/<callee> &park()
    vars := buildChannelVars(map[string]string{
        "origination_caller_id_number": params.CallerID,
        "origination_caller_id_name":   params.CallerID,
        "bos_call_id":                  params.CallID,
        "bos_leg":                      "A",
        "bos_user_id":                  params.UserID,
        "ignore_early_media":           "false",   // 播放回铃音给 A 路
        "call_timeout":                 fmt.Sprintf("%d", params.MaxDialingSec),
        "record_path":                  fmt.Sprintf("/recordings/%s", params.CallID),
        // park_timeout 由 dialplan 设置，此处不重复
    })

    cmd := fmt.Sprintf("originate %s sofia/gateway/%s/%s &park()",
        vars, params.Gateway, params.Callee)

    resp, err := c.conn.SendCommand(ctx, eslgo.NewCommand("bgapi", cmd))
    if err != nil {
        return "", fmt.Errorf("originate A leg failed: %w", err)
    }

    // bgapi 返回 Job-UUID，实际 channel UUID 从 BACKGROUND_JOB 事件获取
    return resp.GetHeader("Job-UUID"), nil
}

// OriginateBLegAndBridge 发起 B 路外呼并桥接到已 park 的 A 路通道
func (c *FSClient) OriginateBLegAndBridge(ctx context.Context, aUUID string, params OriginateParams) error {
    vars := buildChannelVars(map[string]string{
        "origination_caller_id_number": params.CallerID,
        "bos_call_id":                  params.CallID,
        "bos_leg":                      "B",
        "call_timeout":                 fmt.Sprintf("%d", params.MaxDialingSec),
        "ignore_early_media":           "true",    // B 路不播回铃音给 A
    })

    // originate B 路，接通后直接 bridge 到 A 路 UUID
    cmd := fmt.Sprintf("originate %s sofia/gateway/%s/%s &bridge(%s)",
        vars, params.Gateway, params.Callee, aUUID)

    _, err := c.conn.SendCommand(ctx, eslgo.NewCommand("bgapi", cmd))
    return err
}

// HangupCall 挂断指定通道
func (c *FSClient) HangupCall(ctx context.Context, uuid, cause string) error {
    if cause == "" {
        cause = "NORMAL_CLEARING"
    }
    _, err := c.conn.SendCommand(ctx,
        eslgo.NewCommand("api", fmt.Sprintf("uuid_kill %s %s", uuid, cause)))
    return err
}

// StartRecording 开始录音（A/B 路分别录制）
func (c *FSClient) StartRecording(ctx context.Context, uuid, callID, leg string) error {
    path := fmt.Sprintf("/recordings/%s/%s.wav", callID, strings.ToLower(leg))
    _, err := c.conn.SendCommand(ctx,
        eslgo.NewCommand("api", fmt.Sprintf("uuid_record %s start %s", uuid, path)))
    return err
}

// StopRecording 停止录音
func (c *FSClient) StopRecording(ctx context.Context, uuid, callID, leg string) error {
    path := fmt.Sprintf("/recordings/%s/%s.wav", callID, strings.ToLower(leg))
    _, err := c.conn.SendCommand(ctx,
        eslgo.NewCommand("api", fmt.Sprintf("uuid_record %s stop %s", uuid, path)))
    return err
}

// OnEvent 注册事件处理器
func (c *FSClient) OnEvent(eventName string, handler func(CallEvent)) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.handlers[eventName] = append(c.handlers[eventName], handler)
}

// 内部：订阅关心的事件
func (c *FSClient) subscribeEvents() {
    events := []string{
        "CHANNEL_ANSWER",
        "CHANNEL_HANGUP",
        "CHANNEL_BRIDGE",
        "CHANNEL_UNBRIDGE",
        "BACKGROUND_JOB",
    }
    c.conn.Subscribe(strings.Join(events, " "))

    go c.conn.Handle(func(event *eslgo.RawEvent) {
        name := event.GetHeader("Event-Name")
        if name == "" {
            return
        }

        ce := CallEvent{
            CallID:      event.GetHeader("variable_bos_call_id"),
            UUID:        event.GetHeader("Unique-ID"),
            Leg:         event.GetHeader("variable_bos_leg"),
            EventName:   name,
            HangupCause: event.GetHeader("Hangup-Cause"),
            Timestamp:   time.Now(),
            Variables:   make(map[string]string),
        }

        // 透传关键 channel variable
        for _, key := range []string{"variable_bos_call_id", "variable_bos_leg",
            "variable_bos_user_id", "variable_duration", "variable_billsec"} {
            if v := event.GetHeader(key); v != "" {
                ce.Variables[key] = v
            }
        }

        c.mu.RLock()
        handlers := c.handlers[name]
        c.mu.RUnlock()

        for _, h := range handlers {
            h(ce)
        }
    })
}

// OriginateParams 发起外呼的参数
type OriginateParams struct {
    CallID        string
    UserID        string
    CallerID      string // 外显号码
    Callee        string // 被叫号码
    Gateway       string // FreeSWITCH gateway name
    MaxDialingSec int
}

// buildChannelVars 构建 {key=val,...} 格式的 channel variables
func buildChannelVars(vars map[string]string) string {
    parts := make([]string, 0, len(vars))
    for k, v := range vars {
        parts = append(parts, fmt.Sprintf("%s=%s", k, v))
    }
    return "{" + strings.Join(parts, ",") + "}"
}
```

### 4.3 连接管理与重连（含健康探测）

```go
// internal/fsclient/manager.go
// 生产环境：连接池 + 自动重连 + 健康感知轮询

type Manager struct {
    clients    []*FSClient
    hosts      []string    // 支持多 FS 实例（高可用）
    password   string
    mu         sync.RWMutex
    roundRobin atomic.Uint64  // 轮询计数器
}

func NewManager(hosts []string, password string) *Manager {
    m := &Manager{hosts: hosts, password: password}
    for _, host := range hosts {
        client, err := New(host, password)
        if err != nil {
            log.Printf("FS connect failed %s: %v, will retry", host, err)
            go m.reconnectLoop(host)
            continue
        }
        m.clients = append(m.clients, client)
    }
    return m
}

// Pick 选择一个健康的 FS 连接（加权轮询，跳过不健康实例）
func (m *Manager) Pick() (*FSClient, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if len(m.clients) == 0 {
        return nil, fmt.Errorf("no available FreeSWITCH connections")
    }

    n := len(m.clients)
    start := int(m.roundRobin.Add(1)) % n
    for i := 0; i < n; i++ {
        idx := (start + i) % n
        if m.clients[idx].IsHealthy() {
            return m.clients[idx], nil
        }
    }
    // 所有实例不健康，降级返回第一个（仍有 TCP 连接）
    return nil, fmt.Errorf("all FreeSWITCH instances unhealthy")
}

func (m *Manager) reconnectLoop(host string) {
    backoff := 5 * time.Second
    for {
        time.Sleep(backoff)
        client, err := New(host, m.password)
        if err != nil {
            backoff = min(backoff*2, 60*time.Second)
            continue
        }
        m.mu.Lock()
        m.clients = append(m.clients, client)
        m.mu.Unlock()
        log.Printf("FS reconnected: %s", host)
        return
    }
}
```

---

## 5. 数据库设计

### 5.1 核心表结构

```sql
-- 用户表
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(64) UNIQUE NOT NULL,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    user_type     VARCHAR(20) NOT NULL CHECK (user_type IN ('admin', 'client')),
    status        VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active','suspended','closed')),

    phone         VARCHAR(32),
    company_name  VARCHAR(128),

    -- 财务
    account_type  VARCHAR(20) CHECK (account_type IN ('prepaid','postpaid')),
    balance       DECIMAL(10,4) DEFAULT 0,
    credit_limit  DECIMAL(10,4) DEFAULT 0,

    -- 客户费率配置
    -- 优先级：users.a_leg_rate > 0 时使用用户级费率，否则取 rate_plans 模板值
    a_leg_mode    VARCHAR(20) DEFAULT 'free',  -- free / charge / passthrough
    a_leg_rate    DECIMAL(10,4) DEFAULT 0,
    rate_plan_id  UUID,

    -- 限制
    daily_limit      INT DEFAULT 10000,
    concurrent_limit INT DEFAULT 100,

    -- 回调
    webhook_url VARCHAR(255),

    -- API Key
    api_key_hash   VARCHAR(128),
    api_key_prefix VARCHAR(8),
    ip_whitelist   INET[],

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- FreeSWITCH 网关配置表（替代 routing_endpoints）
-- 直接映射到 FS sofia gateway，Go 侧路由选择后将 gateway name 传给 FS originate
CREATE TABLE fs_gateways (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(64) UNIQUE NOT NULL,  -- 对应 FS gateway name，如 "a-pool-01"

    leg     VARCHAR(1) NOT NULL CHECK (leg IN ('A','B')),
    carrier VARCHAR(20) CHECK (carrier IN ('mobile','unicom','telecom','any')),

    -- B路：精确3位前缀（每条记录一个前缀）
    -- A路：此字段为 NULL（全量轮询）
    prefix VARCHAR(3),

    -- 成本与容量
    cost_per_min    DECIMAL(10,4),
    base_rate_per_min DECIMAL(10,4),
    max_concurrent  INT DEFAULT 1000,

    -- 容灾：指向同运营商备用网关
    failover_gateway_id UUID REFERENCES fs_gateways(id),

    priority INT DEFAULT 0,

    -- FS 上报的网关状态（由健康检查同步）
    fs_status   VARCHAR(20) DEFAULT 'active',  -- active / down / restarting
    health_score INT DEFAULT 100,
    last_check_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT NOW()
);

-- DID 号码池
CREATE TABLE did_numbers (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number  VARCHAR(32) UNIQUE NOT NULL,
    carrier VARCHAR(20) NOT NULL,
    leg     VARCHAR(1) NOT NULL CHECK (leg IN ('A','B')),
    user_id UUID REFERENCES users(id),  -- NULL 表示公共池
    status  VARCHAR(20) DEFAULT 'available' CHECK (status IN ('available','in_use','suspended')),
    created_at TIMESTAMP DEFAULT NOW()
);

-- 呼叫主表
-- v3.1：Webhook 字段已拆到 webhook_deliveries 表，减少主表锁竞争
CREATE TABLE callback_calls (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_id VARCHAR(64) UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    custom_id VARCHAR(64),

    -- API 传入字段
    metadata       JSONB,
    record         BOOLEAN DEFAULT TRUE,
    max_dialing_sec INT DEFAULT 45,

    -- A-Leg
    a_fs_uuid   VARCHAR(64),    -- FreeSWITCH channel UUID（A路）
    a_gateway   VARCHAR(64),    -- 实际使用的 gateway name
    a_carrier   VARCHAR(64),
    a_caller    VARCHAR(32) NOT NULL,
    a_callee    VARCHAR(32) NOT NULL,
    a_connect_at    TIMESTAMP,
    a_disconnect_at TIMESTAMP,
    a_duration_sec  INT DEFAULT 0,
    a_cost          DECIMAL(10,4) DEFAULT 0,
    a_charge        DECIMAL(10,4) DEFAULT 0,
    a_record_path   VARCHAR(255),   -- /recordings/{call_id}/a.wav

    -- B-Leg
    b_fs_uuid   VARCHAR(64),    -- FreeSWITCH channel UUID（B路）
    b_gateway   VARCHAR(64),
    b_carrier   VARCHAR(64),
    b_caller    VARCHAR(32) NOT NULL,
    b_callee    VARCHAR(32) NOT NULL,
    b_original_callee VARCHAR(32),
    b_connect_at    TIMESTAMP,
    b_disconnect_at TIMESTAMP,
    b_duration_sec  INT DEFAULT 0,
    b_cost          DECIMAL(10,4) DEFAULT 0,
    b_charge        DECIMAL(10,4) DEFAULT 0,
    b_record_path   VARCHAR(255),   -- /recordings/{call_id}/b.wav

    -- 桥接
    bridge_established_at TIMESTAMP,
    bridge_broken_at      TIMESTAMP,
    bridge_duration_sec   INT DEFAULT 0,
    merged_record_path    VARCHAR(255),  -- /recordings/{call_id}/merged.mp3

    -- 损耗
    wastage_type VARCHAR(20) DEFAULT 'none' CHECK (wastage_type IN (
        'none', 'a_connected_b_failed', 'bridge_broken_early'
    )),
    wastage_cost DECIMAL(10,4) DEFAULT 0,

    -- 财务汇总
    total_cost   DECIMAL(10,4) DEFAULT 0,
    total_charge DECIMAL(10,4) DEFAULT 0,
    profit       DECIMAL(10,4) DEFAULT 0,
    charge_status VARCHAR(20) DEFAULT 'pending',

    -- 挂断原因（透传 FS hangup cause）
    hangup_cause  VARCHAR(64),   -- NORMAL_CLEARING / NO_ANSWER / USER_BUSY / etc.
    hangup_by     VARCHAR(4),    -- 'A' / 'B' / 'SYS'
    failure_reason VARCHAR(50),

    -- v3.1: 去掉 a_ringing / b_ringing 状态（代码未使用）
    status VARCHAR(20) CHECK (status IN (
        'a_dialing','a_connected',
        'b_dialing','b_connected',
        'bridged','finished','failed'
    )),

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Webhook 投递表（v3.1 从 callback_calls 拆出）
-- 独立表避免 webhook 重试时频繁 UPDATE 呼叫主表
CREATE TABLE webhook_deliveries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_id      VARCHAR(64) NOT NULL REFERENCES callback_calls(call_id),
    webhook_url  VARCHAR(255) NOT NULL,
    event_type   VARCHAR(30) NOT NULL,   -- call.finished / call.bridged / ...
    payload      JSONB NOT NULL,

    status       VARCHAR(20) DEFAULT 'pending' CHECK (status IN (
        'pending','sending','sent','failed','dlq'
    )),
    attempts     INT DEFAULT 0,
    next_retry_at TIMESTAMP,             -- 延迟队列调度用
    last_error   TEXT,
    sent_at      TIMESTAMP,

    created_at   TIMESTAMP DEFAULT NOW(),
    updated_at   TIMESTAMP DEFAULT NOW()
);

-- CDR 明细
CREATE TABLE cdr_records (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_id     VARCHAR(64) NOT NULL,
    leg         VARCHAR(1) CHECK (leg IN ('A','B')),
    user_id     UUID NOT NULL REFERENCES users(id),
    fs_uuid     VARCHAR(64),      -- FS channel UUID
    gateway     VARCHAR(64),
    carrier     VARCHAR(64),

    caller      VARCHAR(32) NOT NULL,
    callee      VARCHAR(32) NOT NULL,
    original_callee VARCHAR(32),

    start_time   TIMESTAMP NOT NULL,
    connect_time TIMESTAMP,
    end_time     TIMESTAMP,
    duration_sec  INT DEFAULT 0,
    billing_sec   INT DEFAULT 0,

    cost         DECIMAL(10,4),
    charge       DECIMAL(10,4),
    rate_per_min DECIMAL(10,4),

    hangup_cause VARCHAR(64),
    is_wastage   BOOLEAN DEFAULT FALSE,

    created_at   TIMESTAMP DEFAULT NOW(),  -- v3.1: 修复缺失

    CHECK (end_time IS NULL OR end_time >= start_time)
);

-- 费率模板
CREATE TABLE rate_plans (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(64) NOT NULL,
    description TEXT,

    a_leg_free BOOLEAN DEFAULT TRUE,
    a_leg_rate DECIMAL(10,4) DEFAULT 0,

    -- key 为精确3位前缀
    -- [{"prefix":"130","rate":0.15},{"prefix":"186","rate":0.12}]
    rates JSONB,

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW()
);

-- 黑名单
CREATE TABLE blacklisted_numbers (
    number    VARCHAR(32) PRIMARY KEY,
    reason    VARCHAR(255),
    added_by  UUID REFERENCES users(id),
    expire_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 财务流水
CREATE TABLE transactions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id),
    type         VARCHAR(20) CHECK (type IN ('recharge','charge','refund','adjust')),
    amount       DECIMAL(10,4) NOT NULL,
    balance_after DECIMAL(10,4),
    reference_id VARCHAR(64),
    description  TEXT,
    created_by   UUID REFERENCES users(id),
    created_at   TIMESTAMP DEFAULT NOW()
);

-- 系统配置
CREATE TABLE system_configs (
    key         VARCHAR(64) PRIMARY KEY,
    value       TEXT NOT NULL,
    description TEXT,
    updated_by  UUID REFERENCES users(id),
    updated_at  TIMESTAMP DEFAULT NOW()
);

INSERT INTO system_configs (key, value, description) VALUES
('bridge_broken_early_threshold_sec', '10',    '桥接秒断判定阈值（秒）'),
('webhook_max_retries',               '5',     'Webhook 最大重试次数'),
('webhook_retry_backoff_sec',         '30',    'Webhook 退避基数（秒）'),
('default_max_dialing_sec',           '45',    '默认最大振铃时间（秒）'),
('mnp_cache_ttl_hours',               '168',   '携号转网查询缓存 TTL（小时，默认7天）'),
('park_timeout_sec',                  '60',    'A路 park 超时保护（秒）'),
('web_port',                          '12345', 'Web 服务端口');

-- 审计日志
CREATE TABLE audit_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id),
    user_type    VARCHAR(20) NOT NULL,
    action       VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50),
    resource_id  VARCHAR(64),
    old_value    JSONB,
    new_value    JSONB,
    ip_address   INET,
    user_agent   TEXT,
    created_at   TIMESTAMP DEFAULT NOW()
);
```

### 5.2 关键索引

```sql
CREATE INDEX idx_calls_user_time    ON callback_calls(user_id, created_at DESC);
CREATE INDEX idx_calls_wastage      ON callback_calls(wastage_type, created_at)
    WHERE wastage_type != 'none';
CREATE INDEX idx_calls_custom_id    ON callback_calls(user_id, custom_id);
CREATE INDEX idx_calls_a_uuid       ON callback_calls(a_fs_uuid) WHERE a_fs_uuid IS NOT NULL;
CREATE INDEX idx_calls_b_uuid       ON callback_calls(b_fs_uuid) WHERE b_fs_uuid IS NOT NULL;
CREATE INDEX idx_calls_status       ON callback_calls(status, created_at DESC)
    WHERE status NOT IN ('finished','failed');

CREATE INDEX idx_cdr_user           ON cdr_records(user_id, leg, created_at);
CREATE INDEX idx_user_api_prefix    ON users(api_key_prefix) WHERE user_type = 'client';
CREATE INDEX idx_gateways_prefix    ON fs_gateways(prefix, leg, fs_status)
    WHERE prefix IS NOT NULL;
CREATE INDEX idx_gateways_leg       ON fs_gateways(leg, carrier, fs_status);
CREATE INDEX idx_transactions_user  ON transactions(user_id, created_at DESC);
CREATE INDEX idx_did_available      ON did_numbers(leg, carrier, status)
    WHERE status = 'available';

-- v3.1: webhook_deliveries 索引
CREATE INDEX idx_webhook_pending    ON webhook_deliveries(status, next_retry_at)
    WHERE status IN ('pending','failed');
CREATE INDEX idx_webhook_call_id    ON webhook_deliveries(call_id);
```

### 5.3 一致性校验（启动时）

```go
// internal/consistency/checker.go
// 系统启动时校验 fs_gateways 前缀与 rate_plans 前缀的一致性

func CheckPrefixConsistency(db *sql.DB) []string {
    var warnings []string

    // 查询所有 B 路网关的前缀
    gwPrefixes := db.Query(`SELECT DISTINCT prefix FROM fs_gateways
        WHERE leg='B' AND prefix IS NOT NULL AND fs_status='active'`)

    // 查询所有费率模板中的前缀
    for _, gw := range gwPrefixes {
        exists := db.QueryRow(`
            SELECT EXISTS(
                SELECT 1 FROM rate_plans, jsonb_array_elements(rates) r
                WHERE r->>'prefix' = $1
            )`, gw.Prefix)
        if !exists {
            warnings = append(warnings,
                fmt.Sprintf("WARN: gateway prefix %s has no matching rate_plan entry", gw.Prefix))
        }
    }
    return warnings
}
```

---

## 6. 呼叫流程与状态机

### 6.1 完整呼叫序列

```
POST /api/v1/callbacks (port 12345)
        │
        ▼
┌───────────────────────────────────┐
│ 1. 参数校验                        │
│    · 号码格式（E.164）             │
│    · 黑名单检查（DB查询）          │
└──────────────┬────────────────────┘
               │
               ▼
┌───────────────────────────────────┐
│ 2. 原子余额预扣（PG行锁）          │
│    UPDATE users                   │
│    SET balance = balance - $cost  │
│    WHERE id=$uid                  │
│      AND balance-$cost>=-credit   │
│    RETURNING balance              │
│    → RowsAffected=0 → 402         │
└──────────────┬────────────────────┘
               │
               ▼
┌───────────────────────────────────┐
│ 3. 并发槽位获取（Redis INCR）      │
│    INCR concurrent:{user_id}      │
│    → 超限 → DECR → 退款 → 429     │
│    （不使用 defer，由 finalizeCall │
│     统一释放）                     │
└──────────────┬────────────────────┘
               │
               ▼
┌───────────────────────────────────┐
│ 4. 路由选择                        │
│    A路：轮询健康 FS 网关           │
│    B路：按被叫前3位精确匹配网关    │
│    DID：从 did_numbers 取外显号    │
└──────────────┬────────────────────┘
               │
               ▼
┌───────────────────────────────────┐
│ 5. 创建 DB 记录                    │
│    status = 'a_dialing'           │
└──────────────┬────────────────────┘
               │
               ▼
┌───────────────────────────────────┐
│ 6. FSClient.OriginateALeg()       │
│    bgapi originate {vars}         │
│    sofia/gateway/a-pool-01/138xxx │
│    &park()                        │
│    FS dialplan: park_timeout=60   │
│    → 命令失败 → 全额退款 + DECR    │
└──────────────┬────────────────────┘
               │
    ┌──────────┴──────────┐
    │ A路事件（ESL）      │
    ▼                     ▼
CHANNEL_ANSWER        CHANNEL_HANGUP
(A接通)               (A未接通/park超时)
    │                     │
    │                     └─→ 全额退款
    │                          Redis DECR
    │                          status = 'failed'
    ▼
┌───────────────────────────────────┐
│ 7. 记录 a_connect_at              │
│    status = 'a_connected'         │
│    （暂不开始录音，等桥接后再录）  │
└──────────────┬────────────────────┘
               │
               ▼
┌───────────────────────────────────┐
│ 8. FSClient.OriginateBLegAndBridge│
│    bgapi originate {vars}         │
│    sofia/gateway/b-mobile-01/139xx│
│    &bridge(a_uuid)                │
│    → 命令级失败（非 ESL 事件）：   │
│      标记 wastage → 挂断A路 →     │
│      退款B路预扣 → DECR → failed  │
└──────────────┬────────────────────┘
               │
    ┌──────────┴──────────┐
    │ B路事件（ESL）      │
    ▼                     ▼
CHANNEL_BRIDGE        CHANNEL_HANGUP
(桥接成功)            (B未接通)
    │                     │
    │                     └─→ wastage = a_connected_b_failed
    │                          挂断 A 路
    │                          退款 B 路预扣
    │                          Redis DECR
    ▼
┌───────────────────────────────────┐
│ 9. 记录 bridge_established_at     │
│    status = 'bridged'             │
│    StartRecording(a_uuid, "A")    │  ← 桥接后才录音，保证对齐
│    StartRecording(b_uuid, "B")    │
│    推送 WebSocket 事件            │
└──────────────┬────────────────────┘
               │
               ▼
┌───────────────────────────────────┐
│ 10. 任意一方挂断 → CHANNEL_HANGUP │
│     status = 'finished'           │
│     计算实际费用，退还预扣差额     │
│     Redis DECR                    │
│     创建 webhook_deliveries 记录  │
│     触发异步录音合并              │
└───────────────────────────────────┘
```

### 6.2 Encore.go 事件处理器

```go
// callback/esl_handler.go

func (s *Service) RegisterESLHandlers(fs *fsclient.FSClient) {
    // A/B 路接通
    fs.OnEvent("CHANNEL_ANSWER", s.handleChannelAnswer)

    // 桥接建立
    fs.OnEvent("CHANNEL_BRIDGE", s.handleChannelBridge)

    // 任意挂断
    fs.OnEvent("CHANNEL_HANGUP", s.handleChannelHangup)
}

func (s *Service) handleChannelAnswer(event fsclient.CallEvent) {
    call, err := s.db.GetCallByID(event.CallID)
    if err != nil { return }

    switch event.Leg {
    case "A":
        // A路接通：更新状态，发起 B路
        // v3.1: 此时不录音，等 CHANNEL_BRIDGE 后再录，保证录音与通话对齐
        s.db.UpdateALegConnected(call.ID, event.UUID, time.Now())
        s.dialBLeg(call, event.UUID)

    case "B":
        // B路接通，等待 CHANNEL_BRIDGE 事件确认桥接完成
        s.db.UpdateBLegConnected(call.ID, event.UUID, time.Now())
    }
}

// dialBLeg 发起 B 路外呼，包含命令级失败的显式处理
func (s *Service) dialBLeg(call *CallbackCall, aUUID string) {
    ctx := context.Background()

    bGateway, err := s.router.SelectBGateway(call.BCallee)
    if err != nil {
        // 路由失败 → 标记损耗 → 挂断 A → 退款 → 释放并发
        s.handleBLegOriginateFailure(call, aUUID, "routing_failed", err)
        return
    }

    params := fsclient.OriginateParams{
        CallID:        call.CallID,
        UserID:        call.UserID,
        CallerID:      call.BCaller,
        Callee:        call.BCallee,
        Gateway:       bGateway.Name,
        MaxDialingSec: call.MaxDialingSec,
    }

    err = s.fs.OriginateBLegAndBridge(ctx, aUUID, params)
    if err != nil {
        // ESL 命令级失败（网关 DOWN、FS 拒绝等），不会产生 CHANNEL_HANGUP 事件
        s.handleBLegOriginateFailure(call, aUUID, "originate_failed", err)
        return
    }

    s.db.UpdateCallStatus(call.ID, "b_dialing")
}

// handleBLegOriginateFailure 处理 B 路 originate 命令级别失败
// 此场景不会产生 ESL CHANNEL_HANGUP 事件，需显式清理
func (s *Service) handleBLegOriginateFailure(call *CallbackCall, aUUID string, reason string, err error) {
    ctx := context.Background()
    log.Printf("B-leg originate failed for call %s: %s: %v", call.CallID, reason, err)

    // 1. 标记损耗
    s.markWastage(call, "a_connected_b_failed")

    // 2. 挂断 A 路
    _ = s.fs.HangupCall(ctx, aUUID, "NORMAL_CLEARING")

    // 3. 退款 B 路预扣（A 路成本保留，作为损耗）
    s.billing.RefundBLeg(call)

    // 4. 释放并发槽位
    s.redis.Decr(ctx, "concurrent:"+call.UserID)

    // 5. 更新 DB
    s.db.UpdateCallFailed(call.ID, reason, time.Now())

    // 6. 发布 Webhook
    s.publishWebhook(call, "call.failed")
}

func (s *Service) handleChannelBridge(event fsclient.CallEvent) {
    call, err := s.db.GetCallByID(event.CallID)
    if err != nil { return }

    ctx := context.Background()

    // 双路桥接成功，更新 bridge_established_at
    s.db.UpdateBridgeEstablished(event.CallID, time.Now())

    // v3.1: 桥接后才开始录音，保证 A/B 路录音时间戳对齐
    if call.Record {
        if call.AFsUUID != "" {
            s.fs.StartRecording(ctx, call.AFsUUID, call.CallID, "A")
        }
        s.fs.StartRecording(ctx, event.UUID, call.CallID, "B")
    }

    s.pubsub.Publish("call.bridged", event.CallID)
}

func (s *Service) handleChannelHangup(event fsclient.CallEvent) {
    call, err := s.db.GetCallByID(event.CallID)
    if err != nil { return }

    now := time.Now()

    switch {
    case call.Status == "a_dialing":
        // A路未接通就挂断
        s.finalizeCall(call, event.HangupCause, now)

    case call.Status == "a_connected" && call.BConnectAt == nil:
        // A 通了但 B 没通 → 损耗（ESL 事件路径）
        s.markWastage(call, "a_connected_b_failed")
        s.finalizeCall(call, event.HangupCause, now)

    case call.Status == "b_dialing" && event.Leg == "B":
        // B路振铃中挂断
        s.markWastage(call, "a_connected_b_failed")
        // 挂断 A 路
        _ = s.fs.HangupCall(context.Background(), call.AFsUUID, "NORMAL_CLEARING")
        s.finalizeCall(call, event.HangupCause, now)

    case call.Status == "bridged":
        // 正常通话结束
        threshold := s.config.GetInt("bridge_broken_early_threshold_sec")
        bridgeDuration := int(now.Sub(*call.BridgeEstablishedAt).Seconds())
        if bridgeDuration < threshold {
            s.markWastage(call, "bridge_broken_early")
        }
        s.finalizeCall(call, event.HangupCause, now)
    }
}

func (s *Service) finalizeCall(call *CallbackCall, cause string, endTime time.Time) {
    ctx := context.Background()

    // 1. 计算实际费用
    billing := s.billing.Calculate(call)

    // 2. 原子结算：退还预扣差额，写入 transactions
    s.billing.Settle(call.UserID, billing)

    // 3. 释放并发槽位（统一在此处释放，创建时不 defer）
    s.redis.Decr(ctx, "concurrent:"+call.UserID)

    // 4. 更新 DB
    s.db.FinalizeCall(call.ID, billing, cause, endTime)

    // 5. 创建 webhook_deliveries 记录（异步 Worker 轮询处理）
    s.publishWebhook(call, "call.finished")

    // 6. 触发录音合并（异步）
    if call.Record {
        s.pubsub.Publish("recording.merge", call.CallID)
    }
}

// publishWebhook 创建 webhook 投递记录到 webhook_deliveries 表
func (s *Service) publishWebhook(call *CallbackCall, eventType string) {
    webhookURL := call.WebhookURL
    if webhookURL == "" {
        // 取用户默认 webhook_url
        user, _ := s.db.GetUser(call.UserID)
        webhookURL = user.WebhookURL
    }
    if webhookURL == "" {
        return
    }

    payload := s.buildWebhookPayload(call, eventType)
    s.db.InsertWebhookDelivery(call.CallID, webhookURL, eventType, payload)
}
```

### 6.3 hangup_cause 与 failure_reason 映射

FreeSWITCH 原始 hangup cause 会透传到 DB，同时映射为业务层 failure_reason：

| FS Hangup Cause | failure_reason | 说明 |
|-----------------|----------------|------|
| `NO_ANSWER` | `no_answer` | 超时未接 |
| `USER_BUSY` | `busy` | 用户忙 |
| `NORMAL_CLEARING` | `null` | 正常挂断 |
| `CALL_REJECTED` | `rejected` | 主动拒接 |
| `NO_ROUTE_DESTINATION` | `routing_failed` | 路由失败 |
| `RECOVERY_ON_TIMER_EXPIRE` | `timeout` | SIP 超时 |
| `NETWORK_OUT_OF_ORDER` | `network_error` | 网络故障 |
| N/A（ESL 命令失败） | `originate_failed` | B 路 originate 命令级失败 |

---

## 7. API 接口定义

### 7.1 认证方式

- 管理员：`Cookie: admin_session=<JWT>`
- 客户（Web）：`Authorization: Bearer <JWT>`
- 客户（程序对接）：`X-API-Key: ak_live_xxx`（IP 白名单校验）

所有 API 端点监听在 **port 12345**。

### 7.2 创建回拨

`POST http://<host>:12345/api/v1/callbacks`

```json
// 请求体
{
  "a_callee":       "13800138000",
  "b_callee":       "13900139000",
  "caller_id_a":    "4001234567",    // 可选，A路外显，默认取 DID 池
  "caller_id_b":    "4001234567",    // 可选，B路外显，默认与A路相同
  "custom_id":      "order_12345",
  "webhook_url":    "https://cb.example.com/webhook",
  "record":         true,
  "max_dialing_sec": 45,
  "metadata": {
    "campaign_id": "c123",
    "agent_id":    "a456"
  }
}

// 响应 201
{
  "call_id":        "call_abc123",
  "status":         "a_dialing",
  "estimated_cost": 0.50,
  "created_at":     "2026-03-09T10:00:00Z"
}
```

错误码：`400` 参数错误 / `402` 余额不足 / `403` 黑名单 / `429` 超限（详见第 16 章错误码规范）

### 7.3 查询呼叫状态

`GET http://<host>:12345/api/v1/callbacks/{call_id}`

```json
{
  "call_id":  "call_abc123",
  "status":   "bridged",
  "a_leg": {
    "callee":      "13800138000",
    "status":      "connected",
    "gateway":     "a-pool-01",
    "connect_at":  "2026-03-09T10:00:15Z",
    "duration_sec": 120
  },
  "b_leg": {
    "callee":      "13900139000",
    "status":      "connected",
    "gateway":     "b-mobile-01",
    "carrier":     "mobile",
    "connect_at":  "2026-03-09T10:00:18Z",
    "duration_sec": 117
  },
  "bridge_duration_sec": 117,
  "wastage_type":  "none",
  "total_charge":  0.45,
  "record_urls": {
    "a":      "https://rec.example.com/call_abc123/a.wav",
    "b":      "https://rec.example.com/call_abc123/b.wav",
    "merged": "https://rec.example.com/call_abc123/merged.mp3"
  }
}
```

### 7.4 Webhook 回调格式

触发时机：状态变更（a_connected / b_connected / bridged / finished / failed）

```json
{
  "event":     "call.finished",
  "timestamp": "2026-03-09T10:02:30Z",
  "data": {
    "call_id":           "call_abc123",
    "custom_id":         "order_12345",
    "status":            "finished",
    "a_callee":          "13800138000",
    "b_callee":          "13900139000",
    "a_duration_sec":    135,
    "b_duration_sec":    132,
    "bridge_duration_sec": 132,
    "wastage_type":      "none",
    "hangup_cause":      "NORMAL_CLEARING",
    "hangup_by":         "B",
    "total_cost":        0.30,
    "total_charge":      0.45,
    "failure_reason":    null,
    "metadata": {
      "campaign_id": "c123",
      "agent_id":    "a456"
    }
  }
}
```

### 7.5 其他接口（摘要）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/callbacks` | 列表查询（分页、筛选） |
| `DELETE` | `/api/v1/callbacks/{call_id}` | 强制挂断（返回 204） |
| `GET` | `/api/v1/callbacks/{call_id}/recording` | 获取录音下载链接 |
| `GET` | `/api/admin/gateways` | 网关状态列表 |
| `PUT` | `/api/admin/gateways/{id}/status` | 手动上下线网关 |
| `POST` | `/api/admin/users` | 创建客户 |
| `POST` | `/api/admin/users/{id}/recharge` | 充值 |
| `GET` | `/api/admin/wastage/report` | 损耗分析报表 |
| `GET` | `/api/admin/webhook/dlq` | Webhook 死信队列（v3.1） |
| `POST` | `/api/admin/webhook/dlq/{id}/retry` | 手动重试 DLQ 记录（v3.1） |

所有接口均通过 `http://<host>:12345` 访问。

---

## 8. 计费与损耗模型

### 8.1 费率优先级

```
用户级 a_leg_rate（users.a_leg_rate > 0）
    ↓ 否则
费率模板（rate_plans.a_leg_rate）
    ↓
B路按被叫前缀查 rate_plans.rates JSONB
```

### 8.2 原子余额控制

**余额预扣（PG 行锁）**：

```go
// 单次 SQL，原子操作，无 TOCTOU 竞态
result, err := db.Exec(`
    UPDATE users
    SET    balance = balance - $1,
           updated_at = NOW()
    WHERE  id = $2
      AND  (balance - $1) >= -credit_limit
    RETURNING balance
`, estimatedCost, userID)

if result.RowsAffected() == 0 {
    return ErrInsufficientBalance
}
// 同一事务内写 transactions 流水（type='charge', amount=-estimatedCost）
```

**并发槽位（Redis INCR）**：

```go
key := "concurrent:" + userID
n, _ := redis.Incr(ctx, key)
redis.Expire(ctx, key, 24*time.Hour)  // 防泄漏

if n > int64(user.ConcurrentLimit) {
    redis.Decr(ctx, key)
    // 退还已预扣的余额
    billing.Refund(userID, estimatedCost, "concurrent_limit_exceeded")
    return ErrConcurrentLimitExceeded  // → 429
}
// 注意：不使用 defer Decr！并发槽位由 finalizeCall() 统一释放
```

### 8.3 损耗计算

```go
func (b *BillingService) CalculateWastage(call *CallbackCall) {
    threshold := b.config.GetInt("bridge_broken_early_threshold_sec")

    switch {
    case call.AConnectAt != nil && call.BConnectAt == nil:
        // A 通了，B 没通：A 路成本白费
        call.WastageType = "a_connected_b_failed"
        call.WastageCost = call.ACost
        // 收费策略：默认只收 A 路成本，不收 B 路（B 根本没打通）
        call.TotalCharge = call.ACharge

    case call.BridgeEstablishedAt != nil && call.BridgeDurationSec < threshold:
        // 桥接后秒断
        call.WastageType = "bridge_broken_early"
        call.WastageCost = call.ACost + call.BCost

    default:
        call.WastageType = "none"
        call.WastageCost = 0
    }

    call.Profit = call.TotalCharge - call.TotalCost
}
```

### 8.4 录音合并（异步）

```go
// recording/merger.go（由 Encore PubSub 触发）

func MergeRecordings(callID string) error {
    aPath := fmt.Sprintf("/recordings/%s/a.wav", callID)
    bPath := fmt.Sprintf("/recordings/%s/b.wav", callID)
    outPath := fmt.Sprintf("/recordings/%s/merged.mp3", callID)

    // A路左声道，B路右声道，双声道混轨
    cmd := exec.Command("ffmpeg",
        "-i", aPath,
        "-i", bPath,
        "-filter_complex",
        "[0:a]aformat=channel_layouts=mono,pan=stereo|c0=c0[a];" +
        "[1:a]aformat=channel_layouts=mono,pan=stereo|c1=c0[b];" +
        "[a][b]amerge[out]",
        "-map", "[out]",
        "-b:a", "128k",
        outPath,
    )

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("merge failed for %s: %w", callID, err)
    }

    // 合并完成后立即上传到对象存储
    s3URL, err := s3.Upload(outPath, fmt.Sprintf("recordings/%s/merged.mp3", callID))
    if err != nil {
        log.Printf("S3 upload failed for %s: %v, local file retained", callID, err)
    }

    // 同时上传 A/B 路原始录音
    s3.Upload(aPath, fmt.Sprintf("recordings/%s/a.wav", callID))
    s3.Upload(bPath, fmt.Sprintf("recordings/%s/b.wav", callID))

    return db.UpdateMergedRecordPath(callID, s3URL)
}
```

### 8.5 Webhook 异步重试（非阻塞延迟队列）

```go
// webhook/worker.go（Encore PubSub Consumer / 定时轮询 webhook_deliveries）
// v3.1: 改为非阻塞模式，不在 consumer 内 sleep

func (w *WebhookWorker) ProcessPending(ctx context.Context) {
    // 轮询 webhook_deliveries 中 status='pending' 或 (status='failed' AND next_retry_at <= NOW())
    deliveries, err := w.db.GetPendingWebhooks(ctx, 50)  // 每批最多处理 50 条
    if err != nil { return }

    maxRetries := w.config.GetInt("webhook_max_retries")
    baseSec    := w.config.GetInt("webhook_retry_backoff_sec")

    for _, d := range deliveries {
        err := w.send(d)
        if err == nil {
            w.db.MarkWebhookSent(d.ID, time.Now())
            continue
        }

        d.Attempts++
        if d.Attempts >= maxRetries {
            // 超过最大重试 → 标记 DLQ
            w.db.MarkWebhookDLQ(d.ID, err.Error())
            continue
        }

        // 计算下次重试时间，写回 DB（非阻塞）
        backoff := time.Duration(baseSec) * (1 << d.Attempts) * time.Second
        nextRetry := time.Now().Add(backoff)
        w.db.UpdateWebhookRetry(d.ID, d.Attempts, nextRetry, err.Error())
    }
}

// 由 Encore Cron 每 10 秒触发一次
//encore:api private method=POST path=/internal/webhook/process
func (w *WebhookWorker) CronTrigger(ctx context.Context) error {
    w.ProcessPending(ctx)
    return nil
}
```

---

## 9. 路由引擎

### 9.1 A路路由（轮询）

```go
// routing/router.go

func (r *Router) SelectAGateway() (*FSGateway, error) {
    gateways, err := r.db.GetActiveGateways(leg="A", carrier="any")
    if err != nil || len(gateways) == 0 {
        return nil, ErrNoAvailableGateway
    }
    // 简单轮询（可换成加权轮询）
    selected := gateways[r.aCounter.Add(1) % uint64(len(gateways))]
    return selected, nil
}
```

### 9.2 B路路由（精确前缀匹配）

```go
func (r *Router) SelectBGateway(callee string) (*FSGateway, error) {
    if len(callee) < 3 {
        return nil, ErrInvalidNumber
    }
    prefix := callee[:3]

    // 精确前缀查 DB（一次查询）
    gateways, err := r.db.GetActiveGatewaysByPrefix(leg="B", prefix=prefix)
    if err != nil || len(gateways) == 0 {
        // 降级：取同运营商其他网关
        carrier := r.inferCarrier(prefix)
        gateways, err = r.db.GetActiveGateways(leg="B", carrier=carrier)
        if err != nil || len(gateways) == 0 {
            return nil, ErrNoAvailableGateway
        }
    }

    return gateways[0], nil // 已按 priority 排序
}
```

### 9.3 MNP 携号转网说明

中国自2019年全面实施携号转网，按前缀3位判断运营商存在约 3-5% 的误判率。

**处理策略**（按成熟度分阶段）：

- **Phase 1（MVP）**：按前缀路由，接受少量误判。在 `fs_gateways` 表预留 `mnp_supported BOOLEAN DEFAULT FALSE` 字段，便于后期标记。
- **Phase 2**：接入工信部 HLR 查询接口或第三方号码归属数据库，结果缓存至 Redis（TTL 由 `system_configs.mnp_cache_ttl_hours` 控制，默认7天）。
- **Phase 3**：定期同步本地号码归属数据库（CSV），全量本地查询，零接口依赖。

### 9.4 号码变换

```go
// 简单变换规则：仅支持 strip + add
func ApplyTransform(gw *FSGateway, number string) string {
    if gw.TransformStrip > 0 && len(number) > gw.TransformStrip {
        number = number[gw.TransformStrip:]
    }
    if gw.TransformAdd != "" {
        number = gw.TransformAdd + number
    }
    return number
}
```

---

## 10. 双模式权限体系

### 10.1 管理员模式

鉴权：JWT Cookie（HttpOnly, Secure）

数据范围：全平台

功能：
- 用户管理（开户、充值、冻结、查看 API Key）
- 网关管理（新增/编辑/上下线 FS 网关，配置容灾关系）
- DID 管理（导入外显号、分配专属号码）
- 财务管理（全平台流水、退款、毛利分析）
- 全局监控（实时通话、强制挂断）
- 损耗分析（全平台损耗排名、趋势分析）
- Webhook DLQ 管理（查看死信、手动重试）
- 系统配置（秒断阈值、Webhook 重试策略等）

### 10.2 客户模式

鉴权：JWT Token 或 API Key（IP 白名单）

数据范围：`WHERE user_id = current_user_id`（强制注入，不可绕过）

功能：
- 发起回拨（Web 表单 / API）
- 话单查询（含录音播放/下载）
- 损耗分析（自己的损耗率、失败原因分布）
- 财务中心（余额、流水、费率查询、发票申请）
- API 集成（API Key 管理、Webhook 配置、IP 白名单）

---

## 11. Admin Dashboard 功能结构

```
Admin Dashboard (http://<host>:12345)
├─ Overview（运营大盘）
│  ├─ 实时看板：A路并发、B路并发、今日收入、今日损耗成本
│  ├─ 今日指标：API请求数、桥接成功率、平均损耗率、活跃客户数
│  ├─ 告警卡片：网关 DOWN、余额耗尽客户、Webhook DLQ 积压
│  └─ 快捷操作：创建客户、紧急全局限流、强制挂断活跃通话
│
├─ User Management（客户管理）
│  ├─ 客户列表（余额、信用额度、日限、状态）
│  ├─ 客户详情（财务流水、呼叫统计、损耗分析、API Key）
│  ├─ 开户（设置初始余额、分配费率模板）
│  └─ 资金操作（充值、扣款、信用额度调整）
│
├─ Gateway Management（网关管理）
│  ├─ A路网关池（轮询配置、健康状态、并发使用量）
│  ├─ B路网关（按运营商分组，前缀绑定，容灾配置）
│  ├─ 网关健康监控（FS ping 状态实时同步）
│  └─ 测试外呼（选定网关发起测试呼叫，验证路由）
│
├─ DID Management（外显号码池）
│  ├─ 号码列表（状态、运营商、归属客户、使用率）
│  ├─ 批量导入（CSV/Excel）
│  └─ 分配管理（设为专属 / 放回公共池）
│
├─ Calls & CDR（通话管理）
│  ├─ 实时通话监控（正在进行的呼叫列表，支持强制挂断）
│  ├─ 全量话单查询（跨用户、按状态/损耗类型/时间筛选）
│  ├─ 损耗分析中心
│  │   ├─ 平台损耗趋势（日/周/月）
│  │   ├─ 客户损耗排名
│  │   ├─ B路失败原因分布（hangup_cause 统计）
│  │   └─ A路等待时长分布（优化建议）
│  └─ 录音管理（查询、播放、下载）
│
├─ Financial Center（财务中心）
│  ├─ 对账中心（按客户/按网关/按运营商）
│  ├─ 全平台流水
│  ├─ 费率模板管理
│  ├─ 毛利分析（分客户、分网关盈亏）
│  └─ Webhook DLQ 管理（查看失败记录、手动重试）
│
├─ Compliance（合规风控）
│  ├─ 全局黑名单
│  ├─ 风控规则配置
│  └─ 审计日志
│
├─ Operations（运维工具）
│  ├─ FreeSWITCH 状态（通道数、网关状态、FS 版本）
│  ├─ ESL 连接状态（Encore 到 FS 的连接健康度）
│  ├─ 系统健康监控（DB 连接池、Redis 内存、磁盘录音空间）
│  └─ 告警配置
│
└─ Settings（系统设置）
   ├─ 全局参数（system_configs 可视化编辑）
   ├─ 管理员资料
   └─ 安全设置
```

---

## 12. Client Portal 功能结构

```
Client Portal (http://<host>:12346)
├─ Dashboard（仪表盘）
│  ├─ 今日概览：呼叫总数、成功率、损耗率、消费金额、余额
│  ├─ 实时数据：当前并发（WebSocket 推送）
│  └─ 快捷操作：发起回拨、查看 API 文档
│
├─ Callback Operations（回拨操作）
│  ├─ 发起回拨（表单，支持自定义外显号、custom_id、metadata）
│  ├─ 批量导入（Excel 上传号码对，异步处理）
│  ├─ 进行中的通话（实时列表，支持自助挂断）
│  └─ 测试呼叫
│
├─ CDR & Records（话单）
│  ├─ 话单查询（时间、号码、状态、损耗标记筛选）
│  ├─ 话单导出（Excel/CSV）
│  ├─ 详情页（A/B路分离展示、hangup_cause、录音播放）
│  └─ 录音下载（A路、B路、合并录音）
│
├─ Wastage Analysis（损耗分析）
│  ├─ 损耗概览（近7天趋势）
│  ├─ 损耗明细（A通B不通的记录列表）
│  ├─ B路失败原因分布（饼图）
│  └─ A路等待时长分布（优化参考）
│
├─ Financial Center（财务）
│  ├─ 余额与信用额度
│  ├─ 消费流水
│  ├─ 费率查询（A路费率、B路按前缀费率表）
│  └─ 发票申请
│
├─ API & Integration（API集成）
│  ├─ API Key（查看 prefix、重置密钥）
│  ├─ Webhook 配置（默认回调 URL、测试发送）
│  ├─ Webhook 日志（最近100条发送记录）
│  ├─ IP 白名单
│  └─ API 文档（Encore 自动生成）
│
└─ Settings（账户设置）
   ├─ 基本资料
   ├─ 外显号码池（查看分配的 DID）
   ├─ 安全设置（修改密码、登录日志）
   └─ 通知设置
```

---

## 13. 前端技术栈

### 13.1 技术组合

| 技术 | 版本 | 用途 |
|------|------|------|
| React | 18.3+ | UI 框架 |
| Vite | 5.0+ | 构建工具 |
| TypeScript | 5.3+ | 类型安全，strict mode |
| shadcn/ui | 2.0+ | 组件库（Radix UI + CSS Variables） |
| MagicUI | 最新 | 动效组件（损耗分析看板） |
| TanStack Query | 5.0+ | REST 数据获取 + 缓存 |
| Zustand | 4.5+ | 轻量全局状态 |
| Tailwind CSS | 4.0+ | 样式 |
| Recharts | 2.10+ | 损耗趋势图、财务报表 |
| React Router | 6.x | 路由 |

### 13.2 Monorepo 结构（Turborepo）

```
bos3000-frontend/
├── apps/
│   ├── admin/          # 管理后台（port 12345）
│   └── client/         # 客户前台（port 12346）
├── packages/
│   ├── ui/             # 共享 shadcn/ui + MagicUI 组件
│   ├── shared/         # 共享类型（来自 Encore 生成的 OpenAPI schema）
│   └── api-client/     # Encore 自动生成的类型安全 API 客户端
└── turbo.json
```

### 13.3 环境配置

```typescript
// packages/shared/src/config.ts
// 统一从环境变量读取 API 地址，不硬编码端口

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL
    ?? `http://localhost:12345`;

export const WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL
    ?? `ws://localhost:12345`;
```

### 13.4 关键组件示例

**实时通话监控（WebSocket + shadcn Table）**：

```tsx
// apps/admin/src/components/calls/live-monitor.tsx
import { useWebSocket } from "@/hooks/use-websocket";
import { WS_BASE_URL } from "@bos3000/shared/config";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

const STATUS_BADGE: Record<string, string> = {
  a_dialing:    "secondary",
  a_connected:  "outline",
  b_dialing:    "secondary",
  bridged:      "default",   // 绿色
};

export function LiveCallMonitor() {
  const { data: calls } = useWebSocket<LiveCall[]>(`${WS_BASE_URL}/ws/admin/calls`);

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Call ID</TableHead>
          <TableHead>客户</TableHead>
          <TableHead>A路</TableHead>
          <TableHead>B路</TableHead>
          <TableHead>状态</TableHead>
          <TableHead>桥接时长</TableHead>
          <TableHead>操作</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {calls?.map(call => (
          <TableRow key={call.call_id}>
            <TableCell className="font-mono text-xs">{call.call_id}</TableCell>
            <TableCell>{call.user_name}</TableCell>
            <TableCell>{call.a_callee}</TableCell>
            <TableCell>{call.b_callee}</TableCell>
            <TableCell>
              <Badge variant={STATUS_BADGE[call.status] as any}>
                {call.status}
              </Badge>
            </TableCell>
            <TableCell>{call.bridge_duration_sec}s</TableCell>
            <TableCell>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => adminAPI.calls.hangup(call.call_id)}
              >
                挂断
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
```

**损耗分析趋势图**：

```tsx
// apps/admin/src/components/wastage/wastage-chart.tsx
import { LineChart, Line, XAxis, YAxis, Tooltip, Legend, ResponsiveContainer } from "recharts";
import { useQuery } from "@tanstack/react-query";

export function WastageTrendChart({ days = 7 }: { days?: number }) {
  const { data } = useQuery({
    queryKey: ["wastage-trend", days],
    queryFn: () => adminAPI.wastage.trend({ days }),
    refetchInterval: 60_000,
  });

  return (
    <div className="rounded-lg border bg-card p-6">
      <h3 className="mb-4 font-semibold">损耗趋势（近{days}天）</h3>
      <ResponsiveContainer width="100%" height={280}>
        <LineChart data={data}>
          <XAxis dataKey="date" />
          <YAxis yAxisId="cost" orientation="left" tickFormatter={v => `¥${v}`} />
          <YAxis yAxisId="rate" orientation="right" tickFormatter={v => `${v}%`} />
          <Tooltip />
          <Legend />
          <Line
            yAxisId="cost"
            type="monotone"
            dataKey="wastage_cost"
            name="损耗金额(¥)"
            stroke="hsl(var(--destructive))"
            strokeWidth={2}
          />
          <Line
            yAxisId="rate"
            type="monotone"
            dataKey="wastage_rate"
            name="损耗率(%)"
            stroke="hsl(var(--warning))"
            strokeWidth={2}
            strokeDasharray="4 4"
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
```

---

## 14. 非功能性需求与 SLA

### 14.1 性能目标

| 指标 | 目标 |
|------|------|
| API 响应时间（P99） | < 200ms（创建回拨） |
| 最大并发通话 | 1000+（单 FS 实例） |
| WebSocket 推送延迟 | < 500ms |
| 录音合并完成时间 | 通话结束后 < 5 分钟 |
| Webhook 首次投递延迟 | < 3 秒 |

### 14.2 可用性

| 指标 | 目标 |
|------|------|
| API 可用性 | 99.9%（月度） |
| FreeSWITCH 可用性 | 99.95%（双机热备） |
| ESL 自动切换时间 | < 5 秒（进行中通话断，新通话立即恢复） |

### 14.3 数据保留策略

| 数据类型 | 保留周期 | 存储位置 |
|----------|----------|----------|
| callback_calls 记录 | 永久（归档至分区表） | PostgreSQL |
| cdr_records | 永久 | PostgreSQL |
| 录音文件（原始） | 热存储 7 天 → 冷存储 180 天 | S3/OSS |
| 录音文件（合并） | 热存储 30 天 → 冷存储 365 天 | S3/OSS |
| 本地录音文件 | 合并+上传后保留 24 小时 | FS 本地磁盘 |
| 审计日志 | 永久 | PostgreSQL |
| Webhook DLQ 记录 | 90 天 | PostgreSQL |
| transactions 流水 | 永久 | PostgreSQL |

### 14.4 灾难恢复

| 指标 | 目标 |
|------|------|
| RTO（恢复时间目标） | < 30 分钟 |
| RPO（恢复点目标） | < 5 分钟（PG 流复制） |
| 数据库备份频率 | 每日全量 + 持续 WAL 归档 |
| 录音文件丢失窗口 | 最多丢失上传前的 24 小时本地文件 |

### 14.5 日志保留

| 日志类型 | 保留时长 |
|----------|----------|
| Encore 应用日志 | 30 天 |
| FreeSWITCH SIP 日志 | 14 天 |
| ESL 事件日志 | 7 天（高频） |
| 审计操作日志 | 永久（DB） |

---

## 15. 部署与运维

### 15.1 本地开发（Local-First）

```yaml
# docker-compose.dev.yml
# Encore 内置管理 PostgreSQL 和 Redis，这里只需启动 FreeSWITCH
version: '3.8'

services:
  freeswitch:
    image: signalwire/freeswitch:1.10
    environment:
      - ESL_PASSWORD=${ESL_PASSWORD:-bos3000dev}
    volumes:
      - ./freeswitch/conf:/etc/freeswitch
      - ./freeswitch/docker-entrypoint.sh:/docker-entrypoint.sh:ro  # 环境变量注入
      - ./recordings:/recordings
    entrypoint: ["/docker-entrypoint.sh"]
    ports:
      - "5080:5080/udp"   # SIP (外呼)
      - "8021:8021/tcp"   # ESL（仅本地）
      - "16384-32768:16384-32768/udp"  # RTP 媒体

  # ffmpeg 合并录音时需要可执行文件（开发环境本地安装即可）
```

**FS 容器 entrypoint 脚本**（处理环境变量注入）：

```bash
#!/bin/bash
# freeswitch/docker-entrypoint.sh
# 将环境变量写入 FS 可识别的 vars_env.xml

cat > /etc/freeswitch/vars_env.xml <<EOF
<include>
  <X-PRE-PROCESS cmd="set" data="esl_password=${ESL_PASSWORD}"/>
</include>
EOF

exec /usr/bin/freeswitch -nonat -nf
```

**启动流程**：

```bash
# 1. 启动 FreeSWITCH
docker compose -f docker-compose.dev.yml up -d

# 2. 启动 Encore（自动创建 PG + Redis + 运行迁移，端口 12345）
ENCORE_PORT=12345 encore run

# 3. 验证 ESL 连接
docker exec freeswitch fs_cli -x "status"

# 4. 启动前端
VITE_API_BASE_URL=http://localhost:12345 pnpm --filter admin dev -- --port 12345
VITE_API_BASE_URL=http://localhost:12345 pnpm --filter client dev -- --port 12346
```

### 15.2 生产拓扑

```
[ Encore Cloud / K8s ]              [ 物理机 / 高性能 VM ]
┌────────────────────────┐          ┌───────────────────────────────┐
│ Encore Services (:12345)│  ESL/TCP │ FreeSWITCH 集群               │
│ · callback service     │◄────────►│ · FS-01 (主)                  │
│ · routing service      │          │ · FS-02 (备，热备）            │
│ · billing service      │          │ · 共享 /recordings NFS 存储    │
│ · webhook worker       │          └───────────────────────────────┘
│ · recording merger     │
└──────────┬─────────────┘
           │
┌──────────▼─────────────┐
│ PostgreSQL (Encore)    │
│ Redis (Encore)         │
└────────────────────────┘

[ 对象存储 ]
┌────────────────────────┐
│ S3 / OSS               │
│ · 录音文件归档          │
│ · 合并录音存储          │
│ · 7天后转冷存储         │
└────────────────────────┘
```

### 15.3 FreeSWITCH 高可用

两台 FS 实例通过共享 NFS 存储录音文件，Encore 侧 FSClient Manager 维护双连接并通过 `healthCheckLoop` 实时感知每台实例的健康状态。`Pick()` 方法自动跳过不健康实例，任意一台故障时新建通话立即切换到健康实例（进行中的通话会断，但新建通话恢复时间 < 5s）。

### 15.4 录音文件生命周期

```
FS 本地录音 (/recordings/{call_id}/)
    │
    ├─ 通话结束 → 异步合并 (ffmpeg)
    │
    ├─ 合并完成 → 立即上传 S3/OSS
    │
    ├─ 上传成功 → 更新 DB record_path 为 S3 URL
    │
    └─ 本地文件保留 24h 后清理 (cron)
        └─ 若上传失败，本地文件延长保留至 72h
```

### 15.5 监控指标

```yaml
# Encore 自动暴露（Prometheus 格式）
encore_requests_total{service="callback", route="CreateCallback"}
encore_errors_total{type="insufficient_balance"}
encore_errors_total{type="concurrent_limit_exceeded"}

# 自定义业务指标
bos3000_active_calls_total{leg="A"}
bos3000_active_calls_total{leg="B"}
bos3000_bridged_calls_total
bos3000_wastage_cost_total{type="a_connected_b_failed"}
bos3000_wastage_cost_total{type="bridge_broken_early"}
bos3000_hangup_cause_total{cause="NO_ANSWER", leg="B"}
bos3000_webhook_delivery_total{result="sent|failed|dlq"}
bos3000_esl_connection_status{host="fs-01"}       # 1=healthy, 0=unhealthy
bos3000_esl_health_check_latency_ms{host="fs-01"} # 健康探测延迟
bos3000_recording_merge_duration_seconds           # ffmpeg 合并耗时
bos3000_recording_upload_duration_seconds           # S3 上传耗时
bos3000_local_recording_disk_usage_bytes            # 本地录音磁盘占用
```

---

## 16. 错误码规范

### 16.1 HTTP 状态码映射

| HTTP Status | 场景 |
|-------------|------|
| `200` | 查询成功 |
| `201` | 创建回拨成功 |
| `204` | 挂断成功（无响应体） |
| `400` | 参数校验失败 |
| `401` | 未认证 |
| `402` | 余额不足 |
| `403` | 权限不足 / 号码黑名单 |
| `404` | 资源不存在 |
| `409` | 冲突（如 custom_id 重复） |
| `429` | 并发超限 / 日限超限 |
| `500` | 服务端内部错误 |
| `502` | FreeSWITCH 不可达 |
| `503` | 服务暂不可用（所有 FS 实例不健康） |

### 16.2 错误响应体结构

```json
{
  "error": {
    "code":    "INSUFFICIENT_BALANCE",
    "message": "Account balance insufficient for estimated call cost",
    "details": {
      "current_balance": 0.12,
      "estimated_cost":  0.50,
      "credit_limit":    0.00
    }
  }
}
```

### 16.3 业务错误码枚举

| error.code | HTTP | 说明 |
|------------|------|------|
| `INVALID_PARAMS` | 400 | 参数格式错误（附 field 信息） |
| `INVALID_NUMBER_FORMAT` | 400 | 号码不符合 E.164 格式 |
| `NUMBER_BLACKLISTED` | 403 | 号码在黑名单中 |
| `INSUFFICIENT_BALANCE` | 402 | 余额不足 |
| `CONCURRENT_LIMIT_EXCEEDED` | 429 | 并发超限 |
| `DAILY_LIMIT_EXCEEDED` | 429 | 日呼叫量超限 |
| `NO_AVAILABLE_GATEWAY` | 502 | 无可用网关（A路或B路） |
| `ALL_FS_UNHEALTHY` | 503 | 所有 FreeSWITCH 实例不健康 |
| `ORIGINATE_FAILED` | 500 | ESL originate 命令失败 |
| `CALL_NOT_FOUND` | 404 | 呼叫记录不存在 |
| `DUPLICATE_CUSTOM_ID` | 409 | custom_id 已存在 |
| `USER_SUSPENDED` | 403 | 用户账户已冻结 |
| `UNAUTHORIZED` | 401 | 认证失败 |
| `FORBIDDEN` | 403 | 权限不足 |
| `IP_NOT_WHITELISTED` | 403 | 请求 IP 不在白名单 |

---

## 17. 开发阶段与验收标准

### 17.1 开发阶段

**Phase 1a：后端核心（Week 1-2）—— 不依赖真实 FreeSWITCH**

- DB Schema + sqlc 生成（含 webhook_deliveries 表、cdr_records.created_at 修复）
- 启动时前缀一致性校验（fs_gateways vs rate_plans）
- 双模式鉴权（Admin JWT Cookie / Client JWT / API Key）
- FSClient 封装（含 eslgo 集成 + Mock 模式 + 健康探测）
- 路由引擎（A路加权轮询、B路精确前缀）
- 创建回拨 API（含原子余额/并发控制，并发槽位不使用 defer）
- 损耗计算逻辑（单测覆盖所有 wastage_type 分支 + originate 失败路径）
- Webhook Worker（延迟队列非阻塞重试、DLQ）
- 错误码规范实现（统一错误响应体）
- **全程使用 Mock FSClient，ESL 事件通过单测模拟**

**Phase 1b：FreeSWITCH 集成（Week 3-4）**

- 真实 FS 环境搭建（docker-compose.dev.yml + entrypoint 环境变量注入）
- eslgo 接入 + FSClient 对接真实 ESL（含 healthCheckLoop）
- OriginateALeg → park（含 park_timeout=60） → OriginateBLegAndBridge 端到端联调
- B 路 originate 命令级失败的显式处理验证
- CHANNEL_ANSWER / CHANNEL_HANGUP / CHANNEL_BRIDGE 事件处理
- 录音（uuid_record，在 CHANNEL_BRIDGE 后开始）+ 异步 ffmpeg 合并
- 录音上传 S3/OSS + 本地清理
- 端到端真实通话验证（含录音文件生成与对齐验证）

**Phase 2a：核心 Portal（Week 5-6）**

- Admin Dashboard 核心模块：Overview、User Management、Gateway Management、Calls & CDR
- Client Portal 核心模块：Dashboard、Callback Operations、CDR & Records、Financial Center
- WebSocket 实时通话推送

**Phase 2b：分析与管理 Portal（Week 7-8）**

- Admin Dashboard 高级模块：损耗分析中心、Financial Center（毛利分析）、DID Management
- Client Portal 高级模块：Wastage Analysis、API & Integration
- Admin：Compliance（黑名单、审计日志）、Operations（FS/ESL 状态）、Settings
- Client：Settings
- 批量导入功能
- 在线测试呼叫（Admin 发起测试，验证指定网关路由）
- Webhook DLQ 管理界面

**Phase 3：优化与加固（Week 9-10）**

- 性能压测（1000+ 并发通话）
- FreeSWITCH 高可用（双机热备 + FSClient Manager 健康感知自动切换）
- MNP 携号转网查询接入（视业务需求决策）
- 录音文件生命周期管理（热→冷存储自动迁移、本地清理 cron）
- 告警配置与运维 runbook
- 全链路压测报告与容量规划

### 17.2 验收标准

| # | 验收项 | 通过标准 |
|---|--------|---------|
| 1 | API 发起 | `POST http://<host>:12345/api/v1/callbacks` 成功创建，立即发起 A-Leg 外呼 |
| 2 | 双呼流程 | A 接通后自动发起 B 路，B 接通后双路桥接，通话可正常进行 |
| 3 | Park 超时保护 | A 路 park 超过 60s 未桥接时自动挂断，不泄漏通道 |
| 4 | B 路命令失败 | B 路 originate 命令级失败时，正确标记损耗、挂断 A 路、退款、释放并发 |
| 5 | 余额原子性 | 1000 并发创建请求，余额扣减无超扣无数据竞争 |
| 6 | 并发控制 | 超过 `concurrent_limit` 时返回 429，通话结束后计数正确释放（无双重释放） |
| 7 | 精确前缀路由 | 130-139→移动, 150-159/186→联通, 180-189→电信 正确命中 |
| 8 | 前缀一致性 | 系统启动时校验 gateway 前缀与 rate_plan 前缀一致，不一致输出告警 |
| 9 | 损耗统计 | A通B不通时正确记录 `wastage_type` 和 `wastage_cost` |
| 10 | 录音对齐 | 录音在 CHANNEL_BRIDGE 后开始，A/B 路录音时间戳差异 < 1s |
| 11 | 录音上传 | 合并完成后录音文件自动上传 S3/OSS，本地文件 24h 后清理 |
| 12 | Webhook 非阻塞 | 失败 webhook 通过延迟队列重投递，不阻塞其他 webhook 处理 |
| 13 | 退款一致性 | B路失败后 30 秒内完成退款，余额无误差 |
| 14 | 双模式权限 | 管理员全平台可见；客户严格隔离自有数据 |
| 15 | 网关容灾 | 主网关 DOWN 后，新建呼叫自动切换到备用网关（< 5s） |
| 16 | ESL 健康探测 | FSClient 每 10s 发送 status 命令，连续 3 次失败标记不健康 |
| 17 | 压测 | 1000 并发通话下 API P99 < 200ms，无内存泄漏 |
| 18 | 端口 | 所有 Web 服务通过 12345 端口可达，不依赖 80/443/8080 |

---

## 附录 A：v3.0 → v3.1 变更摘要

| # | 变更项 | 说明 |
|---|--------|------|
| 1 | Web 端口统一 | 所有 Web 服务默认端口从 8080 改为 **12345**，前端 WebSocket 地址从环境变量读取 |
| 2 | Park 超时保护 | Dialplan catch-all 增加 `park_timeout=60` + `park_timeout_transfer`，防止 A 路通道泄漏 |
| 3 | 并发槽位释放 | 去掉创建流程中的 `defer redis.Decr()`，统一由 `finalizeCall()` 释放，修复双重释放 bug |
| 4 | B 路 originate 失败处理 | 新增 `handleBLegOriginateFailure()` 处理 ESL 命令级失败（不产生事件的场景） |
| 5 | Webhook 非阻塞重试 | Webhook Worker 从 `time.Sleep` 阻塞模式改为 `webhook_deliveries` 延迟队列 + Cron 轮询 |
| 6 | webhook_deliveries 表 | 从 callback_calls 拆出 webhook 字段到独立表，减少主表锁竞争 |
| 7 | 录音时机调整 | 录音从 A 路 CHANNEL_ANSWER 时开始 → 改为 CHANNEL_BRIDGE 后开始，保证 A/B 路录音对齐 |
| 8 | 录音上传策略 | 合并完成后立即上传 S3/OSS，本地文件保留 24h 后清理（上传失败延长至 72h） |
| 9 | 状态机精简 | 去掉 `a_ringing` / `b_ringing` 状态（代码未实现对应事件监听） |
| 10 | cdr_records.created_at | 修复原 schema 缺失 `created_at` 列但索引引用该列的问题 |
| 11 | 前缀一致性校验 | 新增启动时 `CheckPrefixConsistency()`，校验 gateway 前缀与 rate_plan 前缀一致性 |
| 12 | ESL 健康探测 | FSClient 新增 `healthCheckLoop()`，每 10s 探测，Manager.Pick() 跳过不健康实例 |
| 13 | FS 环境变量 | ESL 密码从 `${ESL_PASSWORD}` 改为 FS 原生 `$${esl_password}` 语法 + entrypoint 注入 |
| 14 | 错误码规范 | 新增第 16 章，定义统一错误响应体结构和业务错误码枚举 |
| 15 | 非功能性需求 | 新增第 14 章，定义 SLA、数据保留策略、RTO/RPO、日志保留时长 |
| 16 | Phase 拆分 | Phase 2 拆为 2a（核心 Portal，2周）和 2b（分析与管理 Portal，2周），总工期 +1 周 |
