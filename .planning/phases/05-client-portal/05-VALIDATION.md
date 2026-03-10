---
phase: 5
slug: client-portal
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-10
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Frontend)** | Vitest 3.x (Vite-native test runner) |
| **Framework (Backend)** | encore test (Go test with Encore extensions) |
| **Config file (Frontend)** | portal/vitest.config.ts (Wave 0 creates) |
| **Config file (Backend)** | None — Encore test built-in |
| **Quick run command (Frontend)** | `cd portal && npx vitest run --reporter=verbose` |
| **Full suite command (Frontend)** | `cd portal && npx vitest run --coverage` |
| **Quick run command (Backend)** | `encore test ./callback/... -run TestWebSocket -count=1` |
| **Full suite command** | `cd portal && npx vitest run --coverage && encore test ./callback/... -count=1` |
| **Estimated runtime** | ~15 seconds (frontend) + ~5 seconds (backend) |

---

## Sampling Rate

- **After every task commit:** Run `cd portal && npx vitest run --reporter=verbose`
- **After every plan wave:** Run `cd portal && npx vitest run --coverage && encore test ./callback/... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 20 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 1 | UI-04 | unit | `encore test ./callback/... -run TestHub -count=1` | Wave 0 | pending |
| 05-02-01 | 02 | 1 | CLNT-01 | unit | `cd portal && npx vitest run src/features/dashboard/ -x` | Wave 0 | pending |
| 05-02-02 | 02 | 1 | CLNT-02 | unit | `cd portal && npx vitest run src/features/callback/ -x` | Wave 0 | pending |
| 05-03-01 | 03 | 2 | CLNT-03 | unit | `cd portal && npx vitest run src/features/cdr/ -x` | Wave 0 | pending |
| 05-03-02 | 03 | 2 | CLNT-05/WAST-04 | unit | `cd portal && npx vitest run src/features/wastage/ -x` | Wave 0 | pending |
| 05-04-01 | 04 | 2 | CLNT-04 | unit | `cd portal && npx vitest run src/features/finance/ -x` | Wave 0 | pending |
| 05-04-02 | 04 | 2 | CLNT-06 | unit | `cd portal && npx vitest run src/features/api-integration/ -x` | Wave 0 | pending |
| 05-04-03 | 04 | 2 | CLNT-07 | unit | `cd portal && npx vitest run src/features/settings/ -x` | Wave 0 | pending |
| 05-05-01 | 05 | 3 | UI-04 | unit | `cd portal && npx vitest run src/hooks/ -x` | Wave 0 | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [ ] `portal/vitest.config.ts` — Vitest configuration with jsdom environment
- [ ] `portal/src/test/setup.ts` — Test setup (jsdom, mock matchMedia, mock ResizeObserver)
- [ ] `portal/src/test/mocks/api.ts` — Mock Encore client for unit tests
- [ ] `portal/src/test/mocks/websocket.ts` — Mock WebSocket for WS hook tests
- [ ] Install: `npm install -D vitest @testing-library/react @testing-library/jest-dom jsdom @testing-library/user-event`
- [ ] `callback/ws_hub_test.go` — Hub unit tests (register, unregister, broadcast routing)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| WebSocket reconnect banner shows/hides | UI-04 | Requires real WS disconnect simulation | 1. Open portal, 2. Kill backend, 3. Verify banner appears, 4. Restart backend, 5. Verify banner disappears |
| Recording audio playback in CDR detail | CLNT-03 | Requires actual audio file and browser | 1. Open CDR detail sheet, 2. Click play on recording, 3. Verify audio plays |
| Excel/CSV export downloads correctly | CLNT-03 | Requires browser download behavior | 1. Filter CDR list, 2. Click Export Excel, 3. Open file and verify data |
| Theme switching (dark/light) | Visual | Requires visual verification | 1. Toggle theme, 2. Verify all pages render correctly in both modes |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 20s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
