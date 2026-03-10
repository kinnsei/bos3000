# Phase 4: Admin Dashboard - Research

**Researched:** 2026-03-10
**Domain:** React SPA with shadcn/ui, Recharts, TanStack Query/Table, connecting to Encore.go backend
**Confidence:** HIGH

## Summary

Phase 4 builds the Admin Dashboard as a standalone React SPA in the `/admin` directory, consuming the Encore.go backend APIs from Phases 1-3 via a generated TypeScript client. The stack is locked: React 18 + Vite + TypeScript + Tailwind CSS v4 + shadcn/ui + TanStack Query v5 + TanStack Table + React Router v7 (library mode) + Recharts 3.

The Encore ecosystem provides `encore gen client` to generate a fully typed TypeScript client from the Go backend, which pairs directly with TanStack Query for data fetching, caching, and mutations. shadcn/ui provides the complete component library including a Sidebar component and DataTable pattern built on TanStack Table. MagicUI provides animated components (NumberTicker, AnimatedList, etc.) that follow the same copy-paste philosophy as shadcn/ui.

**Primary recommendation:** Use React Router v7 in **library mode** (createBrowserRouter) rather than framework mode, since this is a pure SPA with no SSR needs. Use shadcn/ui's official Sidebar block for the collapsible sidebar layout. Wrap the Encore-generated client with TanStack Query hooks in a `/lib/api/` layer. Structure pages by feature domain (dashboard, customers, gateways, cdr, wastage, finance, compliance, ops, settings).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- 采用左侧可收缩侧边栏 + 顶部导航栏的经典后台布局
- 侧边栏菜单按功能分组：监控（概览、话单、损耗）、管理（客户、网关、DID）、财务（财务中心）、系统（合规、运维、设置）
- 顶部导航栏放置：Logo、主题切换、通知图标、管理员信息
- 桌面优先设计，主要保证 1024px+ 体验，平板可用，手机端不做专门优化
- 参考 Vercel Dashboard 风格：现代感、黑白为主、强调数据和状态
- 主题：跟随操作系统暗色/亮色设置，支持手动切换
- 主色调：绿/青色系，用于按钮、选中态、链接等强调元素
- MagicUI 动效克制点缀：仅在关键位置使用（运营大盘数字滚动、告警卡片闪烁、页面切换过渡）
- 加载状态使用 Skeleton 骨架屏 + 动画过渡
- 中文为主（菜单、标题、按钮、提示文字全部中文），v1 不做 i18n
- API 错误信息为英文（Phase 1 已决策），前端转换为中文提示
- 实时并发数、今日收入/损耗、桥接成功率作为核心指标卡片展示
- 四类告警卡片：桥接成功率低于阈值、客户余额不足、网关 DOWN、损耗率异常
- 数据刷新方式：每 30s 自动轮询刷新概览数据
- 统一使用 Recharts，图表风格与 Vercel Dashboard 保持一致
- 图表配色采用多色彩方案（绿/青/紫/橙/红）
- 简单操作（充值、冻结、上下线）使用 Dialog 弹窗
- 复杂操作（开户、网关配置）使用侧抽屉 Sheet
- 危险操作使用红色警告二次确认弹窗
- 表格列表统一使用 shadcn DataTable（基于 TanStack Table）
- DID 号码和黑名单批量导入使用 CSV/Excel 文件上传（预览数据 + 验证后确认导入）
- 前端项目放在仓库根目录 /admin 子目录
- 技术栈：React 18 + Vite 5 + shadcn/ui + Tailwind CSS 4 + TypeScript
- 状态管理：TanStack Query 管理服务端数据，局部 UI 状态用 React useState/useReducer
- 路由：React Router v7（纯 SPA 模式）
- API 客户端：使用 `encore gen client --lang=typescript` 生成类型安全的 TS 客户端
- 认证流程：JWT Cookie（HttpOnly, Secure）登录页

### Claude's Discretion
- shadcn/ui 组件的具体选择和组合
- MagicUI 动效的具体组件选用
- Recharts 图表的具体配置和交互细节
- 前端目录结构和文件组织方式
- 登录页的具体设计
- Skeleton 骨架屏的具体样式
- 表格列的默认配置
- 告警阈值的默认值

