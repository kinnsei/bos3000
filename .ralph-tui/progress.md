# Ralph Progress Log

This file tracks progress across iterations. Agents update this file
after each iteration and it's included in prompts for context.

## Codebase Patterns (Study These First)

*Add reusable patterns discovered during development here.*

- **Encore errs.ErrDetails**: Must implement marker interface with `ErrDetails()` method — a plain struct won't work as the `Details` field type.
- **No errs.As**: Use standard `errors.As` from the `errors` package instead.
- **Encore auth handler**: Use `AuthParams` struct with `cookie:"session"`, `header:"Authorization"`, `query:"api_key"` tags for multi-method auth dispatch.
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
