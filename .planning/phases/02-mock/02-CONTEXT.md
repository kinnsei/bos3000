# Phase 2: 呼叫引擎（Mock）- Context

**Gathered:** 2026-03-10
**Status:** Ready for planning

<domain>
## 阶段范围

用户可通过 API 发起回拨，系统通过 Mock FSClient 运行完整的 A/B 双外呼状态机（a_dialing→a_connected→b_dialing→b_connected→bridged→finished/failed），支持状态查询、强制挂断和损耗自动分类。不依赖真实 FreeSWITCH。

覆盖需求：CALL-01, CALL-02, CALL-03, CALL-04, CALL-05, CALL-06, CALL-07, WAST-01, WAST-02

</domain>

<decisions>
## 实现决策

### 状态机流程与异常处理
- A 路外呼失败（无人接听/拒接/号码错误）直接结束呼叫，不重试，标记 failed，退还预扣、释放并发，不计损耗（A 未接通无实际成本）
- B 路 originate 失败不尝试备用网关，直接结束。A 路已接通在等待，重试会延长等待时间增加损耗。容灾切换由 Phase 1 路由引擎在选择网关时处理
- "桥接秒断"阈值可配置，全局默认 10 秒，管理员可在 system_configs 中调整
- 强制挂断（管理员或客户）视为正常结束，按实际通话时长计费
- A 路 park 超过 60s 自动挂断（PRD 已定义）

### 回拨 API 设计
- 发起回拨参数：a_number + b_number 必填，可选参数包括 caller_id（指定外显号）、max_duration（最大时长，秒）、custom_data（透传 JSON）、preferred_gateway（优先网关）、callback_url（回调 URL）
- custom_data 接受任意 JSON 对象，系统透明存储，在 CDR 和 Webhook 中原样返回，限制 1KB 大小
- max_duration 可选，不传则使用全局默认值（如 3600 秒），超时自动挂断
- 查询状态返回完整详情：当前状态、A/B 路分别的状态、外显号、桥接时长、损耗类型、费用信息，一次调用获取所有信息
- 仅提供单个回拨 API，不提供批量接口。批量场景由前端循环调用，后端通过并发槽位控制
- 强制挂断为统一接口，后端根据认证角色自动判断权限：管理员可挂任何通话，客户只能挂自己的
- 发起回拨响应只返回 call_id 和初始状态，不返回预估费用。预扣是内部逻辑，实际费用通过 CDR 查看

### CDR 话单记录
- A/B 路信息作为 callback_calls 表的字段嵌入（a_number, a_answer_time, a_hangup_cause, b_number, b_answer_time, b_hangup_cause 等），一条记录 = 一次回拨
- callback_calls 表同时承载实时状态和历史话单职责，通话结束后状态变为 finished/failed，自然成为 CDR，不需要独立的 cdr_records 表
- CDR 记录在回拨发起时创建（状态 initiating），后续状态变更通过 UPDATE 更新同一条记录。保证进行中的呼叫可追踪，不会因崩溃丢失
- 损耗信息嵌入 callback_calls 表：wastage_type（损耗类型）、wastage_cost（损耗成本）、wastage_duration（损耗时长），无损耗时为 NULL
- 所有时间戳存储到毫秒级（timestamptz），时长计算精确到毫秒，计费时按规则向上取整
- CDR 查询 API 支持完整筛选：时间范围、状态、A/B 号码、损耗类型、客户 ID 等，分页返回

### Mock FSClient 模拟策略
- Mock 采用即时模拟方式，立即触发事件回调（ANSWER/BRIDGE/HANGUP），不实际等待。测试运行快
- 核心覆盖 5 种场景：① A 路拒接/无人接听 ② A 路接通但 B 路失败 ③ 桥接成功但秒断 ④ 正常完成 ⑤ A 路 park 超时
- Mock 通过注入配置方式控制行为（如 MockConfig{ALegResult: "answer", BLegResult: "reject"}），每个测试用例可独立配置不同场景
- Mock 支持交互式 API 测试：encore run 时 Mock 也能工作，开发者可用 curl/Postman 调 API 测试完整流程

### Claude 自行决定
- FSClient 接口的具体方法签名
- 状态机的内部实现方式（goroutine + channel 还是其他模式）
- Mock 的具体代码组织（文件结构、包划分）
- CDR 表的完整字段列表和索引设计
- 并发控制的具体实现细节

</decisions>

<specifics>
## 具体想法

- PRD 已定义 FSClient 对外仅暴露 5 个方法：OriginateALeg / OriginateBLegAndBridge / HangupCall / StartRecording / StopRecording（Phase 2 中 StartRecording/StopRecording 为空实现，Phase 3 补充）
- 状态序列固定：a_dialing → a_connected → b_dialing → b_connected → bridged → finished/failed
- 并发槽位由 finalizeCall 统一释放，不使用 defer（避免 ESL 事件异步场景下双重释放）——这是 PROJECT.md 中的关键决策

</specifics>

<code_context>
## 现有代码情况

### 可复用资产
- 无现有代码，项目仍为空（仅 encore.app + go.mod）
- Phase 1 计划已完成但尚未执行

### 已建立的模式
- Encore.go 框架规范：服务定义、API 注解、sqldb、errs 包、auth handler
- Go 1.25.6 + encore.dev v1.52.1

### 集成点
- 依赖 Phase 1 的认证服务（auth handler，双模式：Admin JWT Cookie / Client JWT Token + API Key）
- 依赖 Phase 1 的计费服务（余额预扣、精确结算、交易流水）
- 依赖 Phase 1 的路由服务（A 路加权轮询、B 路前缀匹配）
- 依赖 Phase 1 的合规服务（黑名单检查、日呼叫量限制）

</code_context>

<deferred>
## 延后的想法

无——讨论保持在阶段范围内

</deferred>

---

*Phase: 02-mock*
*Context gathered: 2026-03-10*