### Deferred Ideas (OUT OF SCOPE)
无
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| ADMN-01 | Overview 运营大盘：实时并发、今日收入/损耗、桥接成功率、告警卡片 | TanStack Query with 30s refetchInterval; Recharts AreaChart/BarChart; MagicUI NumberTicker for animated counters |
| ADMN-02 | 客户管理：列表、详情、开户、充值/扣款、冻结、API Key 查看 | shadcn DataTable with server-side pagination; Sheet for create form; Dialog for balance ops; connects to auth + billing APIs |
| ADMN-03 | 网关管理：A/B 路网关池、健康状态实时同步、手动上下线、容灾配置、测试外呼 | DataTable with health status badges; Dialog for toggle/test; Sheet for gateway config; routing service APIs |
| ADMN-04 | 话单管理：全量话单查询、实时通话监控、强制挂断 | DataTable with date range + multi-field filters; AlertDialog for hangup confirmation; callback service APIs |
| ADMN-05 | 损耗分析中心：平台损耗趋势图、客户损耗排名、B 路失败原因分布 | Recharts LineChart (trend), BarChart (ranking), PieChart (failure reasons); billing/callback APIs |
| ADMN-06 | 财务中心：对账、全平台流水、费率模板管理、毛利分析 | DataTable for transactions; Sheet for rate plan CRUD; Recharts for profit analysis; billing service APIs |
| ADMN-07 | DID 管理：号码列表、批量导入、分配管理 | DataTable + CSV upload with preview; Papa Parse for CSV parsing; routing service APIs |
| ADMN-08 | 合规：全局黑名单管理、审计日志查询 | DataTable for both; CSV upload for blacklist bulk import; compliance service APIs |
| ADMN-09 | 运维工具：FreeSWITCH 状态、ESL 连接健康度、系统健康监控 | Status cards with health indicators; gateway service health APIs |
| ADMN-10 | 系统设置：system_configs 可视化编辑 | Form with dynamic field rendering based on config schema |
| WAST-03 | 管理员可查看平台级损耗趋势、客户损耗排名、B 路失败原因分布 | Same as ADMN-05 - Recharts visualizations against wastage analysis APIs |
| UI-01 | 专业商业级水准：shadcn/ui + MagicUI 动效点缀 | shadcn/ui components + selective MagicUI animations (NumberTicker, AnimatedList, ShimmerButton) |
| UI-02 | 数据可视化专业：Recharts 图表风格统一、配色协调 | Recharts with custom theme config matching Vercel palette; shared chart wrapper component |
| UI-03 | 响应式布局，暗色/亮色主题切换，加载状态优雅 | Tailwind dark mode class strategy; shadcn Skeleton; CSS media query prefers-color-scheme |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| react | 18.x | UI framework | Locked by user decision; stable, wide ecosystem |
| vite | 6.x | Build tool | Latest stable; Vite 5 specified but 6 is current and backward compatible |
| typescript | 5.x | Type safety | Required for type-safe Encore client integration |
| tailwindcss | 4.x | Utility CSS | Locked; v4 uses `@tailwindcss/vite` plugin, no config file needed |
| @tailwindcss/vite | 4.x | Vite integration | First-party Tailwind v4 Vite plugin; zero-config |
| shadcn/ui | latest (CLI v4) | Component library | Locked; copy-paste components, uses unified `radix-ui` package |
| react-router | 7.x | Client routing | Locked; use library mode with `createBrowserRouter` |
| @tanstack/react-query | 5.x | Server state management | Locked; v5.90+ current, pairs with Encore client |
| @tanstack/react-table | 8.x | Table/DataTable logic | Required by shadcn DataTable pattern |
| recharts | 3.x | Charts/visualization | Locked; v3.8 current, React + D3 based |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| framer-motion (motion) | 11.x | Animation engine | Required by MagicUI components |
| papaparse | 5.x | CSV parsing | DID/blacklist bulk import (ADMN-07, ADMN-08) |
| xlsx (sheetjs) | 0.20.x | Excel parsing | Excel file import support |
| date-fns | 3.x | Date formatting | CDR date ranges, transaction timestamps |
| lucide-react | latest | Icons | shadcn/ui default icon set |
| class-variance-authority | latest | Component variants | shadcn/ui dependency |
| clsx + tailwind-merge | latest | Class merging | shadcn/ui cn() utility |
| sonner | latest | Toast notifications | shadcn/ui recommended toast component |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| React Router library mode | React Router framework mode (ssr:false) | Framework mode adds complexity (file-based routing, react-router.config.ts) unnecessary for pure admin SPA |
| Recharts | Nivo / Tremor | Recharts is locked decision; Tremor is built on Recharts anyway |
| Papa Parse | native FileReader | Papa Parse handles CSV edge cases (quoted fields, encodings) that manual parsing misses |

