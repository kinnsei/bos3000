---
phase: 3
slug: freeswitch-webhook
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-10
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + encore test |
| **Config file** | none — Encore uses standard Go test conventions |
| **Quick run command** | `encore test ./callback/fsclient/... ./webhook/... ./recording/... -v` |
| **Full suite command** | `encore test ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `encore test ./callback/fsclient/... ./webhook/... ./recording/... -v`
- **After every plan wave:** Run `encore test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | FS-01 | unit | `encore test ./callback/fsclient/... -run TestESLFSClientInterface` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | FS-02 | unit | `encore test ./callback/fsclient/... -run TestEventDispatch` | ❌ W0 | ⬜ pending |
| 03-01-03 | 01 | 1 | FS-03 | unit | `encore test ./callback/fsclient/... -run TestHealthProbe` | ❌ W0 | ⬜ pending |
| 03-01-04 | 01 | 1 | FS-04 | unit | `encore test ./callback/fsclient/... -run TestManagerFailover` | ❌ W0 | ⬜ pending |
| 03-01-05 | 01 | 1 | FS-05 | integration | `encore test ./callback/fsclient/... -run TestFailoverTiming` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 2 | REC-01 | unit | `encore test ./callback/... -run TestRecordingStartOnBridge` | ❌ W0 | ⬜ pending |
| 03-02-02 | 02 | 2 | REC-02 | unit | `encore test ./recording/... -run TestFFmpegMerge` | ❌ W0 | ⬜ pending |
| 03-02-03 | 02 | 2 | REC-03 | integration | `encore test ./recording/... -run TestUploadAndCleanup` | ❌ W0 | ⬜ pending |
| 03-02-04 | 02 | 2 | REC-04 | unit | `encore test ./recording/... -run TestPresignedURL` | ❌ W0 | ⬜ pending |
| 03-03-01 | 03 | 2 | HOOK-01 | unit | `encore test ./webhook/... -run TestDeliveryCreation` | ❌ W0 | ⬜ pending |
| 03-03-02 | 03 | 2 | HOOK-02 | unit | `encore test ./webhook/... -run TestRetryAndDLQ` | ❌ W0 | ⬜ pending |
| 03-03-03 | 03 | 2 | HOOK-03 | unit | `encore test ./webhook/... -run TestAdminDLQ` | ❌ W0 | ⬜ pending |
| 03-03-04 | 03 | 2 | HOOK-04 | unit | `encore test ./webhook/... -run TestClientWebhookConfig` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `callback/fsclient/esl_test.go` — stubs for FS-01, FS-02 (mock ESL conn for unit tests)
- [ ] `callback/fsclient/manager_test.go` — stubs for FS-03, FS-04, FS-05
- [ ] `recording/merge_test.go` — stubs for REC-02 (requires ffmpeg in test env)
- [ ] `recording/recording_test.go` — stubs for REC-03, REC-04
- [ ] `webhook/webhook_test.go` — stubs for HOOK-01, HOOK-02, HOOK-03, HOOK-04
- [ ] `webhook/migrations/1_create_webhook_deliveries.up.sql` — webhook delivery table
- [ ] `callback/migrations/3_add_recording_columns.up.sql` — recording columns on callback_calls
- [ ] `auth/migrations/3_add_webhook_columns.up.sql` — webhook_url/secret on users
- [ ] `docker-compose.dev.yml` — FreeSWITCH container with ESL enabled
- [ ] `docker/freeswitch/conf/` — Minimal FS config with event_socket enabled
- [ ] ffmpeg available in test/dev environment

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real FS ESL connection over network | FS-01 | Requires live FreeSWITCH instance | Start FS container, verify ESLFSClient connects to port 8021 |
| Failover timing < 5s | FS-05 | Requires simulating FS crash in Docker | Stop primary FS container, time until new call routes to standby |
| Real ffmpeg merge output quality | REC-02 | Audio quality is subjective | Listen to merged MP3, verify stereo separation |
| Webhook delivery to external URL | HOOK-01 | Requires external HTTP endpoint | Use webhook.site or ngrok to receive and inspect payload |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
