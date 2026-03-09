# Stack Research

**Domain:** API-driven callback/click-to-call system (VoIP B2B platform)
**Researched:** 2026-03-09
**Confidence:** HIGH (core stack validated against official docs and current releases)

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Encore.go | latest (Go 1.22+) | Backend framework, API layer, infra provisioning | PRD-mandated. Declarative infra (DB, PubSub, cron, cache) eliminates boilerplate. Auto-generates typed TS client for frontend via `encore gen client`. Built-in tracing, auth middleware, secrets management. Handles DB migrations, service discovery, and structured errors natively. |
| FreeSWITCH | 1.10.12 | SIP signalling + media (originate, bridge, park, record) | PRD-mandated. Current stable release with security fixes. The canonical open-source softswitch for programmatic callback/originate workflows. No viable alternative: Asterisk lacks ESL-level programmatic control for multi-step originate sequences; Opal is dead. FS handles thousands of concurrent calls on commodity hardware. |
| PostgreSQL | 16+ (Encore-managed) | Primary relational store (users, calls, CDR, billing, routing) | Encore provisions and manages PG automatically (Docker locally, cloud PG in production). Row-level locking (`SELECT ... FOR UPDATE`) for atomic balance deduction. JSONB for flexible webhook payloads. Partial indexes for active-call queries. |
| Redis | 7+ | Concurrent call slot counting, session cache, rate limiting | Encore cache primitives backed by Redis. INCR atomicity essential for concurrent slot control. Lua scripting for atomic check-and-increment. Key-per-call with TTL pattern prevents counter drift. |
| React | 19.2.x (19.2.4 current) | Frontend SPA (Admin Dashboard + Client Portal) | Current stable (released Jan 2026). Hooks + Suspense patterns are mature. Server components not needed for SPA architecture. Massive ecosystem, stable API surface. |
| Vite | 7.x (7.3.x current) | Frontend build tool | Current stable. Sub-second HMR, native ESM dev server. Do NOT use Vite 8 beta (Rolldown bundler engine still stabilizing, not production-ready). |
| TypeScript | 5.7+ | Type safety across frontend | Non-negotiable for production React. Encore `gen client` outputs fully typed TS types matching backend API contracts. |

### Database & Query Layer

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| sqlc | 1.30.0 (Jan 2025) | SQL-first type-safe query generation | PRD-mandated. Generates Go structs + query functions from raw SQL. Zero runtime overhead (compile-time generation). Validates SQL against actual schema at generation time -- catches typos in column names before runtime. |
| Encore sqldb | built-in | Database provisioning, migrations, connection management | Use for: `sqldb.NewDatabase()` declaration, migration files (numbered `.up.sql`), and `db.Stdlib()` bridge to sqlc. |

**Critical integration pattern -- sqlc + Encore sqldb:**

sqlc generates a `DBTX` interface requiring `ExecContext`, `QueryContext`, `QueryRowContext`. Encore's `sqldb.Database` does NOT directly implement this interface (Encore uses `Exec(ctx, ...)` not `ExecContext(ctx, ...)`). The bridge is `db.Stdlib()` which returns a standard `*sql.DB` that DOES satisfy DBTX.

```go
// In each service that needs DB access:
var db = sqldb.NewDatabase("bos3000", sqldb.DatabaseConfig{
    Migrations: "./migrations",
})

// Bridge to sqlc-generated queries
func newQueries() *sqlcgen.Queries {
    return sqlcgen.New(db.Stdlib())
}

// For transactions, use db.Stdlib() to get *sql.DB, then use standard sql.Tx
func withTx(ctx context.Context, fn func(*sqlcgen.Queries) error) error {
    tx, err := db.Stdlib().BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()
    if err := fn(sqlcgen.New(tx)); err != nil { return err }
    return tx.Commit()
}
```

**sqlc configuration (`sqlc.yaml`):**
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "./queries/"
    schema: "./migrations/"
    gen:
      go:
        package: "sqlcgen"
        out: "./sqlcgen"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_empty_slices: true