**Installation:**
```bash
# In /admin directory
npm create vite@latest . -- --template react-ts
npm install react-router @tanstack/react-query @tanstack/react-table recharts
npm install framer-motion papaparse date-fns sonner
npm install -D @tailwindcss/vite @types/papaparse

# Initialize shadcn/ui
npx shadcn@latest init
```

## Architecture Patterns

### Recommended Project Structure
```
admin/
  src/
    app/                    # App shell, providers, router
      providers.tsx         # QueryClient, ThemeProvider, RouterProvider
      router.tsx            # createBrowserRouter route definitions
      layout.tsx            # Sidebar + TopNav + Outlet shell
    components/
      ui/                   # shadcn/ui components (auto-generated)
      shared/               # Reusable composed components
        data-table.tsx      # Generic DataTable wrapper
        stat-card.tsx       # Metric card component
        chart-wrapper.tsx   # Recharts theme/config wrapper
        confirm-dialog.tsx  # Danger action confirmation
        csv-upload.tsx      # CSV/Excel upload + preview
        skeleton-page.tsx   # Full-page skeleton loader
    features/
      dashboard/            # ADMN-01: Overview
      customers/            # ADMN-02: Customer management
      gateways/             # ADMN-03: Gateway management
      cdr/                  # ADMN-04: Call detail records
      wastage/              # ADMN-05/WAST-03: Wastage analysis
      finance/              # ADMN-06: Financial center
      did/                  # ADMN-07: DID management
      compliance/           # ADMN-08: Blacklist + audit
      ops/                  # ADMN-09: Operations/monitoring
      settings/             # ADMN-10: System settings
      auth/                 # Login page
    lib/
      api/
        client.ts           # Encore generated client (auto-generated)
        hooks.ts            # TanStack Query hooks wrapping client
        error.ts            # Error code to Chinese message mapping
      theme/
        colors.ts           # Vercel-style color palette tokens
        chart-theme.ts      # Recharts shared config
      utils.ts              # cn() and helpers
    styles/
      globals.css           # Tailwind import + CSS variables
  index.html
  vite.config.ts
  tsconfig.json
  package.json
```

### Pattern 1: Encore Client + TanStack Query Integration
**What:** Wrap generated Encore client calls in TanStack Query hooks for caching, refetching, and mutations.
**When to use:** Every API call from the dashboard.
**Example:**
```typescript
// lib/api/hooks.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import Client from './client'; // Encore generated

const api = new Client(window.location.origin);

// Query hook pattern
export function useCustomers(params: { page: number; limit: number }) {
  return useQuery({
    queryKey: ['customers', params],
    queryFn: () => api.auth.ListUsers(params),
  });
}

// Mutation hook pattern
export function useTopUpCustomer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { userId: string; amount: number }) =>
      api.billing.TopUp(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] });
    },
  });
}

// Auto-refresh pattern (30s polling for dashboard)
export function useDashboardOverview() {
  return useQuery({
    queryKey: ['dashboard', 'overview'],
    queryFn: () => api.billing.GetOverview(),
    refetchInterval: 30_000,
  });
}
```

### Pattern 2: Reusable DataTable with Server-Side Operations
**What:** Generic DataTable component wrapping TanStack Table + shadcn Table with server-side pagination, sorting, and filtering.
**When to use:** Every list page (customers, CDR, DID, blacklist, audit logs, transactions).
**Example:**
```typescript
// components/shared/data-table.tsx
import {
  useReactTable,
  getCoreRowModel,
  type ColumnDef,
  type PaginationState,
  type SortingState,
} from '@tanstack/react-table';

interface DataTableProps<TData> {
  columns: ColumnDef<TData>[];
  data: TData[];
  totalCount: number;
  pagination: PaginationState;
  onPaginationChange: (state: PaginationState) => void;
  sorting?: SortingState;
  onSortingChange?: (state: SortingState) => void;
  isLoading?: boolean;
}

// Each feature page defines columns and uses this component
// with useQuery handling the server-side data fetching
```

