---
phase: 4
slug: admin-dashboard
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-10
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 3.x (Vite-native test runner, jsdom environment) |
| **Config file** | admin/vitest.config.ts (Wave 0 installs) |
| **Quick run command** | `cd admin && npx vitest run --reporter=verbose` |
| **Full suite command** | `cd admin && npx vitest run --coverage` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd admin && npx vitest run --reporter=verbose`
- **After every plan wave:** Run `cd admin && npx vitest run --coverage`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 0 | UI-01, UI-03 | unit | `cd admin && npx vitest run src/app/ -x` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 1 | ADMN-01 | unit | `cd admin && npx vitest run src/features/dashboard/ -x` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 1 | UI-02 | unit | `cd admin && npx vitest run src/lib/theme/ -x` | ❌ W0 | ⬜ pending |
| 04-03-01 | 03 | 2 | ADMN-02 | unit | `cd admin && npx vitest run src/features/customers/ -x` | ❌ W0 | ⬜ pending |
| 04-03-02 | 03 | 2 | ADMN-03 | unit | `cd admin && npx vitest run src/features/gateways/ -x` | ❌ W0 | ⬜ pending |
| 04-03-03 | 03 | 2 | ADMN-04 | unit | `cd admin && npx vitest run src/features/cdr/ -x` | ❌ W0 | ⬜ pending |
| 04-03-04 | 03 | 2 | ADMN-05, WAST-03 | unit | `cd admin && npx vitest run src/features/wastage/ -x` | ❌ W0 | ⬜ pending |
| 04-03-05 | 03 | 2 | ADMN-06 | unit | `cd admin && npx vitest run src/features/finance/ -x` | ❌ W0 | ⬜ pending |
| 04-03-06 | 03 | 2 | ADMN-07 | unit | `cd admin && npx vitest run src/components/shared/csv-upload -x` | ❌ W0 | ⬜ pending |
| 04-03-07 | 03 | 2 | ADMN-08 | unit | `cd admin && npx vitest run src/features/compliance/ -x` | ❌ W0 | ⬜ pending |
| 04-03-08 | 03 | 2 | ADMN-09 | unit | `cd admin && npx vitest run src/features/ops/ -x` | ❌ W0 | ⬜ pending |
| 04-03-09 | 03 | 2 | ADMN-10 | unit | `cd admin && npx vitest run src/features/settings/ -x` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `admin/vitest.config.ts` — Vitest configuration with jsdom environment
- [ ] `admin/src/test/setup.ts` — Test setup (jsdom, mock matchMedia, mock ResizeObserver)
- [ ] `admin/src/test/mocks/api.ts` — Mock Encore client for unit tests
- [ ] Install: `npm install -D vitest @testing-library/react @testing-library/jest-dom jsdom @testing-library/user-event`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Theme toggle visual correctness | UI-03 | Visual appearance of dark/light themes | Toggle theme, verify colors, contrast, and readability |
| MagicUI animation smoothness | UI-01 | Animation timing subjective | Navigate dashboard, verify NumberTicker, AnimatedList render smoothly |
| Responsive layout at breakpoints | UI-03 | Layout edge cases at exact breakpoints | Resize browser to 1024px, 768px, verify sidebar collapses properly |
| Recharts chart readability | UI-02 | Visual assessment of chart labels/colors | View each chart type, verify legends readable, colors distinguishable |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
