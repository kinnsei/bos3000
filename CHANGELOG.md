# Changelog

All notable changes to BOS3000 are documented here.

## [0.8.0] — 2026-03-12

### Added
- Audit coverage: dashboard trends, profit analysis, wastage analysis, ops monitoring, system config backend APIs
- Rate query endpoint for portal clients
- Webhook config and delivery APIs
- GitHub Actions release workflow (auto-publish on tag push)
- CHANGELOG.md

### Fixed
- CDR field mapping (call_id, a_number, b_number, bridge_duration_ms, total_cost)
- Rate plan CRUD (mode, uniform_a_rate, uniform_b_rate fields)
- API key creation and gateway status endpoints
- Environment config alignment between install.sh and env.example

### Changed
- Removed static deploy/bos3000.service (install.sh generates it dynamically)
- Improved env.example with full documentation for all variables
- build.sh now cleans up .bak files after sed version injection

## [0.7.0] — 2026-03-11

### Added
- Binary tarball + Docker image build pipeline (`scripts/build.sh`)
- Cloud server deployment: install.sh with PostgreSQL, Redis, FreeSWITCH auto-setup
- Version display in admin/portal header bars
- Frontend test suite: 130+ vitest tests across admin and portal

### Fixed
- ESL event subscription (eslCommand vs rawCommand types)
- SIP call flow: uuid_bridge for B-leg, proper state machine transitions
- Vite proxy and API client route alignment
- Mock pages wired to real APIs

### Changed
- Removed standby FreeSWITCH and Docker-compose deployment
- Simplified to single FreeSWITCH instance architecture

## [0.6.0] — 2026-03-10

### Added
- Portal frontend (Vite + React): callback form, CDR, finance, settings, API integration
- WebSocket hub for real-time call status updates
- Admin dashboard: customers, gateways, CDR, finance, compliance, DID management

## [0.5.0] — 2026-03-09

### Added
- FreeSWITCH ESL integration and callback state machine
- Recording service with merge support
- Webhook delivery with retry and DLQ

## [0.4.0] — 2026-03-08

### Added
- Admin dashboard scaffold (Vite + React + Tailwind + shadcn/ui)
- App shell, router, auth flow, API client layer
- Dashboard overview, wastage analysis, feature pages
- Visual polish pass

## [0.3.0] — 2026-03-07

### Added
- Callback engine with mock FSClient
- Call state machine (initiating → a_ringing → a_answered → b_ringing → bridged → completed)

## [0.2.0] — 2026-03-06

### Added
- Gateway management: migrations, weighted round-robin, health check cron
- DID number pool and caller ID selection
- B-leg prefix matching with failover
- Compliance: blacklist CRUD, daily rate limiting, async audit logging via Pub/Sub

## [0.1.0] — 2026-03-05

### Added
- Initial project scaffold with Encore.go
- Auth service: admin/client login, JWT sessions, API key management
- Billing service: balance, topup, deduction, rate plan CRUD
- Routing service foundation