### Pattern 3: Error Code Translation
**What:** Map Encore backend error codes (English) to Chinese user-facing messages.
**When to use:** All API error handling in mutations and queries.
**Example:**
```typescript
// lib/api/error.ts
const ERROR_MESSAGES: Record<string, string> = {
  INSUFFICIENT_BALANCE: '余额不足，请先充值',
  BLACKLISTED_NUMBER: '该号码已被加入黑名单',
  RATE_LIMIT_EXCEEDED: '已达到今日呼叫限额',
  CONCURRENCY_LIMIT_EXCEEDED: '并发数已达上限',
  INVALID_CREDENTIALS: '用户名或密码错误',
  // ... all error codes from Phase 1
};

export function getErrorMessage(code: string): string {
  return ERROR_MESSAGES[code] || '操作失败，请稍后重试';
}
```

### Pattern 4: Theme System (Vercel-Style)
**What:** CSS custom properties for dark/light theme with green/cyan accent.
**When to use:** Applied globally, consumed by all components.
**Example:**
```css
/* globals.css */
@import "tailwindcss";

:root {
  --background: 0 0% 98%;
  --foreground: 0 0% 9%;
  --primary: 160 84% 39%;      /* Green/cyan accent */
  --primary-foreground: 0 0% 98%;
  /* ... shadcn CSS variables */
}

.dark {
  --background: 0 0% 4%;
  --foreground: 0 0% 95%;
  --primary: 163 72% 47%;
  /* ... dark mode overrides */
}
```

### Pattern 5: CSV/Excel Import with Preview
**What:** Upload flow: file select -> parse -> preview table -> validate -> confirm -> submit.
**When to use:** DID bulk import (ADMN-07), blacklist bulk import (ADMN-08).
**Example:**
```typescript
// components/shared/csv-upload.tsx
// 1. Accept file via <Input type="file" accept=".csv,.xlsx" />
// 2. Parse with Papa Parse (CSV) or SheetJS (Excel)
// 3. Display preview in DataTable (first 10 rows)
// 4. Validate fields (phone format, required columns)
// 5. Show validation errors inline
// 6. On confirm, batch POST to backend API
```

### Anti-Patterns to Avoid
- **Global state for server data:** Do NOT use Redux/Zustand for data from APIs. TanStack Query handles caching, refetching, and invalidation. Use useState only for UI state (modal open, form values).
- **Inline API calls without query keys:** Every API call must go through TanStack Query with proper queryKey for cache invalidation to work.
- **Custom fetch wrappers:** Do NOT write custom fetch/axios wrappers. Use the Encore-generated client which is already typed and handles auth cookies.
- **Hardcoded API URLs:** The Encore client constructor takes the base URL. Use `window.location.origin` for same-origin or env variable for cross-origin.
- **One giant router file:** Split route definitions by feature domain. Lazy-load feature routes with React.lazy().

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| API client | Custom fetch wrapper | `encore gen client --lang=typescript` | Auto-syncs with backend, full type safety, handles auth |
| Data tables | Custom table with sorting/pagination | shadcn DataTable + TanStack Table | Column definitions, virtualization, accessibility built-in |
| CSV parsing | Manual string splitting | Papa Parse | Handles quoted fields, encoding detection, streaming for large files |
| Toast notifications | Custom notification system | sonner (shadcn recommended) | Stacking, auto-dismiss, promise-based, dark mode support |
| Theme switching | Custom CSS class toggle | shadcn ThemeProvider pattern | System preference detection, localStorage persistence, no flash |
| Form validation | Manual if/else checks | React Hook Form + Zod | Schema-based validation, shadcn Form integration |
| Date range picker | Custom calendar | shadcn DateRangePicker (Calendar + Popover) | Locale, range selection, accessibility |
| Confirmation dialogs | Custom modal | shadcn AlertDialog | Focus trap, keyboard nav, consistent styling |

**Key insight:** shadcn/ui provides pre-composed patterns for nearly every admin dashboard need. Copy them from the docs/blocks, don't reinvent.

## Common Pitfalls

### Pitfall 1: Encore Client Cookie Auth in Dev
**What goes wrong:** Vite dev server runs on port 5173, Encore on 12345. CORS blocks cross-origin cookie auth.
**Why it happens:** HttpOnly cookies require same-origin or explicit CORS credentials config.
**How to avoid:** Configure Vite proxy to forward `/api` requests to Encore backend:
```typescript
// vite.config.ts
export default defineConfig({
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:12345',
        changeOrigin: true,
      }
    }
  }
});
```
Also update `encore.app` CORS config for production domains.
**Warning signs:** Login works but subsequent requests return 401.

