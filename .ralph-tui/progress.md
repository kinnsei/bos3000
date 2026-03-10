# Ralph Progress Log

This file tracks progress across iterations. Agents update this file
after each iteration and it's included in prompts for context.

## Codebase Patterns (Study These First)

*Add reusable patterns discovered during development here.*

- **Encore errs.ErrDetails**: Must implement marker interface with `ErrDetails()` method — a plain struct won't work as the `Details` field type.
- **No errs.As**: Use standard `errors.As` from the `errors` package instead.
- **Encore auth handler**: Use `AuthParams` struct with `cookie:"session"`, `header:"Authorization"`, `query:"api_key"` tags for multi-method auth dispatch.
- **Encore test API calls**: Use package-level generated functions (e.g., `CreateAPIKey(ctx)`) not `svc.Method(ctx)` for `encore:api` endpoints in tests. Direct method calls bypass Encore's request pipeline so `auth.Data()` returns nil. Use `auth.WithContext()` to set up auth context for tests.
- **Encore path params**: Path parameters (`:id`, `:prefix`) must be separate function parameters, not struct fields. For endpoints with path params + body: `func (s *Service) Foo(ctx context.Context, id int64, p *Params)`. Use `authpkg.Data()` (import alias) to check roles cross-service.
- **No `sqldb.ErrNoRows.Is()`**: Use `errors.Is(err, sqldb.ErrNoRows)` — the sentinel error has no `.Is()` method.
- **Encore test helpers**: Import `"encore.dev/et"` (NOT `"encore.dev/test"`). Use `et.Topic(topic).PublishedMessages()` for Pub/Sub test assertions.
- **Admin login as raw endpoint**: Encore structured endpoints can't set cookies. Use `//encore:api public raw` for admin login to set `Set-Cookie` header directly.
- **`sqldb.Named()` must be package-level**: `sqldb.Named("dbname")` cannot be called inside functions — must be assigned to a package-level `var` (Encore error E1183).
- **Test unique emails**: Use `time.Now().UnixNano()` in email addresses to avoid unique constraint violations across test runs.
- **Encore path conflicts**: Static paths like `/dids/import` conflict with parameterized `/dids/:id/...` under the same prefix. Use a different path structure (e.g., `/did-import`) to avoid E1094.

---

## 2026-03-10 - bd-3it.1.1
- Created `pkg/errcode/codes.go`: 10 business error constants, `ErrDetails` struct (implements `errs.ErrDetails` marker), `NewError` constructor with `suggestFor` helper
- Created `pkg/errcode/codes_test.go`: TestNewError, TestAllConstantsFormat (SCREAMING_SNAKE regex), TestSuggestions
- Created `pkg/types/types.go`: `Money int64`, `PhoneNumber string`, `CeilDiv` helper
- Files changed: `pkg/errcode/codes.go`, `pkg/errcode/codes_test.go`, `pkg/types/types.go`
- **Learnings:**
  - `errs.Error.Details` field is typed `errs.ErrDetails` (an interface), not `any` — custom detail structs need the `ErrDetails()` marker method
  - `errs.As` doesn't exist; use `errors.As` from stdlib
---

## 2026-03-10 - bd-3it.1.2
- Created `docker-compose.dev.yml` with FreeSWITCH service (`drachtio/drachtio-freeswitch-mrf:latest`)
- SIP port 5060/udp, ESL port 8021/tcp, ESL_PASSWORD=ClueCon, healthcheck on port 8021
- Documented `encore run --listen :12345` in comments (INFR-01)
- Optional volume mount for custom config commented out
- Files changed: `docker-compose.dev.yml`
- **Learnings:**
  - `docker compose config --quiet` validates compose files without output on success
  - `drachtio/drachtio-freeswitch-mrf` is the preferred FreeSWITCH image for media relay use cases
---

## 2026-03-10 - bd-3it.1.3
- Created auth service foundation with dual login (admin cookie + client JWT)
- Files changed: `auth/auth.go`, `auth/handler.go`, `auth/login.go`, `auth/auth_test.go`, `auth/migrations/1_create_users.up.sql`, `auth/migrations/2_create_api_keys.up.sql`
- `auth/auth.go`: Service struct with jwtSecret, AuthData (UserID/Role/Username), AuthParams with cookie/header/query tags, DB init
- `auth/handler.go`: AuthHandler dispatches cookie→bearer→apikey, JWT parsing with HS256, IP whitelist check for API keys, SHA-256 key hash lookup
- `auth/login.go`: AdminLogin (raw endpoint, Set-Cookie HttpOnly/Secure/SameSite=Lax), ClientLogin (structured, returns token+expires_at), bcrypt password verification, 24h JWT expiry
- `auth/auth_test.go`: TestAdminCookieAuth, TestClientAuth, TestAuthHandlerDispatch (4 subtests), TestInvalidCredentials — all passing
- **Learnings:**
  - Encore `Request.Headers` provides HTTP headers including `X-Forwarded-For` for IP detection
  - `testing.AllocsPerRun(0, func(){})` panics with divide by zero — don't use for unique ID generation
  - Admin login must be raw endpoint (`//encore:api public raw`) to set cookies; client login uses structured endpoint
  - Also included `2_create_api_keys.up.sql` migration since auth handler queries `api_keys` table for API key validation
