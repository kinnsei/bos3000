# 需求文档: BOS3000 API 回拨双呼系统

**定义日期:** 2026-03-09
**核心价值:** A/B 双外呼桥接必须可靠完成——从 API 发起到计费归档，整个链路不丢通话、不错扣费

## v1 需求

### 呼叫引擎

- [ ] **CALL-01**: 用户可通过 API 发起 A/B 双外呼回拨，系统先呼 A 路，A 接通后呼 B 路并桥接
- [ ] **CALL-02**: A 路外呼后进入 park 状态等待桥接，park 超时 60s 自动挂断防止通道泄漏
- [ ] **CALL-03**: B 路 originate 命令级失败时，系统显式标记损耗、挂断 A 路、退款、释放并发
- [ ] **CALL-04**: 呼叫状态机完整覆盖 a_dialing→a_connected→b_dialing→b_connected→bridged→finished/failed 全流程
- [ ] **CALL-05**: 用户可查询呼叫实时状态（含 A/B 路详情、桥接时长、损耗类型）
- [ ] **CALL-06**: 用户可强制挂断进行中的通话（管理员全局 / 客户自有）
- [ ] **CALL-07**: 系统通过 Mock FSClient 支持全状态机单测（不依赖真实 FreeSWITCH）

### FreeSWITCH 集成

- [ ] **FS-01**: FSClient 封装 eslgo，仅暴露 OriginateALeg / OriginateBLegAndBridge / HangupCall / StartRecording / StopRecording 5 个方法
- [ ] **FS-02**: ESL 事件处理器正确响应 CHANNEL_ANSWER / CHANNEL_BRIDGE / CHANNEL_HANGUP 事件
- [ ] **FS-03**: FSClient 健康探测每 10s 发送 status 命令，连续 3 次失败标记不健康
- [ ] **FS-04**: FreeSWITCH 双机热备，FSClient Manager 维护双连接，Pick() 自动跳过不健康实例
- [ ] **FS-05**: 新通话在任一 FS 实例故障时 < 5s 恢复到健康实例

### 计费与余额

- [ ] **BILL-01**: 呼叫发起前原子预扣余额（PG 行锁），余额不足返回 402
- [ ] **BILL-02**: 并发槽位通过 Redis INCR 控制，超限返回 429，由 finalizeCall 统一释放
- [ ] **BILL-03**: 通话结束后精确计算实际费用，退还预扣差额，写入 transactions 流水
- [ ] **BILL-04**: 费率优先级：用户级 a_leg_rate > 费率模板 rate_plans > B 路前缀费率
- [ ] **BILL-05**: 管理员可创建/编辑费率模板，为客户指定费率模板或用户级费率

### 路由引擎

- [ ] **ROUT-01**: A 路网关通过加权轮询选择，跳过不健康网关
- [ ] **ROUT-02**: B 路网关通过被叫前 3 位精确前缀匹配，降级到同运营商其他网关
- [ ] **ROUT-03**: 网关配置容灾关系（failover_gateway_id），主网关 DOWN 自动切备用
- [ ] **ROUT-04**: DID 号码池管理：导入、分配专属/公共池、自动选取外显号码

### 损耗分析

- [ ] **WAST-01**: 系统自动分类损耗类型：A通B不通（a_connected_b_failed）、桥接秒断（bridge_broken_early）
- [ ] **WAST-02**: 损耗成本精确计算并记录到呼叫记录（wastage_cost 字段）
- [ ] **WAST-03**: 管理员可查看平台级损耗趋势（日/周/月）、客户损耗排名、B 路失败原因分布
- [ ] **WAST-04**: 客户可查看自己的损耗率、损耗明细、B 路失败原因分布

### 录音

- [ ] **REC-01**: 录音在 CHANNEL_BRIDGE 后开始，保证 A/B 路录音时间戳对齐
- [ ] **REC-02**: A/B 路分轨录音（WAV），通话结束后异步 ffmpeg 合并为双声道 MP3
- [ ] **REC-03**: 录音文件合并后立即上传 S3/OSS，本地文件 24h 后清理
- [ ] **REC-04**: 用户可在线播放和下载 A 路、B 路、合并录音

### Webhook