### Pitfall 2: TanStack Query Cache Invalidation on Mutations
**What goes wrong:** After create/update/delete, stale data shows in tables.
**Why it happens:** Forgetting to invalidate related query keys after mutations.
**How to avoid:** Every `useMutation` must have `onSuccess` that invalidates relevant queryKeys. Use broad invalidation: `queryClient.invalidateQueries({ queryKey: ['customers'] })` invalidates all customer-related queries.
**Warning signs:** User has to manually refresh to see changes.

### Pitfall 3: Tailwind CSS v4 Configuration
**What goes wrong:** Tailwind classes don't apply, custom colors don't work.
**Why it happens:** v4 eliminates `tailwind.config.js`. Customization moves to CSS. Developers looking for config file find nothing.
**How to avoid:** Use `@import "tailwindcss"` in CSS file. Use `@theme` directive for custom values. The `@tailwindcss/vite` plugin handles everything.
**Warning signs:** Classes in HTML but no styles applied; build warnings about missing config.

### Pitfall 4: React Router v7 Mode Confusion
**What goes wrong:** Importing from wrong packages, framework mode APIs in library mode.
**Why it happens:** React Router v7 docs emphasize framework mode. Library mode uses classic APIs.
**How to avoid:** Use `createBrowserRouter` + `RouterProvider` pattern. Import from `react-router` (not `react-router-dom` which is now `react-router`). Do NOT create `react-router.config.ts` or use file-based routing.
**Warning signs:** Import errors, "loader" functions not firing, missing `@react-router/dev` dependency.

### Pitfall 5: Recharts Responsive Container
**What goes wrong:** Charts render at 0 width/height or overflow container.
**Why it happens:** Recharts needs an explicit-sized parent. `ResponsiveContainer` must have a parent with defined dimensions.
**How to avoid:** Always wrap charts in a div with explicit height (e.g., `h-[300px]`). Use `ResponsiveContainer width="100%" height="100%"` inside.
**Warning signs:** Empty chart area, console warnings about 0 dimensions.

### Pitfall 6: Large Bundle Size from MagicUI
**What goes wrong:** Bundle bloats because MagicUI components import framer-motion.
**Why it happens:** framer-motion is ~32KB gzipped. Multiple MagicUI components each import it.
**How to avoid:** MagicUI is copy-paste, so only copy the 3-4 components actually needed (NumberTicker, AnimatedList). framer-motion is tree-shakeable -- use named imports from `motion/react`. Lazy-load the dashboard page which has most animations.
**Warning signs:** Bundle analyzer shows motion as largest dependency.

## Code Examples

### Encore Client Initialization
```typescript
// lib/api/client.ts - generated by: encore gen client bos3000-bdpi --output=./src/lib/api/client.ts --env=local
// This file is auto-generated. Do not edit manually.
// Re-run the generate command after backend API changes.

// lib/api/hooks.ts - manual wrapper
import Client from './client';

// For development with Vite proxy, use empty string (same origin)
// For production, the SPA is served from the same origin as the API
export const api = new Client('');
```

### QueryClient Configuration
```typescript
// app/providers.tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,    // 5 min default stale time
      retry: 1,                     // Single retry on failure
      refetchOnWindowFocus: false,  // Admin dashboard doesn't need aggressive refetch
    },
  },
});
```

### Route Configuration (Library Mode)
```typescript
// app/router.tsx
import { createBrowserRouter, RouterProvider } from 'react-router';
import { lazy, Suspense } from 'react';
import { Layout } from './layout';
import { LoginPage } from '../features/auth/login';
import { SkeletonPage } from '../components/shared/skeleton-page';

const Dashboard = lazy(() => import('../features/dashboard'));
const Customers = lazy(() => import('../features/customers'));
const Gateways = lazy(() => import('../features/gateways'));
// ... other feature lazy imports

const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/',
    element: <Layout />,  // Sidebar + TopNav
    children: [
      { index: true, element: <Suspense fallback={<SkeletonPage />}><Dashboard /></Suspense> },
      { path: 'customers', element: <Suspense fallback={<SkeletonPage />}><Customers /></Suspense> },
      { path: 'customers/:id', element: <Suspense fallback={<SkeletonPage />}><CustomerDetail /></Suspense> },
      { path: 'gateways', element: <Suspense fallback={<SkeletonPage />}><Gateways /></Suspense> },
      { path: 'cdr', element: <Suspense fallback={<SkeletonPage />}><CDR /></Suspense> },
      { path: 'wastage', element: <Suspense fallback={<SkeletonPage />}><Wastage /></Suspense> },
      { path: 'finance', element: <Suspense fallback={<SkeletonPage />}><Finance /></Suspense> },
      { path: 'did', element: <Suspense fallback={<SkeletonPage />}><DID /></Suspense> },
      { path: 'compliance', element: <Suspense fallback={<SkeletonPage />}><Compliance /></Suspense> },
      { path: 'ops', element: <Suspense fallback={<SkeletonPage />}><Ops /></Suspense> },
      { path: 'settings', element: <Suspense fallback={<SkeletonPage />}><Settings /></Suspense> },
    ],
  },
]);
```

