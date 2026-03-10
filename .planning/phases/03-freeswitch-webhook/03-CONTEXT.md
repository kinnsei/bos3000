# Phase 3: FreeSWITCH 集成 + 录音 + Webhook - Context

**Gathered:** 2026-03-10
**Status:** Ready for planning

<domain>
## 阶段范围

系统通过真实 FreeSWITCH 完成 A/B 双外呼桥接，支持通话录音（分轨 + 合并 + S3 上传）和 Webhook 异步通知（含重试和 DLQ）。Phase 2 的 Mock FSClient 切换为真实 ESL 实现，Mock 保留用于测试。

覆盖需求：FS-01, FS-02, FS-03, FS-04, FS-05, REC-01, REC-02, REC-03, REC-04, HOOK-01, HOOK-02, HOOK-03, HOOK-04

</domain>

<decisions>
## 实现决策

### ESL 连接与事件处理
- 采用 Inbound 模式连接 FreeSWITCH ESL 端口（8021），应用主动连接 FS
- 断线后自动重连，指数退避（1s、2s、4s... 最大 30s），重连期间标记实例不健康，Pick() 自动跳过
- ESL 事件采用事件派发器模式驱动状态机：事件处理器解析 ESL 事件后通过回调/channel 通知状态机，状态机不直接依赖 ESL，保持与 Mock 相同的接口
- 只订阅业务相关事件：CHANNEL_ANSWER、CHANNEL_BRIDGE、CHANNEL_HANGUP，减少噪音
- 通话 UUID 由应用端预生成，originate 时传给 FS，ESL 事件通过 UUID 直接关联到 callback_calls 记录
- FSClient 接口不变，Phase 3 提供基于 eslgo 的真实实现替换 Phase 2 的 Mock，通过配置切换。Mock 继续用于单元测试
- 孤儿事件（找不到对应呼叫记录的 ESL 事件）记录警告日志（含 UUID 和事件详情），不崩溃不阻塞

### 录音管线
- 录音在 CHANNEL_BRIDGE 后开始（已决策，PROJECT.md）
- A/B 路分轨录音（WAV），通话结束后异步 ffmpeg 合并为双声道 MP3
- ffmpeg 合并任务通过 Encore Pub/Sub 消息队列异步触发：通话结束时发布录音合并消息，独立 Worker 消费处理。可重试、可观测、不影响主流程
- S3 存储路径：`recordings/{customer_id}/{YYYY-MM-DD}/{call_id}_merged.mp3`（分轨文件类似命名 `_a.wav`、`_b.wav`）
- 用户通过 S3 预签名 URL 播放和下载录音（有效期 15 分钟），减轻后端带宽压力
- 录音文件保留 90 天，通过 S3 生命周期策略自动删除（与审计日志保留期一致，Phase 1 已决策）
- 本地录音文件 24h 后清理（REC-03 已定义）

### Webhook 投递机制
- 呼叫状态变更时创建 webhook_deliveries 记录（独立表，已决策，PROJECT.md）
- Webhook Worker 指数退避重试 5 次，间隔 30s/1m/5m/15m/1h，超过最大次数进 DLQ
- Webhook 请求使用 HMAC-SHA256 签名验证：每个客户生成 webhook_secret，用 HMAC-SHA256 对 payload 签名，签名放在 X-Signature header 中
- Webhook payload 包含完整呼叫详情：event_type + call_id + 当前状态 + A/B 路详情 + custom_data + timestamp，客户无需回查 API
- 全部状态变更触发 Webhook 推送：initiating、a_dialing、a_connected、b_dialing、bridged、finished、failed
- 管理员可查看 DLQ 并手动重试（HOOK-03）
- 客户可配置默认 webhook_url 并查看最近发送记录（HOOK-04）

### FreeSWITCH 高可用
- 采用主备模式：新通话优先走主 FS，主故障时自动切换到备 FS
- 健康探测每 10s 发送 status 命令，连续 3 次失败标记不健康（FS-03 已定义）
- 主 FS 恢复健康后不自动回切，避免频繁切换引起不稳定。等待下次备机故障时自然切换
- 主 FS 故障时，所有在途通话标记为 failed，触发 finalizeCall（退款 + 释放并发）。新通话路由到备 FS
- 开发环境 docker-compose 默认启动单台 FS，双机可选启动（用于测试高可用场景）

### Claude 自行决定
- eslgo 连接的具体配置参数（超时、buffer 大小等）
- 事件派发器的具体实现方式（channel vs callback）
- 录音文件的临时存储路径和清理机制
- ffmpeg 合并的具体命令参数
- Webhook Worker 的具体实现（Encore Pub/Sub subscription 配置）
- FSClient Manager 的内部实现细节
- docker-compose 中 FreeSWITCH 的具体 SIP 配置

</decisions>

<specifics>
## 具体想法

- Phase 2 已定义 FSClient 接口 5 个方法：OriginateALeg / OriginateBLegAndBridge / HangupCall / StartRecording / StopRecording（Phase 2 中 StartRecording/StopRecording 为空实现）
- 并发槽位由 finalizeCall 统一释放，不使用 defer（PROJECT.md 关键决策）
- Webhook 签名参考 Stripe/GitHub 的标准做法
- 状态机流程固定：a_dialing → a_connected → b_dialing → b_connected → bridged → finished/failed

</specifics>

<code_context>
## 现有代码情况

### 可复用资产
- 无现有代码，项目仍为空（仅 encore.app + go.mod）
- Phase 1、2 计划已完成但尚未执行

### 已建立的模式
- Encore.go 框架规范：服务定义、API 注解、sqldb、errs 包、auth handler、Pub/Sub
- Go 1.25.6 + encore.dev v1.52.1
- eslgo 库（ESL 连接）

### 集成点
- 依赖 Phase 2 的 FSClient 接口定义（替换 Mock 为真实实现）
- 依赖 Phase 2 的状态机和 callback_calls 表
- 依赖 Phase 1 的用户表（webhook_url、webhook_secret 字段）
- Encore Object Storage（S3 上传/预签名 URL）
- Encore Pub/Sub（录音合并任务队列、Webhook 投递队列）

</code_context>

<deferred>
## 延后的想法

无——讨论保持在阶段范围内

</deferred>

---

*Phase: 03-freeswitch-webhook*
*Context gathered: 2026-03-10*
