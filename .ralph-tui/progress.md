# Ralph Progress Log

This file tracks progress across iterations. Agents update this file
after each iteration and it's included in prompts for context.

## Codebase Patterns (Study These First)

*Add reusable patterns discovered during development here.*

- **Encore errs.ErrDetails**: Must implement marker interface with `ErrDetails()` method — a plain struct won't work as the `Details` field type.
- **No errs.As**: Use standard `errors.As` from the `errors` package instead.

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
