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

Task IDs follow format: {phase}-{plan}-{task} mapped to actual plan structure.

| Task ID | Plan | Task | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|------|-------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 (INFR) | T1 | 1 | INFR-01, INFR-02, INFR-04 | unit | `encore test ./pkg/... -run TestNewError\|TestAllConstants\|TestSuggestions` | No W0 | pending |
| 01-01-02 | 01 (INFR) | T2 | 1 | INFR-01, INFR-02 | config | `docker compose -f docker-compose.dev.yml config --quiet` | No W0 | pending |
| 01-02-01 | 02 (AUTH) | T1 | 2 | AUTH-01, AUTH-02 | unit | `encore test ./auth/... -run "TestAdmin\|TestClient\|TestAuthHandler\|TestInvalid"` | No W0 | pending |
| 01-02-02 | 02 (AUTH) | T2 | 2 | AUTH-03 | unit | `encore test ./auth/... -run "TestAPIKey\|TestIPWhitelist"` | No W0 | pending |
| 01-03-01 | 03 (BILL) | T1 | 2 | BILL-01, BILL-02 | unit | `encore test ./billing/... -run "TestPreDeduct\|TestAcquireSlot\|TestReleaseSlot"` | No W0 | pending |
| 01-03-02 | 03 (BILL) | T2 | 2 | BILL-03, BILL-04, BILL-05 | unit | `encore test ./billing/... -run "TestFinalize\|TestCreate\|TestResolve\|TestAdmin"` | No W0 | pending |
| 01-03-03 | 03 (BILL) | T3 | 2 | BILL-01 | unit | `encore test ./billing/... -run "TestTopup\|TestDeduct\|TestGetAccount\|TestListTransactions\|TestCreateAccount"` | No W0 | pending |
| 01-04-01 | 04 (ROUT) | T1 | 3 | ROUT-01 | unit | `encore test ./routing/... -run "TestPickALeg"` | No W0 | pending |
| 01-04-02 | 04 (ROUT) | T2 | 3 | ROUT-02, ROUT-03, INFR-03 | unit | `encore test ./routing/... -run "TestPickBLeg\|TestPrefix"` | No W0 | pending |
| 01-04-03 | 04 (ROUT) | T3 | 3 | ROUT-04 | unit | `encore test ./routing/... -run "TestSelectDID\|TestImportDID\|TestAssign"` | No W0 | pending |
| 01-04-04 | 04 (ROUT) | T4 | 3 | ROUT-01 | unit | `encore test ./routing/... -run "TestHealthCheck"` | No W0 | pending |
| 01-05-01 | 05 (COMP) | T1 | 2 | COMP-01, COMP-03 | unit | `encore test ./compliance/... -run "TestBlacklist\|TestDailyLimit\|TestAdd"` | No W0 | pending |
| 01-05-02 | 05 (COMP) | T2 | 2 | COMP-02 | unit | `encore test ./compliance/... -run "TestAudit\|TestQuery"` | No W0 | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [ ] `auth/auth_test.go` — stubs for AUTH-01, AUTH-02, AUTH-03
- [ ] `billing/billing_test.go` — stubs for BILL-01 through BILL-05
- [ ] `routing/routing_test.go` — stubs for ROUT-01 through ROUT-04, INFR-03
- [ ] `compliance/compliance_test.go` — stubs for COMP-01, COMP-02, COMP-03
- [ ] `pkg/errcode/codes_test.go` — stubs for INFR-04
- [ ] Database migrations for all services — required before tests can run

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| docker-compose.dev.yml starts FreeSWITCH | INFR-01, INFR-02 | Requires Docker runtime | Run `docker-compose -f docker-compose.dev.yml up -d` and verify services start |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