```

### ESL / Telephony

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| eslgo (percipia/eslgo) | v1.5.0 (Dec 2025) | Go FreeSWITCH ESL client (inbound mode) | PRD-mandated. Only actively maintained Go ESL library. Idiomatic Go with context support, production-tested at thousands of calls/sec per maintainer claim. Supports inbound ESL, outbound ESL server, event listeners by UUID, originate helpers, DTMF, answer/hangup. |

**eslgo status assessment (HIGH confidence):**
- v1.5.0 released 2025-12-10 on GitHub, 11 releases total across v1 branch
- 130 GitHub stars, 53 forks, 123 commits, MPL-2.0 license
- Module path: `github.com/percipia/eslgo`
- Risk: Small community. Mitigation: PRD mandates shallow wrapper (FSClient with 5 methods), replacement cost < 100 lines of Go code
- Alternative `vma/esl` exists but less idiomatic API and lower maintenance activity

### Frontend UI

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| shadcn/ui | CLI v4 (March 2026) | Component library (tables, forms, dialogs, charts) | Copy-paste ownership model -- zero dependency lock-in. CLI v4 released March 2026 with agent-friendly skills system and presets. Now uses unified `radix-ui` package (Feb 2026 change). Massive ecosystem of blocks and templates. Perfect for admin dashboards with data tables, forms, and dialogs. |
| Tailwind CSS | 4.1+ (v4.0 released Jan 2025) | Utility-first styling | Ground-up rewrite with CSS-first config (no `tailwind.config.js`). 5x faster full builds, 100x faster incremental builds via Lightning CSS engine. Native cascade layers, container queries. shadcn/ui requires Tailwind. |
| TanStack Router | latest | Client-side routing for SPA | Type-safe routing with search params, nested layouts, data loading with caching. Superior to React Router v7 for SPA use case because RR v7's best features (type safety, streaming) only work in framework mode (SSR). Dashboard needs complex nested routes + typed search params for filters, pagination, date ranges. |
| TanStack Query | v5 | Server state management, API call caching | Pairs directly with Encore's generated TS client. Handles caching, background refetching, stale-while-revalidate, optimistic updates. Encore docs explicitly recommend TanStack Query for React frontend integration. |
| Zustand | 5.x (~1KB) | Client-side UI state | No providers needed, selector-based re-render control. Better than Jotai for interconnected dashboard state (active filters affecting multiple views, sidebar state, WebSocket connection state, notification counts). Better than Redux for this scale (no boilerplate). |
| Recharts | 2.x | Dashboard charts (call volume, revenue, loss analysis) | Built on D3, React-native component API. shadcn/ui has built-in Recharts chart components. Simpler than Apache ECharts for needed chart types (line, bar, pie, area). |

### MagicUI Assessment

| Verdict | Details |
|---------|---------|
| **USE SELECTIVELY, NOT AS PRIMARY** | MagicUI (magicui.design) provides 150+ animated components (backgrounds, text effects, transitions, animated counters). Same copy-paste model as shadcn/ui. 19K+ GitHub stars, MIT license. **Good for:** animated number counters on dashboards, shimmer loading states, visual polish on landing/login pages. **Bad for:** core CRUD UI, data tables, forms (shadcn/ui handles these better). Use shadcn/ui as the primary component system; cherry-pick 3-5 MagicUI components for visual flair. |

### Frontend Tooling

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Turborepo | 2.7+ (Jan 2026) | Monorepo task orchestration | PRD specifies frontend in same repo as backend. Turborepo handles: parallel builds across Admin + Portal apps, shared UI package, intelligent caching (local + remote). Composable configuration (v2.7) simplifies per-app turbo.json overrides. Devtools for debugging cache behavior. |
| pnpm | 9.x | Package manager | Faster installs than npm/yarn, strict dependency hoisting (prevents phantom dependency bugs), native workspace support for Turborepo monorepo. Content-addressable store saves disk space. |

### Infrastructure & DevOps

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Docker | latest | FreeSWITCH containerization, local dev | FS 1.10.12 has official Debian 12 packages (AMD64/ARM64). Containerize for reproducible dev/staging. |
| ffmpeg | 6.x+ | A/B track recording merge | PRD requirement. CLI tool called via `os/exec` after bridge ends. Use `-itsoffset` for A/B track alignment based on FS CDR timestamps. |
| MinIO / S3 | - | Recording storage | Encore `objects.NewBucket()` for production S3. MinIO for local dev with S3-compatible API. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| gorilla/websocket | v1.5+ | WebSocket for real-time call status push | Encore raw endpoints for WS upgrade. Admin Dashboard + Client Portal for live call monitoring. |
| golang-jwt/jwt | v5 | JWT token generation/validation | Auth handler for API Key + JWT dual auth mode. Encore `//encore:authhandler` integration. |
| go-playground/validator | v10 | Struct validation | Request validation in Encore `Validate()` methods. Chained field validators (email, required, min/max). |
| slog (stdlib) | Go 1.21+ | Structured logging | Use Go stdlib `log/slog` (not zap/zerolog). Encore has built-in logging; slog for FSClient wrapper debug output where Encore logging is not available. |

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| sqlc | GORM | Never for this project. GORM adds runtime reflection overhead, magic callbacks, and hides SQL behind method chains. For a billing system where every query must be auditable and SQL must be precise, sqlc's SQL-first approach is the only sane choice. GORM is acceptable for rapid prototyping where query precision doesn't matter. |
| sqlc | sqlx | If you want raw SQL without code generation. sqlc is strictly better -- same SQL-first approach but with compile-time type checking and struct generation. sqlx requires manual struct tags and Scan calls. |
| eslgo (percipia) | vma/esl | If eslgo becomes abandoned. vma/esl is functional but has a less idiomatic Go API. Monitor eslgo GitHub activity quarterly. |
| eslgo | goesl (0x19/goesl) | Never. Last meaningful commit years ago, not context-aware, no outbound server support. Effectively abandoned. |
| TanStack Router | React Router v7 | If team has deep React Router experience and doesn't need type-safe search params. RR v7's framework mode is strong but designed for SSR (Remix heritage), overkill for SPA dashboard. RR v7 library mode lacks the type safety and data loading features that compete with TanStack Router. |
| Zustand | Redux Toolkit | For very large teams (10+ frontend devs) needing strict action/reducer patterns and devtools time-travel. Overkill for this project's 2-3 frontend developers. |
| Zustand | Jotai | For highly granular atomic state where individual atoms need independent Suspense boundaries (e.g., complex form builders). Dashboard state is more interconnected (filters affect multiple views simultaneously), which suits Zustand's centralized store model better. |
| Turborepo | Nx | For enterprise monorepos with 50+ packages needing dependency graph visualization, code generators, and module boundary enforcement. Nx is more powerful but significantly heavier configuration. Turborepo is right-sized for 2-3 frontend apps + 1-2 shared packages. |
| Vite SPA | Next.js | If SSR/ISR was needed for SEO. This is an admin dashboard behind authentication -- zero SEO requirement. Vite SPA is simpler architecture, faster builds, no server component complexity, no hydration bugs. |
| FreeSWITCH | Asterisk | If only doing simple IVR or basic PBX. Asterisk's AMI (manager interface) is inferior to FreeSWITCH ESL for complex programmatic call control. FS `originate` + `park` + `uuid_bridge` is the canonical pattern for callback systems. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| GORM | Runtime reflection, implicit behaviors (hooks, auto-migrations), query debugging nightmare for billing-critical SQL. Hidden N+1 queries. | sqlc (SQL-first, compile-time type safety, zero runtime cost) |
| Next.js (for this project) | SSR complexity unnecessary for auth-gated dashboards. Server component mental model overhead. Hydration mismatch bugs. Deployment complexity (needs Node.js server). | Vite + React SPA (static deploy, simpler architecture) |
| Encore.ts | Project is Go-based per PRD. Encore.ts is a completely separate framework with different primitives, not interchangeable with Encore.go. | Encore.go |
| database/sql directly | No type safety, manual `Scan()` boilerplate for every query, error-prone field ordering, no compile-time SQL validation. | sqlc generating code, bridged to Encore via `db.Stdlib()` |
| Redux / Redux Toolkit | Boilerplate heavy (actions, reducers, selectors, thunks). Global store pattern overkill for dashboard UI state. ~15KB bundle. | Zustand (~1KB, no boilerplate, no providers) |
| Ant Design / Material UI | Heavy bundle (200KB+), opinionated styling conflicts with Tailwind CSS, runtime CSS-in-JS overhead, not copy-paste ownership model. | shadcn/ui (Tailwind-native, own the code, tree-shakeable) |
| Socket.IO | Unnecessary abstraction over WebSocket, adds 40KB+ client bundle, auto-reconnect logic conflicts with custom connection management. | gorilla/websocket (raw WS, lighter, full control) |
| OpenSIPS | PRD already decided against it. Callback is FS originate's sweet spot. OpenSIPS B2BUA + rtpengine adds two more components (3 total vs 1) for the same functionality. | FreeSWITCH ESL directly |
| Vite 8 beta | Rolldown bundler engine still stabilizing. Not production-ready. Breaking changes expected before stable release. | Vite 7.x stable (7.3.x) |