---

## 2026-03-10 - bd-3it.1.4
- Created `auth/apikey.go` with 7 API endpoints: CreateAPIKey, ListAPIKeys, ResetAPIKey, RevokeAPIKey, AddIPWhitelist, RemoveIPWhitelist, ListIPWhitelist
- All endpoints enforce ownership via `verifyKeyOwnership()` with admin role override
- Keys use crypto/rand (32 bytes) + base64url encoding with "bos_" prefix, stored as SHA-256 hash
- Added 5 tests: TestAPIKeyCreate, TestAPIKeyListNeverExposesHash, TestAPIKeyReset, TestAPIKeyRevoked, TestIPWhitelist
- Files changed: `auth/apikey.go` (new), `auth/auth_test.go` (appended tests)
- **Learnings:**
  - Must use Encore generated package-level functions (not `svc.Method()`) in tests for `encore:api auth` endpoints — direct method calls bypass request pipeline causing `auth.Data()` to return nil
  - `auth.WithContext(ctx, uid, data)` properly sets up auth for generated function calls in tests
  - `validateIPOrCIDR` helper validates both single IPs and CIDR notation using `net.ParseIP` and `net.ParseCIDR`
---

## 2026-03-10 - bd-3it.1.5
- Created billing service with pre-deduction, concurrent slot management, and call finalization
- Files changed: `billing/billing.go` (new), `billing/balance.go` (new), `billing/billing_test.go` (new), `billing/migrations/1_create_accounts.up.sql` (new), `billing/migrations/2_create_transactions.up.sql` (new)
- `billing/billing.go`: Service struct, DB (`sqldb.NewDatabase`), Redis cache cluster (`cache.NewCluster`), IntKeyspace for concurrent slots (24h expiry)
- `billing/balance.go`: 4 private endpoints — PreDeduct (row-level locking with `FOR UPDATE`, 30min pre-deduction), AcquireSlot (Redis INCR with rollback on limit exceeded), ReleaseSlot (Redis DECR floored at 0), Finalize (A-leg 6s blocks, B-leg 60s blocks, refund/charge diff)
- `billing/billing_test.go`: 6 tests — all passing: PreDeductSufficientBalance, PreDeductInsufficientBalance, PreDeductConcurrentSerializedByRowLock, AcquireSlotUnderLimit, AcquireSlotExceedsLimit, ReleaseSlot
- **Learnings:**
  - Encore `cache.IntKeyspace.Set()` returns only `error` (1 value), not `(int64, error)` — unlike `Increment` which returns `(int64, error)`
  - Encore migrations must be sequentially numbered (1, 2, 3...) — gaps cause errors. Renumbered from spec's 1,3 to 1,2
  - `cache.NewIntKeyspace[int64]` uses the key type as generic param, not the value type
---

## 2026-03-10 - bd-3it.1.6
- Created rate plan CRUD (CreateRatePlan, UpdateRatePlan, ListRatePlans, GetRatePlan), prefix rate management (AddPrefixRate, RemovePrefixRate), multi-tier rate resolution (ResolveRate), and user rate config (SetUserRateConfig)
- Created migration `3_create_rate_plans.up.sql` with `rate_plans` and `rate_plan_prefixes` tables
- Added finalize tests (refund, zero-duration, block rounding) and rate plan tests (uniform, prefix, user priority, no-rate-found, admin-only access)
- Files changed: `billing/rates.go` (new), `billing/migrations/3_create_rate_plans.up.sql` (new), `billing/billing_test.go` (appended)
- **Learnings:**
  - Encore path params (`:id`, `:prefix`) must be separate function parameters, NOT struct fields with `path` tags. For path+body: `func Foo(ctx, id int64, p *Body)`
  - `sqldb.ErrNoRows` has no `.Is()` method — must use `errors.Is(err, sqldb.ErrNoRows)`
  - Cross-service auth checking: import auth package with alias (`authpkg "encore.app/auth"`) and use `authpkg.Data()` to access `AuthData` struct
  - `scanRatePlan` helper with `interface{ Scan(...any) error }` works for both `QueryRow` and `Rows.Next()` scanning
