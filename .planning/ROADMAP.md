# Roadmap: BOS3000 API 回拨双呼系统

## Overview

BOS3000 从基础平台服务（认证、计费、路由、合规）开始构建，然后用 Mock FSClient 实现完整的呼叫状态机和损耗分类（不依赖真实 FreeSWITCH），接着接入真实 FreeSWITCH 并完成录音和 Webhook 管线，最后分别交付 Admin Dashboard 和 Client Portal 两个前端。Mock 优先策略确保最复杂的状态机逻辑在引入真实电话复杂度之前得到充分测试。

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: 平台基础** - 认证、计费引擎、路由引擎、合规风控、基础设施规范
- [ ] **Phase 2: 呼叫引擎（Mock）** - A/B 双外呼状态机 + 损耗分类，全程 Mock FSClient
- [ ] **Phase 3: FreeSWITCH 集成 + 录音 + Webhook** - 真实 ESL 接入、FS 高可用、录音管线、Webhook 投递
- [ ] **Phase 4: Admin Dashboard** - 运营大盘、客户/网关/话单/财务/合规管理、前端品质基础
- [ ] **Phase 5: Client Portal** - 客户自助仪表盘、回拨操作、话单、财务、API 集成、WebSocket 实时推送

## Phase Details

### Phase 1: 平台基础
**Goal**: 平台具备完整的用户认证、余额管理、费率计算、路由选择和合规检查能力，所有后续业务逻辑可在此基础上构建
**Depends on**: Nothing (first phase)
**Requirements**: AUTH-01, AUTH-02, AUTH-03, BILL-01, BILL-02, BILL-03, BILL-04, BILL-05, ROUT-01, ROUT-02, ROUT-03, ROUT-04, COMP-01, COMP-02, COMP-03, INFR-01, INFR-02, INFR-03, INFR-04
**Success Criteria** (what must be TRUE):
  1. 管理员可通过 JWT Cookie 登录并访问全平台数据；客户可通过 JWT Token 或 API Key（含 IP 白名单校验）认证，且只能访问自己的数据
  2. 呼叫发起前可原子预扣余额（余额不足返回 402），通话结束可精确结算并退还差额，交易流水完整记录
  3. 给定被叫号码，路由引擎可返回正确的 B 路网关（前缀匹配 + 容灾降级）和 A 路网关（加权轮询）
  4. 被叫号码命中黑名单时呼叫被拒绝；所有管理操作写入审计日志；日呼叫量超限返回 429
  5. docker-compose.dev.yml 一键启动开发环境；错误码规范统一实现；启动时校验前缀一致性
**Plans**: 5 plans

Plans:
- [ ] 01-01-PLAN.md — Shared foundation: error codes, types, docker-compose, port config
- [ ] 01-02-PLAN.md — Auth service: dual auth handler, login, API Key management
- [ ] 01-03-PLAN.md — Billing service: balance pre-deduction, rate plans, transactions
- [ ] 01-04-PLAN.md — Routing service: gateway selection, DID management, health checks
- [ ] 01-05-PLAN.md — Compliance service: blacklist, audit logging, rate limiting

### Phase 2: 呼叫引擎（Mock）
**Goal**: 用户可通过 API 发起回拨，系统通过 Mock FSClient 运行完整的 A/B 双外呼状态机，包括状态查询、强制挂断和损耗自动分类
**Depends on**: Phase 1
**Requirements**: CALL-01, CALL-02, CALL-03, CALL-04, CALL-05, CALL-06, CALL-07, WAST-01, WAST-02
**Success Criteria** (what must be TRUE):
  1. 用户调用回拨 API 后，系统依次执行 A 路外呼 -> park -> B 路外呼 -> 桥接 -> 结束的完整流程（Mock 模式），并生成准确的 CDR 记录
  2. A 路 park 超过 60s 自动挂断；B 路 originate 失败时 A 路被挂断、余额退款、并发槽位释放
  3. 用户可查询任意呼叫的实时状态（含 A/B 路详情、桥接时长、损耗类型）；管理员或客户可强制挂断进行中的通话
  4. 系统自动将失败呼叫分类为 a_connected_b_failed 或 bridge_broken_early，并精确计算损耗成本
  5. 所有状态机流程可通过 Mock FSClient 进行单元测试，不依赖真实 FreeSWITCH
**Plans**: 3 plans

Plans:
- [ ] 02-01-PLAN.md — FSClient interface, MockFSClient, DB schema (callback_calls + system_configs)
- [ ] 02-02-PLAN.md — State machine, finalizeCall cleanup, wastage classification
- [ ] 02-03-PLAN.md — Callback API endpoints (initiate, status, hangup, list) + integration tests