## Monorepo Structure

```
bos3000/
  encore.app                   # Encore application config
  go.mod
  go.sum
  sqlc.yaml                    # sqlc configuration (project root)

  # Backend (Encore.go services)
  auth/                        # Auth service (JWT, API key, roles)
    migrations/
    queries/                   # sqlc SQL files
    sqlcgen/                   # sqlc generated code
  callback/                    # Callback initiation API
  callengine/                  # Core ESL call control + state machine
  billing/                     # Balance, transactions, rate plans
    migrations/
    queries/
    sqlcgen/
  routing/                     # Gateway routing (prefix + round-robin)
    migrations/
    queries/
    sqlcgen/
  cdr/                         # CDR queries, loss analysis
  recording/                   # Recording merge + S3 upload
  webhook/                     # Webhook delivery + retry + DLQ
    migrations/
    queries/
    sqlcgen/

  # Frontend (Turborepo)
  frontend/
    turbo.json
    package.json               # pnpm workspace root
    pnpm-workspace.yaml
    packages/
      ui/                      # Shared shadcn/ui components
      api-client/              # Encore gen client output (auto-generated TS types)
    apps/
      admin/                   # Admin Dashboard (Vite + React + TanStack Router)
        vite.config.ts
        src/
      portal/                  # Client Portal (Vite + React + TanStack Router)
        vite.config.ts
        src/
```