---

## 2026-03-10 - bd-3it.1.7
- Added 5 admin billing endpoints: Topup, Deduct, GetAccount, ListTransactions, CreateAccount
- Added migration `4_add_account_status.up.sql` for status column (active/suspended/closed)
- Topup/Deduct: admin-only, row-level locking, transaction recording
- GetAccount: ownership-enforced (admin: any user, client: own only), returns balance/credit_limit/max_concurrent/rate_plan_id/status
- ListTransactions: paginated with type/date_from/date_to filters, ownership-enforced
- CreateAccount: admin-only, explicit workflow for creating billing accounts with initial config
- Added 5 tests: TestTopup, TestDeduct, TestGetAccountOwnershipEnforced, TestListTransactionsPagination, TestCreateAccount
- Files changed: `billing/balance.go`, `billing/billing_test.go`, `billing/migrations/4_add_account_status.up.sql` (new)
- **Learnings:**
  - Dynamic SQL query building with `strconv.Itoa(argIdx)` for parameterized filters works well with Encore's sqldb
  - Go 1.22+ `max()` builtin preferred over `if` for defaulting page numbers
---

## 2026-03-10 - bd-3it.1.12
- Created compliance service with blacklist CRUD and daily rate limiting
- Files changed: `compliance/compliance.go` (new), `compliance/blacklist.go` (new), `compliance/ratelimit.go` (new), `compliance/migrations/1_create_blacklist.up.sql` (new), `compliance/compliance_test.go` (new)
- `compliance/compliance.go`: Service struct, DB (`sqldb.NewDatabase`), Redis cache cluster (`cache.NewCluster`)
- `compliance/blacklist.go`: 4 endpoints — CheckBlacklist (private, global-first then client-level), AddBlacklist (auth, admin/client permission checks), RemoveBlacklist (auth, ownership enforced), ListBlacklist (auth, admin sees all, client sees own+global)
- `compliance/ratelimit.go`: CheckDailyLimit (private, Redis IntKeyspace with optimistic increment, rollback on limit exceeded, fail-open on cache errors with CurrentCount=-1)
- `compliance/compliance_test.go`: 8 tests all passing — TestBlacklistGlobalHit, TestBlacklistClientHit, TestBlacklistNotBlocked, TestBlacklistGlobalCannotBeOverridden, TestDailyLimitUnderLimit, TestDailyLimitExceeded, TestDailyLimitFailOpen, TestAddBlacklistPermissions (4 subtests)
- **Learnings:**
  - Encore GET endpoints don't support `*int64` in query params — use `int64` with 0 as sentinel instead
  - `UNIQUE NULLS NOT DISTINCT (number, user_id)` works in PG 15+ for treating NULL=NULL in unique constraints
  - `cache.NewIntKeyspace` Increment returns `(int64, error)` — can use negative increment (-1) for decrement
---

## 2026-03-10 - bd-3it.1.13
- Created async audit logging via Pub/Sub with `AuditEvents` topic and `write-audit-log` subscription
- `PublishAuditEvent` (private POST) for other services to record audit events asynchronously
- `QueryAuditLogs` (auth GET, admin-only) with filters (operator_id, action, resource_type, date_from, date_to) and pagination
- `CleanupAuditLogs` cron job at 3am UTC daily, deletes records older than 90 days
- Files changed: `compliance/audit.go` (new), `compliance/migrations/2_create_audit_logs.up.sql` (new), `compliance/compliance_test.go` (appended 3 tests)
- **Learnings:**
  - Encore `et.Topic(topic).PublishedMessages()` works for verifying Pub/Sub publishes in tests — import `"encore.dev/et"` (not `"encore.dev/test"`)
  - `TIMESTAMPTZ` columns must be scanned as `time.Time`, not `string`
  - `db.Exec` returns a result with `RowsAffected()` method (int64) for counting affected rows
---