### Recharts Theme Wrapper
```typescript
// lib/theme/chart-theme.ts
export const CHART_COLORS = {
  primary: 'hsl(160, 84%, 39%)',     // Green
  secondary: 'hsl(190, 80%, 45%)',   // Cyan
  tertiary: 'hsl(270, 60%, 55%)',    // Purple
  warning: 'hsl(35, 90%, 55%)',      // Orange
  danger: 'hsl(0, 75%, 55%)',        // Red
  muted: 'hsl(0, 0%, 60%)',          // Gray
};

// Usage in charts - consistent across all visualizations
// <LineChart>
//   <Line dataKey="wastageRate" stroke={CHART_COLORS.danger} />
//   <Line dataKey="bridgeRate" stroke={CHART_COLORS.primary} />
// </LineChart>
```

### MagicUI NumberTicker for Dashboard Metrics
```typescript
// Copy NumberTicker component from magicui.design
// Use in dashboard stat cards:
// <NumberTicker value={concurrentCalls} className="text-3xl font-bold" />
// Only used on the overview dashboard page, not on management pages
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| react-router-dom v6 | react-router v7 (unified package) | 2024 Q4 | Single import package, library/framework modes |
| tailwind.config.js | CSS-first config (@theme directive) | Tailwind v4, 2025 Q1 | No JS config file, @import "tailwindcss" |
| @radix-ui/react-* (individual) | radix-ui (unified package) | shadcn 2026-02 | Single dependency instead of 20+ |
| Recharts 2.x | Recharts 3.x | 2025 | Performance improvements, new chart types |
| cacheTime in TanStack Query | gcTime | TanStack Query v5 | Renamed for clarity |

**Deprecated/outdated:**
- `react-router-dom`: Merged into `react-router` v7
- `tailwind.config.js`: Replaced by CSS `@theme` directive in v4
- Individual `@radix-ui/react-*` packages: Use unified `radix-ui` with shadcn

## Open Questions

1. **Encore Client Base URL in Production**
   - What we know: Vite proxy handles dev. In production, the SPA could be served from the same origin or a different one.
   - What's unclear: How will the admin SPA be deployed? Same-origin with Encore or separate static host?
   - Recommendation: Design for same-origin (empty base URL). If separate hosting needed later, switch to env variable and configure CORS.

2. **Backend API Completeness for Dashboard**
   - What we know: Phase 1-3 plans define service APIs. Some dashboard-specific aggregation endpoints (overview stats, wastage trends) may need new backend endpoints.
   - What's unclear: Whether existing APIs return all data shapes needed (e.g., daily aggregated wastage, profit by customer).
   - Recommendation: Plan should include a task for identifying and creating any missing aggregation/dashboard-specific API endpoints on the backend.

3. **Auth Guard / Protected Routes**
   - What we know: JWT Cookie auth is HttpOnly so JS cannot read it. Auth state must be inferred from a "who am I" API call.
   - What's unclear: Exact endpoint for checking current session.
   - Recommendation: Use a `/auth/me` endpoint (likely exists in Phase 1 auth service). TanStack Query caches the result. Router redirects to `/login` if query fails with 401.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 3.x (Vite-native test runner) |
| Config file | admin/vitest.config.ts (Wave 0) |
| Quick run command | `cd admin && npx vitest run --reporter=verbose` |
| Full suite command | `cd admin && npx vitest run --coverage` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ADMN-01 | Dashboard renders stat cards and charts | unit | `cd admin && npx vitest run src/features/dashboard/ -x` | Wave 0 |
| ADMN-02 | Customer table renders, create form validates | unit | `cd admin && npx vitest run src/features/customers/ -x` | Wave 0 |
| ADMN-03 | Gateway list shows health status badges | unit | `cd admin && npx vitest run src/features/gateways/ -x` | Wave 0 |
| ADMN-04 | CDR table with filters renders correctly | unit | `cd admin && npx vitest run src/features/cdr/ -x` | Wave 0 |
| ADMN-05 | Wastage charts render with mock data | unit | `cd admin && npx vitest run src/features/wastage/ -x` | Wave 0 |
| ADMN-06 | Finance table and rate plan form work | unit | `cd admin && npx vitest run src/features/finance/ -x` | Wave 0 |
| ADMN-07 | CSV upload parses and validates DID data | unit | `cd admin && npx vitest run src/components/shared/csv-upload -x` | Wave 0 |
| ADMN-08 | Blacklist table and audit log query work | unit | `cd admin && npx vitest run src/features/compliance/ -x` | Wave 0 |
| ADMN-09 | Ops status cards render health info | unit | `cd admin && npx vitest run src/features/ops/ -x` | Wave 0 |
| ADMN-10 | Settings form renders and saves | unit | `cd admin && npx vitest run src/features/settings/ -x` | Wave 0 |
| UI-01 | Components render without errors | unit | `cd admin && npx vitest run src/components/ -x` | Wave 0 |
| UI-02 | Chart wrapper applies theme colors | unit | `cd admin && npx vitest run src/lib/theme/ -x` | Wave 0 |
| UI-03 | Theme toggle switches dark/light classes | unit | `cd admin && npx vitest run src/app/ -x` | Wave 0 |
| WAST-03 | Same as ADMN-05 | unit | Same as ADMN-05 | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd admin && npx vitest run --reporter=verbose`
- **Per wave merge:** `cd admin && npx vitest run --coverage`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `admin/vitest.config.ts` -- Vitest configuration with jsdom environment
- [ ] `admin/src/test/setup.ts` -- Test setup (jsdom, mock matchMedia, mock ResizeObserver)
- [ ] `admin/src/test/mocks/api.ts` -- Mock Encore client for unit tests
- [ ] Install: `npm install -D vitest @testing-library/react @testing-library/jest-dom jsdom @testing-library/user-event`