## Installation

### Backend

```bash
# Install Encore CLI
curl -L https://encore.dev/install.sh | bash

# Install sqlc
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

# Install Go dependencies (managed by go.mod)
go mod tidy

# Run locally
encore run
```

### Frontend

```bash
# From frontend/ directory
corepack enable
pnpm install

# Initialize shadcn/ui in each app
cd apps/admin && pnpm dlx shadcn@latest init
cd apps/portal && pnpm dlx shadcn@latest init

# Generate Encore API client (run after backend API changes)
encore gen client <APP_ID> --output=./packages/api-client/src/client.ts --lang=typescript

# Dev server
pnpm dev        # runs both admin + portal via Turborepo
```

### FreeSWITCH (Docker)

```bash
# FreeSWITCH 1.10.12 on Debian 12
docker pull signalwire/freeswitch:1.10.12

# Or build custom image with recording modules + custom ESL config
docker build -t bos3000-fs ./docker/freeswitch/
```

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| Encore.go | Go 1.22+ | Encore requires modern Go features (generics, slog) |
| sqlc 1.30.0 | PostgreSQL 16, Go 1.22+ | Generates code using `database/sql` interface. Works with Encore via `db.Stdlib()` |
| eslgo v1.5.0 | FreeSWITCH 1.10.x ESL | ESL protocol is stable across all FS 1.10.x releases |
| React 19.2 | Vite 7.x, TanStack Router/Query v5 | No known conflicts. Concurrent features stable. |
| shadcn/ui CLI v4 | Tailwind CSS 4.x, React 19, radix-ui unified | Uses unified `radix-ui` package (not individual `@radix-ui/react-*`) |
| Tailwind CSS 4.1 | Vite 7.x | CSS-first config, no PostCSS plugin needed, Lightning CSS engine |
| Turborepo 2.7+ | pnpm 9.x workspaces | Composable configuration for per-app turbo.json overrides |
| TanStack Router | React 19, Vite 7.x | File-based or code-based route definitions both supported |