- [ ] **HOOK-01**: 呼叫状态变更时创建 webhook_deliveries 记录（独立表，不锁呼叫主表）
- [ ] **HOOK-02**: Webhook Worker 非阻塞延迟队列重试，指数退避，超过最大次数进 DLQ
- [ ] **HOOK-03**: 管理员可查看 Webhook DLQ、手动重试失败记录
- [ ] **HOOK-04**: 客户可配置默认 webhook_url、查看最近发送记录

### 认证与权限

- [ ] **AUTH-01**: 管理员通过 JWT Cookie（HttpOnly, Secure）认证，数据范围全平台
- [ ] **AUTH-02**: 客户通过 JWT Token 或 API Key（IP 白名单校验）认证，数据严格隔离 WHERE user_id = current_user_id
- [ ] **AUTH-03**: 客户可管理 API Key（查看 prefix、重置密钥）和 IP 白名单

### 合规风控

- [ ] **COMP-01**: 全局 + 每客户黑名单，呼叫前检查被叫号码
- [ ] **COMP-02**: 所有管理操作写入审计日志（操作人、操作类型、资源、变更前后值、IP）
- [ ] **COMP-03**: 日呼叫量限制（daily_limit），超限返回 429

### Admin Dashboard

- [ ] **ADMN-01**: Overview 运营大盘：实时并发、今日收入/损耗、桥接成功率、告警卡片
- [ ] **ADMN-02**: 客户管理：列表、详情、开户、充值/扣款、冻结、API Key 查看
- [ ] **ADMN-03**: 网关管理：A/B 路网关池、健康状态实时同步、手动上下线、容灾配置、测试外呼
- [ ] **ADMN-04**: 话单管理：全量话单查询（跨用户筛选）、实时通话监控（强制挂断）
- [ ] **ADMN-05**: 损耗分析中心：平台损耗趋势图、客户损耗排名、B 路失败原因分布、A 路等待时长分布
- [ ] **ADMN-06**: 财务中心：对账、全平台流水、费率模板管理、毛利分析（分客户/分网关盈亏）
- [ ] **ADMN-07**: DID 管理：号码列表、批量导入、分配管理
- [ ] **ADMN-08**: 合规：全局黑名单管理、审计日志查询
- [ ] **ADMN-09**: 运维工具：FreeSWITCH 状态、ESL 连接健康度、系统健康监控
- [ ] **ADMN-10**: 系统设置：system_configs 可视化编辑

### Client Portal

- [ ] **CLNT-01**: Dashboard 仪表盘：今日概览（呼叫数/成功率/损耗率/消费/余额）、实时并发
- [ ] **CLNT-02**: 回拨操作：Web 表单发起回拨、批量导入（Excel）、进行中通话列表（自助挂断）
- [ ] **CLNT-03**: 话单：查询/导出（Excel/CSV）、详情页（A/B 路分离、hangup_cause、录音播放）
- [ ] **CLNT-04**: 财务中心：余额与信用额度、消费流水、费率查询、发票申请
- [ ] **CLNT-05**: 损耗分析：损耗概览趋势、损耗明细、B 路失败原因分布
- [ ] **CLNT-06**: API 集成：API Key 管理、Webhook 配置/测试/日志、IP 白名单、API 文档
- [ ] **CLNT-07**: 账户设置：基本资料、外显号码池查看、安全设置、通知设置

### 前端品质（全局约束）

- [ ] **UI-01**: 前端设计达到专业商业级水准：shadcn/ui 为基础组件 + MagicUI 动效点缀，整体视觉高级精致
- [ ] **UI-02**: 数据可视化专业：Recharts 图表风格统一、配色协调、交互丝滑（损耗趋势、财务报表、运营大盘）
- [ ] **UI-03**: 响应式布局，暗色/亮色主题切换，加载状态优雅（Skeleton + 动画过渡）
- [ ] **UI-04**: WebSocket 实时通话推送到 Admin/Client Dashboard，无需轮询

### 基础设施

- [ ] **INFR-01**: 所有 Web 服务统一端口 12345，不依赖 80/443/8080
- [ ] **INFR-02**: docker-compose.dev.yml 一键启动 FreeSWITCH 开发环境（含 entrypoint 环境变量注入）
- [ ] **INFR-03**: 启动时校验 fs_gateways 前缀与 rate_plans 前缀一致性，不一致输出告警
- [ ] **INFR-04**: 错误码规范统一实现（HTTP 状态码 + 业务错误码 + 结构化错误响应体）

## v2 需求

