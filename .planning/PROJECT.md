# BOS3000 — API 回拨双呼系统

## What This Is

BOS3000 是一个 API 驱动的回拨双呼（Click-to-Call）平台。管理员或客户通过 API/Web 发起呼叫，系统先外呼 A 路（主叫方），A 接通后再外呼 B 路（被叫方），桥接双方完成通话。面向 B2B 通信服务商，提供完整的计费、损耗分析、录音和客户自助管理能力。

## Core Value

**A/B 双外呼桥接必须可靠完成**——从 API 发起到 A 路外呼、A 接通后 B 路外呼、双路桥接、通话结束后精准计费和录音归档，整个链路不能丢通话、不能错扣费。

## Requirements

### Validated

<!-- 已交付并确认有价值的需求 -->

（尚无——交付后验证）

### Active

<!-- 当前范围，正在构建 -->

- [ ] API 发起回拨，A/B 双外呼 + 桥接
- [ ] FreeSWITCH ESL 控制（通过 eslgo 封装的 FSClient）
- [ ] A 路 park + 超时保护（60s）
- [ ] B 路 originate 命令级失败的显式处理
- [ ] 原子余额预扣（PG 行锁）+ 并发槽位控制（Redis INCR）
- [ ] B 路精确前缀路由 + A 路轮询路由
- [ ] 损耗分析（A通B不通、桥接秒断）
- [ ] 录音：桥接后开始、A/B 分轨、ffmpeg 合并、S3 上传
- [ ] Webhook 异步重试 + DLQ 死信队列
- [ ] 双模式权限（Admin 全平台 / Client 数据隔离）
- [ ] Admin Dashboard（运营大盘、客户管理、网关管理、话单、损耗分析、财务、合规、运维）
- [ ] Client Portal（仪表盘、发起回拨、话单、损耗分析、财务、API 集成）
- [ ] WebSocket 实时通话推送
- [ ] FreeSWITCH 高可用（双机热备 + 健康感知自动切换）

### Out of Scope

<!-- 明确排除，附原因 -->

- 呼入路由——纯 API 外呼系统，无接入号
- 移动端 App——Web 优先，移动端后续考虑
- OAuth 社交登录——v1 用邮箱密码 + API Key 足够
- 实时聊天——不在核心业务范围
- 视频通话——存储带宽成本高，非核心需求

## Context

- **架构**：Encore.go（控制面）+ FreeSWITCH（信令媒体面）+ React/Vite/shadcn（前端）
- **从 OpenSIPS B2BUA 切换到纯 FreeSWITCH**：回拨双呼是 FS `originate` 的经典用例，去掉 OpenSIPS + rtpengine 后信令媒体层简化为单一组件
- **eslgo 封装策略**：引用 eslgo 但浅度封装为 FSClient，对外仅暴露 5 个方法，隔离底层依赖
- **Mock 优先开发**：Phase 1a 全程使用 Mock FSClient，ESL 事件通过单测模拟
- **前端同仓**：Admin Dashboard + Client Portal 与 Encore 后端放在同一仓库
- **数据库**：9 张核心表（users, fs_gateways, did_numbers, callback_calls, webhook_deliveries, cdr_records, rate_plans, blacklisted_numbers, transactions）+ 审计日志 + 系统配置
- **默认端口 12345**：所有 Web 服务统一使用 12345，避开受限端口
- **MNP 携号转网**：Phase 1 按前缀路由接受少量误判，后续可接入 HLR 查询

## Constraints

- **Tech stack**: Encore.go + FreeSWITCH 1.10+ ESL + sqlc + React 18/Vite 5/shadcn/Tailwind 4 — PRD 已确定技术栈
- **ESL 库**: eslgo，浅度封装，替换成本 < 100 行
- **数据库**: PostgreSQL（Encore 托管）+ Redis（并发计数 + 缓存）
- **录音依赖**: ffmpeg（合并 A/B 路录音）
- **端口**: Web 服务统一 12345，SIP 5080，ESL 8021
- **B 路路由**: 精确 3 位前缀匹配（130→移动, 150→联通, 180→电信）

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| FreeSWITCH 替代 OpenSIPS B2BUA | 回拨双呼是 FS originate 经典用例，去掉 OpenSIPS+rtpengine 简化为单组件 | — Pending |
| eslgo 浅封装为 FSClient | 隔离底层依赖，对外仅暴露 5 个方法，替换成本低 | — Pending |
| Mock 优先开发 | Phase 1a 不依赖真实 FS，降低开发环境复杂度 | — Pending |
| 录音在 CHANNEL_BRIDGE 后开始 | 保证 A/B 路录音时间戳对齐 | — Pending |
| Webhook 拆独立表 webhook_deliveries | 避免重试时频繁 UPDATE 呼叫主表，减少锁竞争 | — Pending |
| 并发槽位由 finalizeCall 统一释放 | 不使用 defer，避免 ESL 事件异步场景下的双重释放 | — Pending |
| 前端与后端同仓 | 简化开发流程，共享类型定义 | — Pending |

---
*Last updated: 2026-03-09 after initialization*
