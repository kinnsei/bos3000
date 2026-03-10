# Ralph Progress Log

This file tracks progress across iterations. Agents update this file
after each iteration and it's included in prompts for context.

## Codebase Patterns (Study These First)

*Add reusable patterns discovered during development here.*

- **Encore errs.ErrDetails**: Must implement marker interface with `ErrDetails()` method — a plain struct won't work as the `Details` field type.
- **No errs.As**: Use standard `errors.As` from the `errors` package instead.
- **Encore auth handler**: Use `AuthParams` struct with `cookie:"session"`, `header:"Authorization"`, `query:"api_key"` tags for multi-method auth dispatch.
- **Encore test API calls**: Use package-level generated functions (e.g., `CreateAPIKey(ctx)`) not `svc.Method(ctx)` for `encore:api` endpoints in tests. Direct method calls bypass Encore's request pipeline so `auth.Data()` returns nil. Use `auth.WithContext()` to set up auth context for tests.
- **Admin login as raw endpoint**: Encore structured endpoints can't set cookies. Use `//encore:api public raw` for admin login to set `Set-Cookie` header directly.
- **Test unique emails**: Use `time.Now().UnixNano()` in email addresses to avoid unique constraint violations across test runs.

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
