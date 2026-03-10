# Phase 4: Admin Dashboard - 上下文

**收集日期:** 2026-03-10
**状态:** 待规划

<domain>
## 阶段范围

管理员可通过专业级 Web 界面完成全部平台运营操作：实时监控（运营大盘）、客户全生命周期管理、网关池管理、话单查询与实时通话监控、损耗分析、财务对账与毛利分析、DID 号码管理、合规审计、FreeSWITCH 运维监控和系统设置。同时建立前端工程基础设施，供 Phase 5 Client Portal 复用。

覆盖需求：ADMN-01~10, WAST-03, UI-01, UI-02, UI-03

</domain>

<decisions>
## 实现决策

### 整体布局与导航
- 采用左侧可收缩侧边栏 + 顶部导航栏的经典后台布局
- 侧边栏菜单按功能分组：监控（概览、话单、损耗）、管理（客户、网关、DID）、财务（财务中心）、系统（合规、运维、设置）
- 顶部导航栏放置：Logo、主题切换、通知图标、管理员信息
- 桌面优先设计，主要保证 1024px+ 体验，平板可用，手机端不做专门优化

### 视觉设计语言
- 参考 Vercel Dashboard 风格：现代感、黑白为主、强调数据和状态
- 主题：跟随操作系统暗色/亮色设置，支持手动切换
- 主色调：绿/青色系，用于按钮、选中态、链接等强调元素
- MagicUI 动效克制点缀：仅在关键位置使用（运营大盘数字滚动、告警卡片闪烁、页面切换过渡），其余管理页面保持干净高效
- 加载状态使用 Skeleton 骨架屏 + 动画过渡

### 界面语言
- 中文为主（菜单、标题、按钮、提示文字全部中文），v1 不做 i18n
- API 错误信息为英文（Phase 1 已决策），前端转换为中文提示

### 数据大盘（运营概览）
- 实时并发数、今日收入/损耗、桥接成功率作为核心指标卡片展示
- 四类告警卡片：桥接成功率低于阈值、客户余额不足、网关 DOWN、损耗率异常
- 数据刷新方式：每 30s 自动轮询刷新概览数据（Phase 5 WebSocket 可后续升级）

### 图表与数据可视化
- 统一使用 Recharts，图表风格与 Vercel Dashboard 保持一致
- 图表配色采用多色彩方案（绿/青/紫/橙/红），不同数据系列易区分
- 损耗分析中心按 PRD 实现三张图：① 损耗率趋势图（日/周/月）② 客户损耗 TOP 10 排名 ③ B 路失败原因饼图/柱状图
- 财务毛利分析支持分客户和分网关两个维度（每个客户/网关的收入/成本/毛利）

### 管理操作交互模式
- 简单操作（充值、冻结、上下线）使用 Dialog 弹窗
- 复杂操作（开户、网关配置）使用侧抽屉 Sheet
- 危险操作（冻结客户、删除黑名单、强制挂断）使用红色警告二次确认弹窗
- 表格列表统一使用 shadcn DataTable（基于 TanStack Table），支持排序、筛选、分页、列可见性控制
- DID 号码和黑名单批量导入使用 CSV/Excel 文件上传（预览数据 + 验证后确认导入）

### 前端工程基础设施
- 前端项目放在仓库根目录 /admin 子目录，后续 Client Portal 放同级 /portal
- 技术栈：React 18 + Vite 5 + shadcn/ui + Tailwind CSS 4 + TypeScript
- 状态管理：TanStack Query 管理服务端数据（缓存、重试、分页），局部 UI 状态用 React useState/useReducer
- 路由：React Router v7（纯 SPA 模式）
- API 客户端：使用 `encore gen client --lang=typescript` 生成类型安全的 TS 客户端，自动与后端 API 同步
- 认证流程：JWT Cookie（HttpOnly, Secure）登录页，Phase 1 已定义 Admin 认证方式

### Claude 自行决定
- shadcn/ui 组件的具体选择和组合
- MagicUI 动效的具体组件选用
- Recharts 图表的具体配置和交互细节
- 前端目录结构和文件组织方式
- 登录页的具体设计
- Skeleton 骨架屏的具体样式
- 表格列的默认配置
- 告警阈值的默认值

</decisions>

<specifics>
## 具体想法

- 视觉参考 Vercel Dashboard：现代感、黑白为主、数据驱动、状态清晰
- 动效要克制——运营大盘可以炫一点，管理页面要高效干净
- 管理员后台主要在桌面使用，不需要在手机上操作

</specifics>

<code_context>
## 现有代码情况

### 可复用资产
- 无现有前端代码，项目为全新 React 应用
- Phase 1-3 后端计划已完成但尚未执行，API 接口设计已在各阶段 PLAN 中定义

### 已建立的模式
- 后端：Encore.go 框架规范（服务定义、API 注解、sqldb、errs 包、auth handler）
- Encore 内置 TypeScript 客户端生成（`encore gen client --lang=typescript`）
- JWT Cookie 认证方式（Phase 1 已设计）

### 集成点
- Encore gen client 生成的 TS 客户端连接所有后端 API
- JWT Cookie 认证对接 Phase 1 的 auth handler
- 后端所有 API 响应遵循 Encore 默认规范（成功直接返回数据，失败返回 {code, message, details}）
- 前端需要对接的核心后端服务：auth、billing、routing、compliance、callback、recording、webhook

</code_context>

<deferred>
## 延后的想法

无——讨论保持在阶段范围内

</deferred>

---

*Phase: 04-admin-dashboard*
*上下文收集日期: 2026-03-10*
