---
phase: 1
slug: platform-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-10
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + encore test |
| **Config file** | none — Encore uses standard Go test conventions |
| **Quick run command** | `encore test ./auth/... ./billing/... ./routing/... ./compliance/... ./pkg/...` |
| **Full suite command** | `encore test ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `encore test ./auth/... ./billing/... ./routing/... ./compliance/... ./pkg/...`
- **After every plan wave:** Run `encore test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | AUTH-01 | unit | `encore test ./auth/... -run TestAdminCookieAuth` | ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | AUTH-02 | unit | `encore test ./auth/... -run TestClientAuth` | ❌ W0 | ⬜ pending |
| 01-01-03 | 01 | 1 | AUTH-03 | unit | `encore test ./auth/... -run TestAPIKey` | ❌ W0 | ⬜ pending |
| 01-02-01 | 02 | 1 | BILL-01 | unit | `encore test ./billing/... -run TestPreDeduct` | ❌ W0 | ⬜ pending |
| 01-02-02 | 02 | 1 | BILL-02 | unit | `encore test ./billing/... -run TestConcurrentSlot` | ❌ W0 | ⬜ pending |
| 01-02-03 | 02 | 1 | BILL-03 | unit | `encore test ./billing/... -run TestFinalize` | ❌ W0 | ⬜ pending |
| 01-02-04 | 02 | 1 | BILL-04 | unit | `encore test ./billing/... -run TestRatePriority` | ❌ W0 | ⬜ pending |
| 01-02-05 | 02 | 1 | BILL-05 | unit | `encore test ./billing/... -run TestRatePlan` | ❌ W0 | ⬜ pending |
| 01-03-01 | 03 | 1 | ROUT-01 | unit | `encore test ./routing/... -run TestALegRoundRobin` | ❌ W0 | ⬜ pending |
| 01-03-02 | 03 | 1 | ROUT-02 | unit | `encore test ./routing/... -run TestBLegPrefix` | ❌ W0 | ⬜ pending |
| 01-03-03 | 03 | 1 | ROUT-03 | unit | `encore test ./routing/... -run TestGatewayFailover` | ❌ W0 | ⬜ pending |
| 01-03-04 | 03 | 1 | ROUT-04 | unit | `encore test ./routing/... -run TestDIDSelection` | ❌ W0 | ⬜ pending |
| 01-04-01 | 01 | 1 | COMP-01 | unit | `encore test ./compliance/... -run TestBlacklist` | ❌ W0 | ⬜ pending |
| 01-04-02 | 01 | 1 | COMP-02 | unit | `encore test ./compliance/... -run TestAuditLog` | ❌ W0 | ⬜ pending |
| 01-04-03 | 01 | 1 | COMP-03 | unit | `encore test ./compliance/... -run TestDailyLimit` | ❌ W0 | ⬜ pending |
| 01-05-01 | 01 | 1 | INFR-03 | unit | `encore test ./routing/... -run TestPrefixConsistency` | ❌ W0 | ⬜ pending |
| 01-05-02 | 01 | 1 | INFR-04 | unit | `encore test ./pkg/errcode/... -run TestErrorCodes` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `auth/auth_test.go` — stubs for AUTH-01, AUTH-02, AUTH-03
- [ ] `billing/billing_test.go` — stubs for BILL-01 through BILL-05
- [ ] `routing/routing_test.go` — stubs for ROUT-01 through ROUT-04
- [ ] `compliance/compliance_test.go` — stubs for COMP-01, COMP-02, COMP-03
- [ ] `pkg/errcode/codes_test.go` — stubs for INFR-04
- [ ] Database migrations for all services — required before tests can run

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| docker-compose.dev.yml 一键启动 | INFR-01, INFR-02 | Requires Docker runtime | Run `docker-compose -f docker-compose.dev.yml up -d` and verify services start |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