### 通知

- **NOTF-01**: 用户收到 app 内通知（余额不足告警、网关 DOWN 告警）
- **NOTF-02**: 邮件通知（余额低于阈值、每日摘要报告）

### 高级路由

- **ROUT-05**: MNP 携号转网查询接入（HLR 查询 + Redis 缓存）
- **ROUT-06**: ASR/ACD 质量指标实时计算，网关质量劣化时自动告警

### 高级管理

- **ADMN-11**: 多层级客户体系（代理商→子客户）
- **ADMN-12**: 费率模板版本管理（生效日期、历史版本）

### 性能与可靠性

- **PERF-01**: 1000+ 并发通话压测报告与容量规划
- **PERF-02**: 录音文件生命周期自动管理（热→冷存储迁移）

## Out of Scope

| 功能 | 原因 |
|------|------|
| 呼入路由 / IVR | 纯 API 外呼系统，呼入是完全不同的架构 |
| 移动端 App | Web 优先，响应式可覆盖移动端，B2B 场景 API 优先 |
| OAuth 社交登录 | B2B 平台用 API Key + 邮箱密码足够，社交登录增加复杂度无收益 |
| 实时通话监听/耳语 | 需要媒体分叉 + WebRTC，复杂度极高，v1 提供录音回放 |
| AI 语音转写 | 基础设施成本高，非核心能力，提供录音下载 API 让客户自行转写 |
| 视频通话 | 10x 带宽/存储成本，完全不同的产品品类 |
| 亚秒级计费 | 行业标准为秒级或 6 秒块，亚秒引发舍入争议 |

## Traceability

| 需求 | Phase | 状态 |
|------|-------|------|
| CALL-01 | — | Pending |
| CALL-02 | — | Pending |
| CALL-03 | — | Pending |
| CALL-04 | — | Pending |
| CALL-05 | — | Pending |
| CALL-06 | — | Pending |
| CALL-07 | — | Pending |
| FS-01 | — | Pending |
| FS-02 | — | Pending |
| FS-03 | — | Pending |
| FS-04 | — | Pending |
| FS-05 | — | Pending |
| BILL-01 | — | Pending |
| BILL-02 | — | Pending |
| BILL-03 | — | Pending |
| BILL-04 | — | Pending |
| BILL-05 | — | Pending |
| ROUT-01 | — | Pending |
| ROUT-02 | — | Pending |
| ROUT-03 | — | Pending |
| ROUT-04 | — | Pending |
| WAST-01 | — | Pending |
| WAST-02 | — | Pending |
| WAST-03 | — | Pending |
| WAST-04 | — | Pending |
| REC-01 | — | Pending |
| REC-02 | — | Pending |
| REC-03 | — | Pending |
| REC-04 | — | Pending |
| HOOK-01 | — | Pending |
| HOOK-02 | — | Pending |
| HOOK-03 | — | Pending |
| HOOK-04 | — | Pending |
| AUTH-01 | — | Pending |
| AUTH-02 | — | Pending |
| AUTH-03 | — | Pending |
| COMP-01 | — | Pending |
| COMP-02 | — | Pending |
| COMP-03 | — | Pending |
| ADMN-01 | — | Pending |
| ADMN-02 | — | Pending |
| ADMN-03 | — | Pending |
| ADMN-04 | — | Pending |
| ADMN-05 | — | Pending |
| ADMN-06 | — | Pending |
| ADMN-07 | — | Pending |
| ADMN-08 | — | Pending |
| ADMN-09 | — | Pending |
| ADMN-10 | — | Pending |
| CLNT-01 | — | Pending |
| CLNT-02 | — | Pending |
| CLNT-03 | — | Pending |
| CLNT-04 | — | Pending |
| CLNT-05 | — | Pending |
| CLNT-06 | — | Pending |
| CLNT-07 | — | Pending |
| UI-01 | — | Pending |
| UI-02 | — | Pending |
| UI-03 | — | Pending |
| UI-04 | — | Pending |
| INFR-01 | — | Pending |
| INFR-02 | — | Pending |
| INFR-03 | — | Pending |
| INFR-04 | — | Pending |

**Coverage:**
- v1 需求: 55 total
- 已映射到 phase: 0
- 未映射: 55 ⚠️

---
*需求定义: 2026-03-09*
*最后更新: 2026-03-09 初始定义*