## 2026-03-10 - bd-3it.1.9
- Created `routing/bleg.go` with PickBLeg (prefix matching + failover chain), AddPrefix, RemovePrefix, ListPrefixes, and GetPrefixMismatches
- PickBLeg: extracts 3-digit prefix, queries gateway_prefixes by priority, picks first healthy gateway, follows failover_gateway_id chain (max 3 hops)
- Returns PREFIX_NOT_FOUND (no prefix match) or NO_HEALTHY_GATEWAY (prefix matched but all down)
- GetPrefixMismatches: exported testable version of consistency check comparing gateway vs billing rate plan prefixes
- Added 4 tests: TestPickBLegPrefixMatch, TestPickBLegFailover, TestPickBLegUnknownPrefix, TestPrefixConsistencyWarning — all passing
- Files changed: `routing/bleg.go` (new), `routing/routing_test.go` (appended)
- **Learnings:**
  - `followFailover` pattern: iterative loop with hop counter prevents infinite loops in failover chains
  - B-leg gateway selection is priority-ordered (not weighted RR like A-leg) — first healthy wins
  - `GetPrefixMismatches` export pattern avoids needing rlog capture in tests — just return the mismatch lists
---

## 2026-03-10 - bd-3it.1.8
- Created routing service with gateway migrations, A-leg weighted round-robin, and gateway admin CRUD
- Files changed: `routing/routing.go` (new), `routing/aleg.go` (new), `routing/routing_test.go` (new), `routing/migrations/1_create_gateways.up.sql` (new)
- `routing/routing.go`: Service struct with sync.Mutex + aLegGateways slice, DB, billingDB (sqldb.Named), initService loads A-leg gateways, validatePrefixConsistency compares gateway vs billing rate plan prefixes
- `routing/aleg.go`: PickALeg (private POST, smooth weighted RR nginx algorithm), CreateGateway, UpdateGateway, ListGateways, ToggleGateway (all auth, admin-only), auto-reloads A-leg pool on changes
- `routing/routing_test.go`: 3 tests — WeightedDistribution (1000 iterations, ±5% tolerance), SkipsUnhealthy, NoHealthy (503 Unavailable)
- **Learnings:**
  - `sqldb.Named("billing")` must be a package-level variable — cannot call inside a function (Encore E1183)
  - Smooth weighted RR: each tick currentWeight += weight, pick max, subtract totalWeight from winner — gives exact weight-proportional distribution
  - Go 1.22+ `for i := range N` preferred over `for i := 0; i < N; i++`
---

## 2026-03-10 - bd-3it.1.11
- Created `routing/health.go` with gateway health check system
- `RunHealthCheck` (private POST, cron every 1min): queries all enabled gateways, TCP dials SIP address (5s timeout), tracks failure count, marks unhealthy after N failures (default 3), resets on success, updates both DB and in-memory A-leg state
- `ManualHealthCheck` (auth POST, admin-only): force health check on single gateway by ID
- `checkTCP` helper: strips `sip:` prefix, defaults port 5060, uses `net.DialTimeout`
- Cron job `gateway-health-check` at 1-minute interval (minimum Encore allows)
- Added 2 tests: TestHealthCheckMarksUnhealthy (3 consecutive failures → unhealthy), TestHealthCheckRecovery (listener → healthy + failures reset)
- Files changed: `routing/health.go` (new), `routing/routing_test.go` (appended)
- **Learnings:**
  - `net.DialTimeout("tcp", addr, timeout)` is sufficient for basic SIP gateway health checks — no need for SIP-level probes
  - Test recovery pattern: use `net.Listen("tcp", "127.0.0.1:0")` to create a temporary listener for TCP reachability tests
  - Unreachable IP `192.0.2.1` (TEST-NET-1 per RFC 5737) reliably times out for failure simulation, but adds ~15s per test due to 5s timeout × 3 iterations
---

## 2026-03-10 - bd-3it.1.10
- Created DID number pool management with 5 endpoints: SelectDID (private), ImportDIDs, AssignDID, UnassignDID, ListDIDs (all auth/admin)
- SelectDID: user pool > public pool (user_id IS NULL) priority with RANDOM() selection
- ImportDIDs: bulk INSERT with ON CONFLICT DO NOTHING, returns imported/skipped counts
- AssignDID/UnassignDID: update user_id to manage pool membership
- ListDIDs: paginated with user_id and status filters, dynamic WHERE clause
- Created migration `2_create_did_numbers.up.sql` with UNIQUE number, status CHECK constraint, indexes on user_id and status
- Added 5 tests: TestSelectDIDUserPool, TestSelectDIDPublicFallback, TestSelectDIDNoneAvailable, TestImportDIDs, TestAssignUnassignDID — all passing
- Files changed: `routing/did.go` (new), `routing/migrations/2_create_did_numbers.up.sql` (new), `routing/routing_test.go` (appended)
- **Learnings:**
  - Encore path conflict: `/routing/dids/import` (static) conflicts with `/routing/dids/:id/assign` (parameterized) — must use different path prefix like `/routing/did-import`
  - `db.Exec` result `.RowsAffected()` returns `int64` directly (not `(int64, error)` like database/sql)
---