## Sources

### Primary (HIGH confidence)
- [Encore Go frontend integration docs](https://encore.dev/docs/go/how-to/integrate-frontend) - Client generation, CORS, TanStack Query integration
- [shadcn/ui docs](https://ui.shadcn.com/) - Component library, DataTable pattern, Sidebar blocks, CLI v4
- [shadcn/ui changelog 2026-02](https://ui.shadcn.com/docs/changelog/2026-02-radix-ui) - Unified radix-ui package
- [shadcn/ui changelog 2026-03](https://ui.shadcn.com/docs/changelog/2026-03-cli-v4) - CLI v4 release
- [React Router v7 SPA docs](https://reactrouter.com/how-to/spa) - SPA mode configuration
- [React Router modes](https://reactrouter.com/start/modes) - Library vs framework mode
- [Tailwind CSS v4](https://tailwindcss.com/blog/tailwindcss-v4) - v4 changes, Vite plugin
- [TanStack Query npm](https://www.npmjs.com/package/@tanstack/react-query) - v5.90.21 current
- [Recharts npm](https://www.npmjs.com/package/recharts) - v3.8.0 current
- [MagicUI](https://magicui.design/) - Component catalog, copy-paste pattern

### Secondary (MEDIUM confidence)
- [Vite official docs](https://vite.dev/guide/) - Vite 6 setup with React template
- [sadmann7/tablecn](https://github.com/sadmann7/tablecn) - Server-side DataTable reference implementation
- [satnaing/shadcn-admin](https://github.com/satnaing/shadcn-admin) - Admin dashboard template reference

### Tertiary (LOW confidence)
- None - all findings verified with primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries locked by user, versions verified against npm/official docs
- Architecture: HIGH - Patterns well-established (shadcn + TanStack Query + React Router is the standard React admin stack)
- Pitfalls: HIGH - CORS/proxy, cache invalidation, Tailwind v4 config are well-documented common issues

**Research date:** 2026-03-10
**Valid until:** 2026-04-10 (stable ecosystem, 30 days)
