# Project Research Summary

**Project:** BOS3000
**Domain:** API Callback / Click-to-Call (B2B Wholesale VoIP)
**Researched:** 2026-03-09
**Confidence:** HIGH

## Executive Summary

BOS3000 is a B2B wholesale VoIP platform for API-initiated A/B dual-outbound callbacks. This is a well-established telecom pattern: originate A-leg, park it, originate B-leg, bridge them, record, bill. The canonical way to build this is FreeSWITCH ESL for call control, a stateful backend for billing/CDR/routing, and lightweight portals for admin and client self-service. The domain is mature -- Twilio, ASTPP, and Kolmisoft have validated the feature set -- but BOS3000 differentiates with built-in wastage analysis (categorizing why calls fail) and route quality metrics (ASR/ACD per gateway), features that wholesale operators need but generic platforms lack.

The recommended approach is Encore.go for the backend (declarative infrastructure, auto-generated TypeScript client, built-in auth/PubSub/caching), FreeSWITCH 1.10.12 with the eslgo library for call control, sqlc for type-safe SQL, and a React/Vite SPA frontend with shadcn/ui. The critical architectural decision is the Mock FSClient pattern: the entire call engine state machine, billing, and routing can be built and tested without FreeSWITCH by implementing a mock ESL client. This de-risks the hardest part of the system (the A/B call state machine) before introducing real telephony complexity.

The primary risks are: (1) parked A-leg channel leaks if the ESL connection drops -- mitigated by setting `park_timeout` in the originate string itself, not post-park; (2) A/B state machine race conditions where events arrive in unexpected order -- mitigated by serializing events per-call through dedicated goroutines and exhaustive property-based testing with mock events; (3) prepaid balance double-deduction under concurrency -- mitigated by atomic `UPDATE ... WHERE balance >= amount` rather than read-then-write. All three must be solved in the first implementation phase, not retrofitted.

## Key Findings

### Recommended Stack

The stack is PRD-mandated and well-validated. Encore.go handles API layer, database provisioning, PubSub, caching, and auth. FreeSWITCH 1.10.12 is the only viable open-source softswitch for programmatic callback workflows. sqlc generates type-safe Go code from raw SQL, bridged to Encore via `db.Stdlib()`. The frontend uses React 19.2 + Vite 7.x + shadcn/ui + TanStack Router/Query + Zustand, deployed as SPAs (no SSR needed for auth-gated dashboards).

**Core technologies:**
- **Encore.go** (Go 1.22+): Backend framework -- declarative infra, auto-gen TS client, built-in auth/PubSub/secrets
- **FreeSWITCH 1.10.12**: SIP signalling + media -- originate, park, bridge, record via ESL
- **eslgo v1.5.0** (percipia/eslgo): Go ESL client -- only actively maintained option, shallow wrapper makes replacement trivial
- **sqlc 1.30.0**: SQL-first type-safe query generation -- compile-time validation, zero runtime overhead
- **PostgreSQL 16+**: Primary store (Encore-managed) -- row-level locking for atomic balance operations
- **Redis 7+**: Concurrent slot counting, session cache -- atomic INCR via Lua scripts
- **React 19.2 + Vite 7.x + shadcn/ui**: Frontend SPA -- Tailwind-native components, copy-paste ownership
- **TanStack Router + Query**: Type-safe routing and server state -- pairs with Encore gen client
- **Turborepo + pnpm**: Monorepo orchestration -- parallel builds for Admin + Portal apps

**Critical integration note:** sqlc's `DBTX` interface needs `ExecContext/QueryContext`, but Encore sqldb uses `Exec/Query`. Bridge via `db.Stdlib()` which returns standard `*sql.DB`.

### Expected Features

**Must have (table stakes -- v1 launch):**
- API-initiated A/B dual-outbound callback (core product)
- CDR generation with full call lifecycle timestamps
- Per-second/per-minute billing with prefix-based rate plans
- Prepaid balance management with atomic deduction
- Concurrent call limits per client (Redis-backed)
- Call recording (A/B split + ffmpeg merge + S3 upload)
- Webhook notifications with retry + DLQ
- Prefix-based B-leg routing + A-leg round-robin
- Blacklist/number blocking
- Admin dashboard (gateway CRUD, client management, CDR search, financials)
- Client self-service portal (CDR search, balance, recordings, API keys)
- API key authentication with admin/client role separation

