# Pitfalls Research

**Domain:** API Callback/Click-to-Call System (FreeSWITCH ESL + Encore.go)
**Researched:** 2026-03-09
**Confidence:** HIGH (domain-specific, corroborated by FreeSWITCH issue tracker and production patterns)

## Critical Pitfalls

### Pitfall 1: Parked A-Leg Channel Leak on ESL Disconnect

**What goes wrong:**
When A-leg answers and is parked (waiting for B-leg originate), if the ESL connection drops or the Go process crashes, FreeSWITCH has no built-in mechanism to clean up the parked channel. The call remains allocated indefinitely, consuming FS resources (memory, file descriptors, session slots). Accumulated leaked channels eventually prevent FreeSWITCH from accepting new calls. This is a confirmed FS architectural issue ([signalwire/freeswitch#1688](https://github.com/signalwire/freeswitch/issues/1688)).

**Why it happens:**
Per SIP specification, the timeout responsibility for an unanswered/parked session is on the client side. When the ESL client (our Go app) disappears, no one sends the hangup. FS `park_timeout` only fires if it was set before parking -- if the app crashes before setting it, or if the channel variable was not applied, the channel lives forever.

**How to avoid:**
1. Always set `park_timeout=60` as a channel variable in the originate string itself, not as a post-park ESL command. This way FS enforces the timeout even if the ESL client vanishes.
2. Implement a FS-side watchdog: a periodic `show channels` or `show calls` ESL command that detects channels parked longer than 90s and issues `uuid_kill`.
3. On Go app startup, run a reconciliation sweep: query FS for any existing channels and hangup orphans.

**Warning signs:**
- `show channels count` on FS CLI trending upward over time
- FS memory usage growing without corresponding call volume
- "Maximum sessions reached" errors in FS logs

**Phase to address:**
Phase 1b (Real FS Integration) -- the park timeout must be in the originate string from day one. The watchdog sweep should be implemented in the same phase.

---

### Pitfall 2: ESL Event Queue Saturation on Network Partition

**What goes wrong:**
When the Go app loses network to FreeSWITCH but the TCP connection is not yet detected as dead (TCP keepalive timeout can be minutes), FS continues queuing events for the dead listener. The event queue fills to its maximum (default 100,000 events), FS starts logging "Event enqueue ERROR" and "Lost [N] events", thread count grows, and eventually FS stops processing new calls ([signalwire/freeswitch#2143](https://github.com/signalwire/freeswitch/issues/2143)).

**Why it happens:**
FS relies on the OS TCP stack to detect dead connections. Default TCP keepalive is often 2+ hours. During that window, events pile up. Even after the fix in FS PR #2275, older FS 1.10.x versions may not have this patch.

**How to avoid:**
1. Configure aggressive TCP keepalive on the ESL socket: `TCP_KEEPIDLE=10, TCP_KEEPINTVL=5, TCP_KEEPCNT=3` (detect dead connection in ~25s).
2. In `eslgo` connection setup, implement application-level heartbeat: send `api status` every 5s, treat no response within 3s as connection dead.
3. Set FS `event_socket.conf.xml` parameter `apply-inbound-acl` and consider adjusting queue size to something reasonable for your call volume.
4. On reconnect, re-subscribe to events and run channel reconciliation (Pitfall 1).

**Warning signs:**
- ESL command responses arriving with increasing latency
- Gap in received FS events (missing CHANNEL_HANGUP without CHANNEL_DESTROY)
- FS log showing "Lost events" messages

**Phase to address:**
Phase 1b (Real FS Integration) -- ESL connection management with heartbeat is foundational.

---

### Pitfall 3: A/B Leg State Machine Race Conditions

**What goes wrong:**
The callback flow has a multi-step state machine: ORIGINATING_A -> A_RINGING -> A_ANSWERED -> PARKING_A -> ORIGINATING_B -> B_RINGING -> B_ANSWERED -> BRIDGED -> COMPLETED. Race conditions arise when:
- A hangs up while B originate is in-flight (B originate succeeds but A is gone)
- B answers the exact moment A hangs up (CHANNEL_BRIDGE and CHANNEL_HANGUP arrive nearly simultaneously)
- FS delivers CHANNEL_HANGUP for A before the bgapi originate response for B arrives
- Two ESL events for the same call arrive on different goroutines and both try to transition state

**Why it happens:**
ESL events are asynchronous. `bgapi originate` returns a Job-UUID, and the actual result comes as a BACKGROUND_JOB event. Meanwhile, CHANNEL_* events for leg A continue flowing. Without careful sequencing, the state machine can end up in an impossible state (e.g., B is bridged but A's hangup was already processed, leaving B orphaned).

**How to avoid:**
1. Serialize all ESL events for a given call through a single goroutine (channel-per-call pattern). Use a `map[callID]chan Event` to route events to the owning goroutine.
2. Use a proper state machine with explicit valid transitions. Reject impossible transitions with logging rather than panicking.
3. For the A-hangup-during-B-originate case: when A hangs up, immediately send `uuid_kill` for B's UUID (which was allocated at originate time). Don't wait for B to answer.
4. Add a `finalizeCall` function that is idempotent and handles cleanup regardless of which state the call is in. Guard with `sync.Once`.

**Warning signs:**
- CDR records showing B-leg duration but no A-leg duration
- "Orphan channel" alerts from the watchdog (Pitfall 1)
- State transition errors in logs: "unexpected transition from X to Y"

**Phase to address:**
Phase 1a (Mock development) -- design and test the state machine exhaustively with mock events before touching real FS. Property-based testing with random event orderings is invaluable here.

---

### Pitfall 4: Prepaid Balance Double-Deduction and Overcharge

**What goes wrong:**
Two concurrent calls for the same client both read the current balance, both determine there is sufficient funds, both proceed to originate. The balance is deducted twice, potentially going negative. Alternatively: a call is billed for estimated duration at originate time, but the actual duration differs -- the refund/adjustment logic has its own race window.

**Why it happens:**
Read-then-write on the balance column without proper locking. Even with `SELECT ... FOR UPDATE`, if the transaction scope is too wide (spanning the entire call setup), you hold a row lock for seconds, causing contention. If the scope is too narrow, concurrent transactions slip through.

**How to avoid:**
1. Use a single atomic UPDATE with a WHERE clause: `UPDATE accounts SET balance = balance - :amount WHERE id = :id AND balance >= :amount`. Check `rows_affected == 1`. This is the canonical prepaid deduction pattern.
2. Pre-authorize (hold) the maximum estimated cost at originate time. On call completion, settle the actual cost. The hold and settlement are separate atomic operations.
3. Record the hold in a `transactions` table with status `HELD` -> `SETTLED` or `RELEASED`. This provides an audit trail and handles crash recovery (orphaned holds are released by a periodic sweep).
4. Never calculate balance by summing transactions on every call -- maintain a materialized balance column updated atomically.

**Warning signs:**
- Customer balance going negative
- Discrepancies between sum of transactions and the balance column
- Slow call setup times (symptom of lock contention on the balance row)

**Phase to address:**
Phase 1a (Core API) -- the billing model must be atomic from the first implementation. Retrofitting atomicity is extremely painful.

---

### Pitfall 5: Redis Concurrent Slot Counter Drift

**What goes wrong:**
The concurrent call slot counter in Redis (`INCR` on call start, `DECR` on call end) drifts over time. If a call's finalization code crashes after `INCR` but before the call completes, the counter is never decremented. Over days/weeks, the counter shows more concurrent calls than reality, eventually blocking the client from making any calls (false positive rate limit).

**Why it happens:**
`INCR` and `DECR` are individually atomic, but the pair is not. Network failures, Go process crashes, or panics between increment and the deferred decrement leave the counter permanently inflated. Using `defer` for decrement is insufficient because the decrement must happen when the *call* ends (an async ESL event), not when the *function* returns.

**How to avoid:**
1. Use Redis key-per-active-call pattern instead of a counter: `SET call:{uuid} 1 EX 3600` (with TTL as safety net). Count active calls with `KEYS call:*` or maintain a Redis SET. When a call ends, `DEL call:{uuid}`. If the app crashes, keys auto-expire.
2. If using INCR/DECR, add a periodic reconciliation job: compare Redis counter value against actual active calls in the database. Reset if drifted.
3. The `finalizeCall` function (which sends `DECR`) must be idempotent -- use a flag in the call record to prevent double-decrement.
4. Use a Lua script for atomic check-and-increment: `if redis.call('GET', key) < max then redis.call('INCR', key); return 1 else return 0 end`.

**Warning signs:**
- Clients reporting "concurrent call limit reached" when they have no active calls
- Redis counter value higher than `SELECT COUNT(*) FROM callback_calls WHERE status = 'active'`
- Counter never reaching zero even during off-hours

**Phase to address:**
Phase 1a (Core API) -- choose the right concurrency tracking pattern from the start. The key-per-call with TTL pattern is strongly preferred over bare INCR/DECR.

---

### Pitfall 6: Recording Alignment and Missing Audio

**What goes wrong:**
Three distinct failure modes:
1. Recording started before media is established -- FS returns error "Cannot execute app 'record_session' media required" ([signalwire/freeswitch#1052](https://github.com/signalwire/freeswitch/issues/1052))
2. A-leg and B-leg stereo recordings have different start times, so when ffmpeg merges them the audio is misaligned
3. A-leg hangs up first, FS stops processing the dialplan, and the recording file is not properly closed or uploaded

**Why it happens:**
`record_session` requires active media. In a callback flow, the A-leg is parked (no media partner), so recording cannot start until bridge. The B-leg recording starts when B answers, but the A-leg recording starts when bridge completes -- these can differ by milliseconds to seconds. If using `RECORD_STEREO`, both channels are in one file but FS stereo recording has known bugs where one channel contains noise ([darkworks/sipek2#18](https://github.com/darkworks/sipek2/issues/18)).

**How to avoid:**
1. Start recording on CHANNEL_BRIDGE event, not on originate. Use `uuid_record` ESL command targeting the A-leg UUID after bridge confirmation.
2. For separate A/B recordings destined for ffmpeg merge: record both using `uuid_record` triggered at the same CHANNEL_BRIDGE event. Accept that sub-second alignment differences are inevitable; use FS timestamps in CDR to calculate the offset for ffmpeg.
3. Handle CHANNEL_HANGUP by verifying the recording file exists and is valid (non-zero size) before attempting S3 upload.
4. Set `RECORD_WRITE_OVER=true` to avoid FS appending to existing files if a UUID is reused (edge case but possible).

**Warning signs:**
- Recording files with 0 bytes
- Customer complaints about echo or misaligned audio
- S3 upload failures on call teardown (file still being written)

**Phase to address:**
Phase 2 (Recording) -- but the `uuid_record` trigger point must be designed into the state machine in Phase 1a.

---

### Pitfall 7: Webhook Retry Storm and Endpoint Overwhelm

**What goes wrong:**
When a client's webhook endpoint goes down, the retry system generates exponentially more delivery attempts across all their calls. If 100 calls complete during a 30-minute outage and each gets 5 retries, that is 500 delivery attempts hitting the endpoint the moment it recovers. The thundering herd overwhelms the endpoint again, causing another round of failures.

**Why it happens:**
Naive retry implementations use fixed exponential backoff without jitter and without per-endpoint rate limiting. All retries for the same endpoint are scheduled independently.

**How to avoid:**
1. Implement per-endpoint circuit breaker: after N consecutive failures (e.g., 3), mark the endpoint as DOWN. Stop scheduling new deliveries. Probe with a single delivery every 60s. When probe succeeds, mark UP and drain queued deliveries with rate limiting.
2. Add jitter to retry delays: `delay = base_delay * 2^attempt * (0.8 + random(0.4))`.
3. Separate the webhook_deliveries table from the calls table (already planned per PROJECT.md) to avoid lock contention on the hot calls table.
4. Set a maximum retry count (e.g., 5) and maximum retry window (e.g., 24h). After exhaustion, move to DLQ with client notification.
5. Respect `Retry-After` headers from the client endpoint.

**Warning signs:**
- Webhook delivery worker queue depth growing unboundedly
- Spike in 5xx responses from a single client endpoint
- Database write amplification on webhook_deliveries table

**Phase to address:**
Phase 2 (Webhooks) -- the DLQ and circuit breaker must be designed from the start, not bolted on after the first storm.

---

### Pitfall 8: FreeSWITCH Gateway Health Detection Lag

**What goes wrong:**
A SIP gateway/trunk goes down but FS continues routing calls to it for minutes. All calls during this window fail at the SIP level, wasting the client's balance (if pre-deducted) and creating a burst of failed CDRs.

**Why it happens:**
FS gateway health checking via SIP OPTIONS has a minimum ping interval of 5 seconds, but the default detection requires multiple failed pings. A gateway that returns 503 on INVITE but 200 on OPTIONS appears healthy. Some carriers silently blackhole traffic without responding to OPTIONS.

**How to avoid:**
1. Implement application-level gateway health tracking in Go: maintain a sliding window of originate success/failure per gateway. If failure rate exceeds threshold (e.g., 3 consecutive failures or >50% in last 10 attempts), mark gateway as degraded and prefer alternatives.
2. Do not rely solely on FS `sofia status gateway` -- it only reflects registration and OPTIONS health, not actual call routing success.
3. For the B-leg prefix routing, maintain a primary/fallback gateway per prefix. On primary failure, auto-failover to fallback and alert operations.
4. Set `originate_timeout` to a reasonable value (30s) to avoid tying up resources on a non-responsive gateway.

**Warning signs:**
- Burst of ORIGINATOR_CANCEL or NO_ANSWER hangup causes for a single gateway
- A-leg connects but B-leg consistently fails (high loss rate in analytics)
- Gateway marked UP in `sofia status` but calls failing

**Phase to address:**
Phase 1b (Real FS Integration) -- gateway health must be tracked from the first real call. The routing service needs a health-aware gateway selector.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Bare INCR/DECR for concurrency slots | Simple, 2 lines of code | Counter drift requires reconciliation job, false positives block clients | Never -- use key-per-call with TTL from day one |
| Single ESL connection for all calls | Simple connection management | Single point of failure, event processing bottleneck at 100+ concurrent calls | Phase 1a/1b only, must add connection pooling before production |
| Polling FS for call state instead of event-driven | Avoids async complexity | Adds latency to state transitions, wastes FS resources, misses rapid state changes | Never for call control -- polling OK for monitoring/watchdog only |
| Storing recordings locally before S3 upload | Simpler pipeline, no streaming upload | Disk fills up under load, recordings lost on server crash | MVP only with aggressive disk monitoring and small call volume |
| In-memory call state (Go map) without DB persistence | Fast, no DB round-trip | All in-flight call state lost on process restart, no crash recovery | Phase 1a mock only, must persist to DB before real FS integration |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| eslgo library | Not calling `linger` on outbound connections, missing final CHANNEL_HANGUP/DESTROY events | Always send `linger` command after connecting. Without it, FS closes the socket before delivering final events |
| eslgo `bgapi` | Treating bgapi as synchronous -- blocking until response | bgapi returns a Job-UUID immediately. The actual result comes as a BACKGROUND_JOB event. Must correlate by Job-UUID |
| FreeSWITCH originate | Putting channel variables in the wrong position (after the endpoint instead of in `{}` prefix) | Variables go in `{var=val}` before the endpoint: `{originate_timeout=30}sofia/gateway/gw1/number` |
| FreeSWITCH uuid_record | Starting record before bridge, getting "media required" error | Trigger `uuid_record` only after CHANNEL_BRIDGE event confirms media is flowing |
| Redis + PostgreSQL | Assuming Redis state and PG state are always consistent | They will drift. PG is source of truth. Redis is a cache/optimization. Build reconciliation. Always check PG on critical decisions (balance) |
| ffmpeg recording merge | Assuming A and B leg recordings start at the exact same timestamp | Calculate offset from FS CDR timestamps (`bridge_epoch`, `answer_epoch`). Apply offset with ffmpeg `-itsoffset` |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Single ESL connection bottleneck | Event processing lag, commands queuing, high goroutine count waiting for responses | Use connection pool (2-4 connections) with command affinity by call UUID | ~50-100 concurrent calls |
| SELECT FOR UPDATE on balance row | Call setup latency >500ms, database CPU spike, lock wait timeouts | Use single atomic UPDATE with WHERE clause (no read-then-write) | ~20 concurrent calls per client |
| Webhook delivery in call teardown path | Call finalization delayed by slow/unreachable webhook endpoints | Async webhook delivery via queue (PubSub or separate goroutine pool). Never block call state machine on HTTP | ~10 concurrent calls with slow endpoints |
| Recording file I/O on FS server | FS CPU/IO spike during recording start/stop, call quality degradation | Use separate disk/partition for recordings, consider tmpfs for active recordings | ~50 concurrent recorded calls on standard disk |
| Full CDR table scan for analytics | Dashboard timeout, DB connection pool exhaustion | Partition cdr_records by date. Add composite indexes on (client_id, created_at). Consider materialized views for aggregates | ~1M CDR records |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| ESL password in source code or environment variable without encryption | Attacker gains full control of FreeSWITCH (can make calls, listen to recordings, modify routing) | Use Encore secrets manager. Rotate ESL password quarterly. Restrict ESL ACL to app server IPs only |
| No rate limiting on callback API | Attacker drains client balance by flooding callback requests | Per-client rate limit (calls/minute). Per-IP rate limit. Require valid API key with scope restrictions |
| Client A can query Client B's CDRs via parameter tampering | Data breach, compliance violation | Enforce client_id from auth context, never from request parameters. Middleware-level tenant isolation |
| Recording files accessible without authentication | Compliance violation (recordings contain PII/sensitive conversations) | S3 pre-signed URLs with short expiry (15 min). Never serve recordings through the app server. Audit access logs |
| Webhook payload contains sensitive call data sent over HTTP | Man-in-the-middle intercept of call metadata | Enforce HTTPS-only webhook URLs. Include HMAC signature header for payload verification. Allow clients to configure which fields are included |
| SIP credentials for gateways stored in plain text in FS config | Gateway compromise, toll fraud | Use FS `sofia` password storage with file permissions 600. Encrypt at rest. Audit gateway registration logs for unauthorized registrations |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No feedback between A-leg pickup and B-leg connect | A-party hears silence for 5-10s, assumes call failed, hangs up | Play ringback tone or "please wait" prompt to A while B is being dialed. Use `uuid_broadcast` to play audio to parked A-leg |
| Generic error messages on call failure ("Call failed") | Client cannot diagnose issues, floods support | Return structured error with hangup cause mapping: `NO_USER_RESPONSE` -> "B party did not answer", `USER_BUSY` -> "B party is busy" |
| Dashboard showing stale call status | Admin sees "active" calls that ended minutes ago | WebSocket push for real-time status. Show "last updated" timestamp. Auto-refresh with exponential backoff |
| Exposing SIP-level error codes to end users | Confusing, meaningless to business users | Map SIP codes to human-readable business categories: success, no answer, busy, network error, invalid number |
| No indication of remaining balance before call | Client discovers insufficient funds only after A-leg is already connected | Check balance in API response (before originate). Show estimated cost and remaining balance. Block if insufficient with clear message |

## "Looks Done But Isn't" Checklist

- [ ] **A-leg parking:** Verify `park_timeout` is set in originate string, not just in Go code -- test by killing Go process during a parked call and confirming FS auto-hangs up after timeout
- [ ] **Call state machine:** Verify all terminal states (FAILED, COMPLETED, TIMEOUT) trigger `finalizeCall` -- test by simulating every hangup cause (A hangs up during B ring, B rejects, network error, FS restart)
- [ ] **Balance deduction:** Verify negative balance is impossible -- test with concurrent calls at exact remaining balance boundary using parallel goroutines
- [ ] **Redis counter:** Verify counter matches reality after 1000 calls with random crashes -- run chaos test killing the process at random points in call lifecycle
- [ ] **Recording upload:** Verify recordings are uploaded even when A hangs up first -- test both A-first and B-first hangup scenarios
- [ ] **Webhook delivery:** Verify DLQ works by setting webhook URL to a non-routable IP -- confirm all events eventually land in DLQ with correct metadata
- [ ] **Gateway failover:** Verify calls route to backup gateway when primary is down -- test by blocking primary gateway IP and confirming B-leg succeeds via backup
- [ ] **ESL reconnect:** Verify in-flight calls survive ESL reconnection -- test by restarting Go process during active bridged calls and confirming calls continue (they should, bridge is in FS)
- [ ] **Tenant isolation:** Verify Client A cannot see Client B's calls, recordings, CDRs, or balance -- test with explicit cross-tenant API requests

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Leaked parked channels | LOW | Run `show channels` on FS, identify channels with state CS_PARK older than 2 minutes, `uuid_kill` each. Automate as cron |
| ESL event queue saturation | MEDIUM | Restart FS (causes all active calls to drop). Alternative: apply PR #2275 patch and restart cleanly |
| State machine in impossible state | MEDIUM | Mark call as FAILED with reason "state_error". Ensure finalizeCall runs (release balance hold, decrement counter, hangup both legs). Review logs to fix root cause |
| Balance drift (negative or incorrect) | HIGH | Recalculate all client balances from transaction ledger (`SELECT SUM(amount) FROM transactions WHERE client_id = X GROUP BY type`). This is why the transactions table is critical -- it is the audit trail for recovery |
| Redis counter drift | LOW | Run reconciliation: `SELECT client_id, COUNT(*) FROM callback_calls WHERE status = 'active' GROUP BY client_id`. Set Redis counters to match. Schedule as periodic job |
| Missing recordings | HIGH | Cannot recover audio after the fact. Prevention only. If FS recording file exists but S3 upload failed, retry from FS disk. Add monitoring for recording files not uploaded within 5 minutes of call end |
| Webhook storm aftermath | MEDIUM | Pause all deliveries for the affected endpoint. Drain DLQ manually. Resume with rate limiting. Notify client of delivery gap |
| Gateway health misdetection | LOW | Manual override: mark gateway as DOWN in admin dashboard. Route all traffic to backup. Investigate SIP-level connectivity. Re-enable after confirmed healthy |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Parked channel leak | Phase 1b (Real FS) | Kill Go process during parked call, verify FS auto-cleans up within park_timeout seconds |
| ESL queue saturation | Phase 1b (Real FS) | Simulate network partition (iptables DROP), verify Go app detects within 30s and reconnects |
| A/B state machine races | Phase 1a (Mock) | Property-based tests with randomized event orderings covering all state transitions |
| Balance double-deduction | Phase 1a (Core API) | Parallel goroutine test: 10 goroutines deducting from balance that can only cover 1 call |
| Redis counter drift | Phase 1a (Core API) | Chaos test: 1000 calls with random process kills, verify counter matches DB count |
| Recording alignment | Phase 2 (Recording) | Record 10 test calls, verify ffmpeg merge produces correctly aligned stereo audio |
| Webhook retry storm | Phase 2 (Webhooks) | Set endpoint to return 503 for 5 minutes, verify circuit breaker activates and no thundering herd on recovery |
| Gateway health lag | Phase 1b (Real FS) | Block primary gateway, verify failover within 3 failed call attempts (not minutes of SIP OPTIONS) |

## Sources

- [FreeSWITCH: Stuck calls on inbound parked voice call leg (#1688)](https://github.com/signalwire/freeswitch/issues/1688)
- [FreeSWITCH: ESL event queue saturation on network loss (#2143)](https://github.com/signalwire/freeswitch/issues/2143)
- [FreeSWITCH: Recording not starting immediately with originate (#1052)](https://github.com/signalwire/freeswitch/issues/1052)
- [FreeSWITCH: BRIDGE THREAD DONE but channel not close (#1963)](https://github.com/signalwire/freeswitch/issues/1963)
- [FreeSWITCH: mod_sofia SIP BYE processing delay (#1268)](https://github.com/signalwire/freeswitch/issues/1268)
- [FreeSWITCH: park_timeout documentation](https://developer.signalwire.com/freeswitch/Channel-Variables-Catalog/park_timeout_16352650/)
- [FreeSWITCH: RECORD_STEREO documentation](https://developer.signalwire.com/freeswitch/Channel-Variables-Catalog/RECORD_STEREO_16352883/)
- [FreeSWITCH: mod_event_socket documentation](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Modules/mod_event_socket_1048924/)
- [FreeSWITCH: Gateway Configuration](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Configuration/Sofia-SIP-Stack/Gateways-Configuration_7144069/)
- [Redis INCR race condition patterns](https://dev.to/silentwatcher_95/fixing-race-conditions-in-redis-counters-why-lua-scripting-is-the-key-to-atomicity-and-reliability-38a4)
- [Webhook retry best practices (Svix)](https://www.svix.com/resources/webhook-best-practices/retries/)
- [Webhook retry strategies (Hookdeck)](https://hookdeck.com/outpost/guides/outbound-webhook-retry-best-practices)
- [eslgo library](https://github.com/percipia/eslgo)
- [FreeSWITCH: Missing CHANNEL_HANGUP event discussion](https://freeswitch-users.freeswitch.narkive.com/AOv90cFX/missing-channel-hangup-event-in-mod-event-socket)

---
*Pitfalls research for: API Callback/Click-to-Call System (FreeSWITCH ESL + Encore.go)*
*Researched: 2026-03-09*
