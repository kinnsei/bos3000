---
phase: 2
slug: mock
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-10
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + encore test |
| **Config file** | none — Encore uses standard Go test conventions |
| **Quick run command** | `encore test ./callback/... -v` |
| **Full suite command** | `encore test ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `encore test ./callback/... -v`
- **After every plan wave:** Run `encore test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | CALL-01 | integration | `encore test ./callback/... -run TestFullCallbackFlow` | ❌ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | CALL-02 | unit | `encore test ./callback/... -run TestParkTimeout` | ❌ W0 | ⬜ pending |
| 02-01-03 | 01 | 1 | CALL-03 | unit | `encore test ./callback/... -run TestBLegOriginateFailure` | ❌ W0 | ⬜ pending |
| 02-01-04 | 01 | 1 | CALL-04 | integration | `encore test ./callback/... -run TestStateTransitions` | ❌ W0 | ⬜ pending |
| 02-01-05 | 01 | 1 | CALL-05 | unit | `encore test ./callback/... -run TestGetCallStatus` | ❌ W0 | ⬜ pending |
| 02-01-06 | 01 | 1 | CALL-06 | unit | `encore test ./callback/... -run TestForceHangup` | ❌ W0 | ⬜ pending |
| 02-01-07 | 01 | 1 | CALL-07 | integration | `encore test ./callback/... -run TestMock` | ❌ W0 | ⬜ pending |
| 02-01-08 | 01 | 1 | WAST-01 | unit | `encore test ./callback/... -run TestWastageClassification` | ❌ W0 | ⬜ pending |
| 02-01-09 | 01 | 1 | WAST-02 | unit | `encore test ./callback/... -run TestWastageCost` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `callback/callback_test.go` — test stubs for CALL-01 through CALL-07, WAST-01, WAST-02
- [ ] `callback/fsclient/mock.go` — MockFSClient implementation
- [ ] `callback/fsclient/mock_test.go` — Mock behavior verification
- [ ] `callback/migrations/1_create_callback_calls.up.sql` — table schema
- [ ] `callback/migrations/2_create_system_configs.up.sql` — bridge threshold config

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