**Should have (competitive -- v1.x after first clients):**
- Wastage analysis (A-through-B-fail categorization, short-duration detection)
- ASR/ACD quality metrics per gateway/route with alerting
- DID number management and caller-ID assignment
- WebSocket real-time call status push
- Rate plan versioning with effective dates
- Webhook delivery audit trail in client portal
- FreeSWITCH HA (dual hot-standby)

**Defer (v2+):**
- Multi-tier client hierarchy (reseller model) -- adds query complexity everywhere
- HLR lookup for number validation -- requires external provider
- Batch callback API, cost estimation API

### Architecture Approach

The system is 9 Encore.go services organized around a central Call Engine that owns all FreeSWITCH interaction. The Call Engine maintains a persistent ESL inbound connection, runs per-call state machines, and coordinates with Billing (sync, pre-deduct before originate), Routing (sync, gateway selection before originate), Recording (async via PubSub), and Webhook (async via PubSub). The callback_calls table serves dual duty as live state tracker and CDR -- no separate CDR table, no sync bugs. Transactions table is append-only for audit trail and balance recovery.

**Major components:**
1. **Callback Service** -- thin API layer, validates input, checks balance/slots, delegates to Call Engine
2. **Call Engine Service** -- ESL connection, per-call state machines, event correlation, orchestrates A/B flow
3. **Billing Service** -- atomic balance pre-deduction/finalization/refund, rate plan lookup, transaction ledger
4. **Routing Service** -- prefix-based B-leg gateway selection, round-robin A-leg distribution
5. **Recording Service** -- ffmpeg A/B track merge, S3 upload (async via PubSub)
6. **Webhook Service** -- event delivery with exponential backoff, circuit breaker, DLQ (async via PubSub)
7. **Auth Service** -- JWT/API key dual auth, admin vs client role middleware
8. **Admin Service** -- dashboard APIs for platform operations
9. **Client Service** -- self-service portal APIs scoped to authenticated client

### Critical Pitfalls

1. **Parked A-leg channel leak** -- If ESL connection drops while A is parked, channel lives forever in FS. Set `park_timeout=60` in the originate string itself (not post-park command). Add watchdog sweep for orphaned channels on startup and periodically.

2. **A/B state machine race conditions** -- ESL events are asynchronous; A can hang up while B originate is in-flight. Serialize all events per-call through a dedicated goroutine. Use explicit state machine with valid transition table. Make `finalizeCall` idempotent with `sync.Once`.

3. **Prepaid balance double-deduction** -- Concurrent calls can both pass balance check without proper locking. Use atomic `UPDATE ... WHERE balance >= amount` and check rows_affected. Pre-authorize estimated cost, settle actual cost on completion.

4. **Redis slot counter drift** -- INCR without guaranteed DECR causes permanent inflation. Use key-per-call with TTL pattern (`SET call:{uuid} 1 EX 3600`) instead of bare INCR/DECR. Add periodic reconciliation against DB.

5. **ESL event queue saturation** -- Network partition causes FS to queue 100K+ events, degrading performance. Configure aggressive TCP keepalive (detect dead connection in ~25s). Implement application-level heartbeat every 5s.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Foundation + Core Call Engine (Mock)
**Rationale:** Everything depends on auth, database schema, and the call state machine. The Mock FSClient pattern means this entire phase can be built and tested without FreeSWITCH, de-risking the hardest logic first.
**Delivers:** Working API that accepts callback requests, runs them through the full state machine (mock), performs atomic billing, tracks concurrent slots, generates CDRs, and publishes events.
**Addresses:** Auth, balance management, concurrent call limits, CDR generation, billing/rating, routing logic, blacklist, call state machine design
**Avoids:** Balance double-deduction (Pitfall 4), Redis counter drift (Pitfall 5), state machine races (Pitfall 3 -- tested exhaustively with mock events)