### Phase 3: FreeSWITCH 集成 + 录音 + Webhook
**Goal**: 系统通过真实 FreeSWITCH 完成 A/B 双外呼桥接，支持通话录音（分轨 + 合并 + S3 上传）和 Webhook 异步通知（含重试和 DLQ）
**Depends on**: Phase 2
**Requirements**: FS-01, FS-02, FS-03, FS-04, FS-05, REC-01, REC-02, REC-03, REC-04, HOOK-01, HOOK-02, HOOK-03, HOOK-04
**Success Criteria** (what must be TRUE):
  1. FSClient 通过 eslgo 连接真实 FreeSWITCH，仅暴露 5 个方法；ESL 事件处理器正确响应 CHANNEL_ANSWER/BRIDGE/HANGUP
  2. FreeSWITCH 双机热备运行，健康探测每 10s 执行，故障实例 < 5s 内被跳过，新通话自动路由到健康实例
  3. 通话桥接后自动开始 A/B 分轨录音，通话结束后异步 ffmpeg 合并为双声道 MP3 并上传 S3；用户可在线播放和下载录音
  4. 呼叫状态变更时自动创建 Webhook 投递记录，Worker 以指数退避重试，超限进 DLQ；管理员可查看 DLQ 并手动重试
  5. 客户可配置 webhook_url 并查看最近发送记录
**Plans**: 4 plans

Plans:
- [ ] 03-01-PLAN.md — ESLFSClient implementation (eslgo wrapper) + FSClientManager HA + Docker FreeSWITCH
- [ ] 03-02-PLAN.md — Recording service: Pub/Sub merge worker, ffmpeg, S3 upload, presigned URL API
- [ ] 03-03-PLAN.md — Webhook service: delivery worker, HMAC signing, DLQ admin APIs, client config APIs
- [ ] 03-04-PLAN.md — Integration wiring: state machine recording/webhook triggers + FSClientManager init

### Phase 4: Admin Dashboard
**Goal**: 管理员可通过专业级 Web 界面完成全部平台运营操作：实时监控、客户管理、网关管理、话单查询、损耗分析、财务对账、合规审计和系统运维
**Depends on**: Phase 3
**Requirements**: ADMN-01, ADMN-02, ADMN-03, ADMN-04, ADMN-05, ADMN-06, ADMN-07, ADMN-08, ADMN-09, ADMN-10, WAST-03, UI-01, UI-02, UI-03
**Success Criteria** (what must be TRUE):
  1. 管理员登录后看到运营大盘：实时并发数、今日收入/损耗、桥接成功率、告警卡片；数据图表专业美观（Recharts 统一风格）
  2. 管理员可完成客户全生命周期管理（开户、充值/扣款、冻结、API Key 查看）和网关池管理（健康状态实时同步、手动上下线、容灾配置、测试外呼）
  3. 管理员可跨用户查询全量话单、监控实时通话并强制挂断；可查看平台损耗趋势、客户损耗排名、B 路失败原因分布
  4. 管理员可进行财务对账（全平台流水、费率模板管理、毛利分析）、DID 号码管理（批量导入/分配）、黑名单管理和审计日志查询
  5. 界面达到商业级水准：shadcn/ui + MagicUI 动效、响应式布局、暗色/亮色主题切换、加载状态优雅
**Plans**: 8 plans

Plans:
- [ ] 04-01-PLAN.md — Project scaffold, Vite config, shadcn/ui, Tailwind v4 theme, test infra
- [ ] 04-02-PLAN.md — App shell (sidebar + topnav), router, auth flow, API layer, shared components
- [ ] 04-03-PLAN.md — Dashboard overview (stats, alerts, charts) + wastage analysis center
- [ ] 04-04-PLAN.md — Customer management + gateway management
- [ ] 04-05-PLAN.md — CDR management (query + live monitoring) + finance center
- [ ] 04-06-PLAN.md — DID management + compliance (blacklist, audit logs)
- [ ] 04-07-PLAN.md — Operations monitoring + system settings
- [ ] 04-08-PLAN.md — Visual polish pass + human verification checkpoint

### Phase 5: Client Portal
**Goal**: 客户可通过自助 Web 门户完成日常操作：查看仪表盘、发起回拨、查询话单和损耗、管理财务和 API 集成，并通过 WebSocket 接收实时通话状态推送
**Depends on**: Phase 4
**Requirements**: CLNT-01, CLNT-02, CLNT-03, CLNT-04, CLNT-05, CLNT-06, CLNT-07, WAST-04, UI-04
**Success Criteria** (what must be TRUE):
  1. 客户登录后看到今日概览仪表盘（呼叫数/成功率/损耗率/消费/余额/实时并发），数据实时更新
  2. 客户可通过 Web 表单发起回拨、批量导入（Excel）、查看进行中通话并自助挂断
  3. 客户可查询/导出话单（Excel/CSV），查看详情页（A/B 路分离、hangup_cause、录音播放）；可查看损耗概览趋势和明细
  4. 客户可查看余额与信用额度、消费流水、费率查询；可管理 API Key、配置/测试 Webhook、设置 IP 白名单
  5. WebSocket 实时推送通话状态变更到 Admin 和 Client Dashboard，无需页面轮询

**Plans**: TBD

Plans:
- [ ] 05-01: TBD
- [ ] 05-02: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. 平台基础 | 0/5 | Planning complete | - |
| 2. 呼叫引擎（Mock） | 0/3 | Planning complete | - |
| 3. FreeSWITCH + 录音 + Webhook | 0/4 | Planning complete | - |
| 4. Admin Dashboard | 0/8 | Planning complete | - |
| 5. Client Portal | 0/2 | Not started | - |