## Stack Patterns by Variant

**If call volume < 50 concurrent calls:**
- Single FreeSWITCH instance, single ESL connection
- Redis optional (can use PG advisory locks for slot counting)
- Recording merge can be synchronous in call finalization path

**If call volume 50-500 concurrent calls:**
- Single FreeSWITCH (handles thousands), single Encore process
- Redis required for slot counting (PG advisory locks too slow)
- Recording merge must be async (PubSub job queue)
- Monitor PG connection pool utilization

**If call volume > 500 concurrent calls:**
- FreeSWITCH HA pair (active-standby)
- ESL connection pool (2-4 connections)
- PG read replicas for dashboard/analytics queries
- Consider partitioning callback_calls table by date

## Sources

- [percipia/eslgo GitHub](https://github.com/percipia/eslgo) -- v1.5.0 release Dec 2025, 130 stars, actively maintained (HIGH confidence)
- [eslgo on pkg.go.dev](https://pkg.go.dev/github.com/percipia/eslgo) -- Published Dec 10, 2025 (HIGH confidence)
- [sqlc changelog v1.30.0](https://docs.sqlc.dev/en/stable/reference/changelog.html) -- Released Jan 20, 2025 (HIGH confidence)
- [Encore sqldb package docs](https://pkg.go.dev/encore.dev/storage/sqldb) -- `Stdlib()` returns `*sql.DB`, `RegisterStdlibDriver()` for third-party libs (HIGH confidence)
- [Encore frontend integration docs](https://encore.dev/docs/go/how-to/integrate-frontend) -- `encore gen client` + TanStack Query recommended pattern (HIGH confidence)
- [Encore blog on sqlc](https://encore.dev/blog/go-get-it-001-sqlc) -- Encore team uses sqlc internally (HIGH confidence)
- [FreeSWITCH 1.10.x release notes](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Release-Notes/FreeSWITCH-1.10.x-Release-notes_25460878/) -- 1.10.12 current stable (HIGH confidence)
- [shadcn/ui changelog](https://ui.shadcn.com/docs/changelog) -- CLI v4 March 2026, unified radix-ui Feb 2026 (HIGH confidence)
- [shadcn/ui visual builder](https://www.infoq.com/news/2026/02/shadcn-ui-builder/) -- npx shadcn create (HIGH confidence)
- [Tailwind CSS v4 announcement](https://tailwindcss.com/blog/tailwindcss-v4) -- v4.0 Jan 2025, 5x faster builds (HIGH confidence)
- [Vite releases page](https://vite.dev/releases) -- v7.3.x current stable, v8 beta available (HIGH confidence)
- [React versions page](https://react.dev/versions) -- 19.2.4 current stable, released Jan 26, 2026 (HIGH confidence)
- [Turborepo releases](https://github.com/vercel/turborepo/releases) -- v2.7+ with composable config, Devtools, Biome rule (HIGH confidence)
- [MagicUI](https://magicui.design/) -- 150+ components, 19K+ stars, MIT license (MEDIUM confidence on long-term maintenance)
- [React state management comparison 2026](https://inhaq.com/blog/react-state-management-2026-redux-vs-zustand-vs-jotai.html) -- Zustand recommended for interconnected state (MEDIUM confidence)
- [TanStack Router vs React Router v7](https://medium.com/ekino-france/tanstack-router-vs-react-router-v7-32dddc4fcd58) -- TanStack better for SPA, RR v7 best in framework mode (MEDIUM confidence)
- [Encore Go ORMs comparison 2026](https://encore.cloud/resources/go-orms) -- Encore team analysis of Go data access patterns (HIGH confidence)

---
*Stack research for: BOS3000 API callback/click-to-call system*
*Researched: 2026-03-09*