### Phase 2: Real FreeSWITCH Integration
**Rationale:** State machine is proven correct from Phase 1. Now swap Mock FSClient for real eslgo integration. This phase introduces real telephony complexity (ESL events, network issues, timing).
**Delivers:** Actual A/B calls via FreeSWITCH. ESL connection management with heartbeat and reconnection. Gateway health tracking.
**Addresses:** FreeSWITCH ESL integration, real originate/park/bridge flow, gateway health detection
**Avoids:** Parked channel leak (Pitfall 1 -- park_timeout in originate string), ESL queue saturation (Pitfall 2 -- aggressive keepalive), gateway health lag (Pitfall 8 -- app-level health tracking)

### Phase 3: Recording + Webhooks
**Rationale:** Both depend on real calls being bridged (Phase 2). Recording is async (PubSub job). Webhooks are async and independent. Can be built in parallel.
**Delivers:** A/B split recording with ffmpeg merge and S3 upload. Webhook delivery with exponential backoff, circuit breaker, and DLQ.
**Addresses:** Call recording, webhook notifications, recording pipeline, webhook retry storm prevention
**Avoids:** Recording alignment issues (Pitfall 6 -- trigger uuid_record on CHANNEL_BRIDGE only), webhook retry storm (Pitfall 7 -- circuit breaker from day one)

### Phase 4: Admin Dashboard
**Rationale:** Backend APIs are stable from Phases 1-3. Frontend consumes them. Admin portal is higher priority than client portal (operators need visibility first).
**Delivers:** React SPA with gateway CRUD, client management, CDR search, financial overview, system health. Uses Encore gen client for type-safe API integration.
**Addresses:** Admin portal (table stakes), dashboard charts (Recharts), data tables (shadcn/ui)

### Phase 5: Client Portal
**Rationale:** Similar frontend work to Phase 4 but scoped to client self-service. Can reuse shared UI components from Phase 4.
**Delivers:** Client-facing SPA with CDR search, balance history, recording playback, API key management.
**Addresses:** Client self-service portal, API key management

### Phase 6: Analytics + Quality (v1.x)
**Rationale:** Needs accumulated CDR data to be meaningful. Lower priority than core launch features.
**Delivers:** Wastage analysis dashboard, ASR/ACD metrics per gateway/route, DID management, WebSocket real-time push.
**Addresses:** Differentiator features (wastage analysis, ASR/ACD, DID management, WebSocket push, rate plan versioning)

### Phase 7: HA + Scale (v1.x)
**Rationale:** Single FS handles hundreds of concurrent calls. HA is critical before scaling but not for initial launch.
**Delivers:** FreeSWITCH dual hot-standby, ESL connection pooling, health-aware failover.
**Addresses:** FreeSWITCH HA, production hardening

### Phase Ordering Rationale

- **Mock-first is the key insight.** Phases 1-2 split along the mock/real boundary. The state machine, billing atomicity, and slot management are battle-tested before any FreeSWITCH complexity enters. Every critical pitfall in the billing/concurrency domain is addressed in Phase 1.
- **Recording and webhooks are async producers.** They subscribe to PubSub events and operate independently of the call flow. Grouping them in Phase 3 is natural -- both trigger on call completion events.
- **Frontend follows backend.** Phases 4-5 cannot start until the API surface is stable. Admin before Client because operators need the platform to manage clients before clients self-serve.
- **Analytics requires data.** Phase 6 needs CDR volume to be meaningful. Launching analytics with 10 test CDRs is pointless.
- **HA is insurance, not launch-blocking.** Single FreeSWITCH handles thousands of concurrent calls. HA matters for uptime SLAs, not for functionality.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (Real FS Integration):** eslgo library internals, ESL event format parsing, FreeSWITCH originate string syntax, TCP keepalive configuration. eslgo has limited documentation -- plan for source code reading.
- **Phase 3 (Recording):** ffmpeg A/B track merge with time offset alignment. FreeSWITCH uuid_record behavior edge cases. S3 multipart upload for large recordings.
- **Phase 7 (HA):** FreeSWITCH clustering patterns, session recovery limitations, ESL connection pool management across multiple FS nodes.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Foundation):** Encore.go services, sqlc integration, PostgreSQL row locking, Redis Lua scripts -- all well-documented with official examples.
- **Phase 4-5 (Frontend):** React + Vite + shadcn/ui + TanStack -- extremely well-documented ecosystem. Encore gen client integration documented in official docs.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All technologies validated against official docs and current releases. Version compatibility confirmed. eslgo is the only risk (small community) but shallow wrapper makes replacement trivial. |
| Features | HIGH | Domain well-established. Feature set validated against Twilio, ASTPP, Kolmisoft. Clear table stakes vs differentiators vs anti-features. |
| Architecture | HIGH | ESL inbound + bgapi + per-call state machine is canonical FS pattern. Encore service boundaries follow framework conventions. |
| Pitfalls | HIGH | FreeSWITCH-specific pitfalls corroborated by FS GitHub issues (specific issue numbers cited). Billing atomicity patterns are standard. |

