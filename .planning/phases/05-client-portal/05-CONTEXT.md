# Phase 5: Client Portal - 上下文

**收集日期:** 2026-03-10
**状态:** 待规划

<domain>
## 阶段范围

客户可通过自助 Web 门户完成日常操作：查看仪表盘概览（呼叫数/成功率/损耗率/消费/余额/实时并发）、通过 Web 表单发起回拨、查询/导出话单（含 A/B 路详情和录音播放）、查看损耗分析、管理财务（余额/流水/费率）、配置 API 集成（API Key/Webhook/IP 白名单），并通过 WebSocket 接收实时通话状态推送。WebSocket 同时升级 Admin Dashboard（替换 30s 轮询）。

覆盖需求：CLNT-01, CLNT-02（移除批量导入）, CLNT-03, CLNT-04, CLNT-05, CLNT-06, CLNT-07, WAST-04, UI-04

</domain>

<decisions>
## 实现决策

### Portal 与 Admin 代码复用
- 复制 Admin 脚手架（Vite/shadcn/Tailwind 配置）作为 Portal 起点，两个独立项目，各自维护组件
- Portal 放在仓库根目录 /portal 子目录（与 /admin 同级）
- 主题配置完全一致：Vercel Dashboard 风格、绿/青色系、暗色/亮色主题
- 登录页复用 Admin 结构，替换标题为"客户门户"
- API 客户端：Admin 和 Portal 共用同一份 encore gen client 生成的 TS 客户端，各自项目内部生成到 src/lib/client
- 客户认证：邮箱+密码登录，后端返回 JWT Token，前端存 localStorage（与 Phase 1 客户 JWT Token 认证一致）

### WebSocket 实时推送
- 仅推送通话状态变更事件（initiating → a_dialing → a_connected → b_dialing → bridged → finished/failed）
- WebSocket 端点放在 callback 服务内（Encore raw endpoint），状态机变更时直接推送，无跨服务通信
- 按用户隔离推送：WebSocket 连接时验证 JWT Token，服务端按 user_id 维护连接池，客户只收到自己的通话状态。Admin 接收所有通话
- Admin Dashboard 同步升级为 WebSocket（替换 Phase 4 的 30s 轮询），一次实现两处受益
- 断线处理：前端自动指数退避重连，顶部显示"实时连接已断开"提示条，重连成功后自动消失
- 其他数据（概览指标、财务、话单列表等）继续用 TanStack Query 定时重新获取

### 回拨操作交互
- **移除批量导入（Excel）功能**——B2B 客户通过 API 批量调用更合理，Web 表单面向单次操作
- 回拨操作独占一页：上半部分为发起表单（A 号码 + B 号码必填，可展开高级选项），下半部分显示最近发起的回拨历史
- 进行中通话使用卡片列表展示：每张卡片显示 A/B 号码、当前状态（彩色标签）、时长、挂断按钮。WebSocket 实时更新卡片状态
- 发起回拨后，新通话卡片立即出现在同页面的进行中列表，WebSocket 实时更新状态，无需跳转

### Portal 导航与仪表盘布局
- 与 Admin 一致的左侧可收缩侧边栏 + 顶部导航栏布局
- 侧边栏菜单按客户功能分组：概览、回拨、话单、损耗分析、财务中心、API 集成、账户设置
- 仪表盘概览页：顶部 4-6 个指标卡片一行排列（今日呼叫、成功率、损耗率、消费、余额、实时并发），下方放 1-2 个趋势图表
- 话单详情使用侧抽屉 Sheet：点击话单列表某条后右侧滑出，显示 A/B 路信息分区、状态时间线、录音播放器、费用明细
- 桌面优先设计，平板可用，手机不做专门优化（与 Admin 一致）
- 界面语言中文为主，v1 不做 i18n（与 Admin 一致）

### Claude 自行决定
- shadcn/ui 组件的具体选择和组合
- 回拨表单的高级选项展开/收起交互细节
- 进行中通话卡片的具体设计和动画效果
- 仪表盘图表的具体配置（Recharts）
- 侧抽屉中录音播放器的具体实现
- WebSocket 重连的具体退避参数
- 导出话单的具体格式处理（Excel/CSV）
- 账户设置页的具体布局
- 前端目录结构和文件组织方式

</decisions>

<specifics>
## 具体想法

- 视觉与 Admin Dashboard 完全一致的品牌体验——客户感受是同一产品的不同入口
- 批量回拨场景应通过 API 完成，Web Portal 只做单次操作和查看监控
- 回拨页面是"操作中心"：发起 + 实时监控在同一页，WebSocket 让体验流畅
- Phase 2 已决策：仅提供单个回拨 API，批量由前端循环调用（现在前端也不做批量）

</specifics>

<code_context>
## 现有代码情况

### 可复用资产
- 无现有前端代码，项目仍为空（仅 encore.app + go.mod）
- Phase 4 Admin Dashboard 计划已完成但尚未执行，Portal 可复制其脚手架配置
- Phase 1-3 后端 API 设计已在各阶段 PLAN 中定义

### 已建立的模式
- 后端：Encore.go 框架规范（服务定义、API 注解、sqldb、errs 包、auth handler、raw endpoint）
- Encore 内置 TypeScript 客户端生成（`encore gen client --lang=typescript`）
- Phase 4 前端技术栈：React 18 + Vite 5 + shadcn/ui + Tailwind CSS 4 + TypeScript + TanStack Query + React Router v7
- Phase 4 视觉规范：Vercel Dashboard 风格、绿/青色系、MagicUI 动效克制点缀、Recharts 图表
- Phase 4 交互模式：Dialog 弹窗（简单操作）、Sheet 侧抽屉（复杂操作/详情）、DataTable 表格

### 集成点
- Encore gen client 生成的 TS 客户端连接所有后端 API
- JWT Token 认证对接 Phase 1 的 auth handler（客户模式）
- WebSocket 连接 callback 服务的 raw endpoint
- 后端所有 API 响应遵循 Encore 默认规范（成功直接返回数据，失败返回 {code, message, details}）
- Portal 需要对接的核心后端服务：auth、billing、routing、callback、recording、webhook
- Admin Dashboard 也需接入 WebSocket（替换轮询）

</code_context>

<deferred>
## 延后的想法

无——讨论保持在阶段范围内

</deferred>

---

*Phase: 05-client-portal*
*上下文收集日期: 2026-03-10*