**Overall confidence:** HIGH

### Gaps to Address

- **eslgo library depth:** Documentation is sparse. Plan to read eslgo source code during Phase 2 planning. The 5-method FSClient interface limits exposure, but understanding eslgo's reconnection behavior and event parsing internals is needed.
- **FreeSWITCH Docker image customization:** The official `signalwire/freeswitch:1.10.12` image may need custom ESL config, recording modules, and codec packages. Dockerfile creation needs validation during Phase 2.
- **ffmpeg alignment precision:** The `-itsoffset` approach for A/B track merge alignment is theoretically sound but needs empirical testing with real FS recordings to determine acceptable tolerance. Address during Phase 3.
- **Encore + Turborepo monorepo integration:** Backend (Go, Encore) and frontend (Node, Turborepo) coexist in one repo. Build orchestration needs validation -- no official Encore documentation covers this specific setup.
- **Redis failure mode:** Research specifies fail-closed (reject calls if Redis down). Correct for production but needs a fallback strategy during development. Consider PG advisory locks as development-only fallback.

## Sources

### Primary (HIGH confidence)
- [Encore.go official documentation](https://encore.dev/docs/go) -- service patterns, sqldb, PubSub, auth, gen client
- [FreeSWITCH ESL documentation](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Client-and-Developer-Interfaces/Event-Socket-Library/) -- event socket protocol, originate, park, bridge
- [percipia/eslgo v1.5.0](https://github.com/percipia/eslgo) -- Go ESL library, Dec 2025 release
- [sqlc v1.30.0 documentation](https://docs.sqlc.dev/) -- configuration, code generation, PostgreSQL support
- [FreeSWITCH issue tracker](https://github.com/signalwire/freeswitch/issues) -- issues #1688, #2143, #1052 cited in pitfalls
- [shadcn/ui CLI v4](https://ui.shadcn.com/docs/changelog) -- March 2026, unified radix-ui
- [Tailwind CSS v4](https://tailwindcss.com/blog/tailwindcss-v4) -- CSS-first config, Lightning CSS engine

### Secondary (MEDIUM confidence)
- [Twilio Voice API](https://www.twilio.com/docs/voice/api) -- feature landscape comparison
- [ASTPP/Kolmisoft](https://wiki.kolmisoft.com/) -- wholesale VoIP feature expectations
- [Webhook retry best practices (Svix, Hookdeck)](https://www.svix.com/resources/webhook-best-practices/retries/) -- exponential backoff, circuit breaker patterns
- [Redis counting semaphore patterns](https://redis.io/commands/incr) -- atomic operations, Lua scripting

### Tertiary (LOW confidence)
- MagicUI long-term maintenance -- 19K stars but no guarantee of sustained development. Use selectively for visual polish only.
- eslgo community size (130 stars, 53 forks) -- mitigation is shallow wrapper pattern making replacement < 100 LOC.

---
*Research completed: 2026-03-09*
*Ready for roadmap: yes*
