# Architecture Research

**Domain:** API Callback / Click-to-Call (FreeSWITCH ESL + Encore.go)
**Researched:** 2026-03-09
**Confidence:** HIGH (core patterns well-established in telecom, Encore.go patterns from official docs)

## System Overview

```
                          ┌─────────────────────────────────────────────────────┐
                          │                   API Layer (Encore.go)              │
                          │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ │
                          │  │ Callback │ │  Admin   │ │  Client  │ │  Auth  │ │
                          │  │ Service  │ │ Service  │ │ Service  │ │Service │ │
                          │  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬───┘ │
                          ├───────┼────────────┼────────────┼────────────┼──────┤
                          │       │     Call Control Layer                │      │
                          │  ┌────┴─────────────────────────────────┐    │      │
                          │  │         Call Engine Service           │    │      │
                          │  │  ┌───────────┐  ┌──────────────────┐ │    │      │
                          │  │  │ ESL Client│  │ Call State Machine│ │    │      │
                          │  │  │ (FSClient)│  │    (per-call)     │ │    │      │
                          │  │  └─────┬─────┘  └──────────────────┘ │    │      │
                          │  └────────┼────────────────────────────-┘    │      │
                          ├───────────┼─────────────────────────────────┼──────┤
                          │           │     Supporting Services         │      │
                          │  ┌────────┤  ┌──────────┐ ┌──────────┐ ┌───┴────┐ │
                          │  │Routing │  │ Billing  │ │Recording │ │Webhook │ │
                          │  │Service │  │ Service  │ │ Service  │ │Service │ │
                          │  └────────┘  └──────────┘ └──────────┘ └────────┘ │
                          ├───────────────────────────────────────────────────-┤
                          │                   Data Layer                        │
                          │  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
                          │  │PostgreSQL│  │  Redis   │  │ S3/Minio │          │
                          │  │ (Encore) │  │ (slots)  │  │(records) │          │
                          │  └──────────┘  └──────────┘  └──────────┘          │
                          └───────────────────────────────────────────────────-┘
                                    │
                          ┌─────────┴──────────┐
                          │    FreeSWITCH       │
                          │  ┌───────┐ ┌─────┐  │
                          │  │  SIP  │ │ RTP │  │
                          │  │Signaling│ │Media│  │
                          │  └───────┘ └─────┘  │
                          │  ESL :8021           │
                          └────────────────────-┘
```

### Component Responsibilities

| Component | Responsibility | Communicates With |
|-----------|----------------|-------------------|
| **Callback Service** | API endpoint for initiating calls, validates input, checks balance/slots | Call Engine, Billing |
| **Call Engine Service** | Manages ESL connection, call state machines, orchestrates A/B leg flow | FreeSWITCH (ESL), Routing, Recording, Webhook |
| **Routing Service** | Prefix-based gateway selection for B-leg, round-robin for A-leg | Call Engine (called by) |
| **Billing Service** | Balance pre-deduction, CDR generation, rate plan lookup, transaction log | PostgreSQL, Call Engine (called by) |
| **Recording Service** | Triggers FS recording, merges A/B tracks (ffmpeg), uploads to S3 | FreeSWITCH, S3, Call Engine (called by) |
| **Webhook Service** | Async event delivery to customer endpoints with retry + DLQ | PostgreSQL (webhook_deliveries), external HTTP |
| **Auth Service** | JWT/API key auth, admin vs client role enforcement | All services (middleware) |
| **Admin Service** | Dashboard APIs: gateway CRUD, user mgmt, system stats | PostgreSQL, all services |
| **Client Service** | Client portal APIs: call history, balance, API key mgmt | PostgreSQL, Billing |

## Recommended Project Structure

```
bos3000/
├── encore.app                    # Encore application config
├── auth/                         # Auth service
│   ├── auth.go                   # Auth handler, JWT/API key validation
│   ├── middleware.go             # Role-based middleware (admin/client)
│   └── migrations/
│       └── 1_create_users.up.sql
├── callback/                     # Callback initiation service
│   ├── callback.go               # POST /callback/initiate API
│   ├── callback_test.go
│   └── types.go                  # Request/response types
├── callengine/                   # Core call control service
│   ├── callengine.go             # Service struct, ESL event loop
│   ├── fsclient.go               # FSClient interface + eslgo wrapper
│   ├── fsclient_mock.go          # Mock implementation for dev/test
│   ├── statemachine.go           # Per-call state machine
│   ├── statemachine_test.go
│   ├── events.go                 # ESL event handlers
│   └── types.go
├── routing/                      # Gateway routing service
│   ├── routing.go                # Prefix match + round-robin
│   ├── routing_test.go
│   └── migrations/
│       └── 1_create_gateways.up.sql
├── billing/                      # Billing + balance service
│   ├── billing.go                # Pre-deduct, finalize, refund
│   ├── billing_test.go
│   ├── rateplan.go               # Rate calculation
│   └── migrations/
│       └── 1_create_billing.up.sql
├── recording/                    # Recording pipeline
│   ├── recording.go              # Merge + upload orchestration
│   └── recording_test.go
├── webhook/                      # Webhook delivery service
│   ├── webhook.go                # Delivery + retry logic
│   ├── webhook_test.go
│   └── migrations/
│       └── 1_create_webhooks.up.sql
├── admin/                        # Admin dashboard APIs
│   ├── admin.go
│   └── types.go
├── client/                       # Client portal APIs
│   ├── client.go
│   └── types.go
├── shared/                       # Shared types (NOT a service)
│   ├── callstates.go             # Call state enum
│   └── events.go                 # Domain event types
└── frontend/                     # React/Vite/shadcn frontend
    ├── package.json
    ├── src/
    │   ├── admin/                # Admin dashboard
    │   └── portal/               # Client portal
    └── vite.config.ts
```

### Structure Rationale

- **callengine/ is the heart:** All FreeSWITCH interaction funnels through this one service. No other service touches ESL directly. This is critical for call state consistency.
- **callback/ is thin:** It validates, checks balance/slots, then delegates to callengine. Separation keeps the API surface clean and testable.
- **billing/ owns money:** All balance mutations happen here via PG row-level locks. No other service modifies balances.
- **webhook/ is independent:** Decoupled from call flow. Call engine publishes events via Encore PubSub; webhook service subscribes and handles delivery/retry independently.
- **shared/ is NOT a service:** Encore convention -- sub-packages cannot define APIs. This holds shared types only.

## Architectural Patterns

### Pattern 1: ESL Inbound Mode with bgapi + Event Correlation

**What:** The Call Engine maintains a single persistent ESL inbound connection to FreeSWITCH. All call operations use `bgapi` (background API) to avoid blocking the connection. Each bgapi call returns a Job-UUID; the engine subscribes to BACKGROUND_JOB events and correlates responses by Job-UUID.

**When to use:** Always. This is the only correct pattern for a multi-call system over a single ESL connection.

**Trade-offs:** Requires careful event correlation logic, but avoids the complexity of managing per-call outbound ESL connections.

**Example:**
```go
// FSClient interface -- the 5 methods from PROJECT.md
type FSClient interface {
    // Originate a call, returns Job-UUID for correlation
    Originate(ctx context.Context, dest string, vars map[string]string) (string, error)
    // Bridge two UUIDs
    Bridge(ctx context.Context, uuidA, uuidB string) error
    // Hangup a specific channel
    Hangup(ctx context.Context, uuid string, cause string) error
    // Park a channel (hold with MOH or silence)
    Park(ctx context.Context, uuid string) error
    // Execute an app on a channel (record, playback, etc.)
    Execute(ctx context.Context, uuid string, app string, args string) error
}

// Event correlation via channel map
type CallEngine struct {
    conn        *eslgo.Conn
    pendingJobs sync.Map // jobUUID -> chan *eslgo.RawResponse
    calls       sync.Map // callUUID -> *CallStateMachine
}
```

### Pattern 2: Per-Call State Machine

**What:** Each active call gets its own state machine instance tracking the A/B leg lifecycle. States are explicit; transitions are driven by ESL events. The state machine is the single source of truth for "where is this call right now."

**When to use:** Every call. No exceptions.

**Trade-offs:** Memory overhead per call (negligible -- struct with enum + timestamps). The alternative (ad-hoc if/else chains) leads to impossible-to-debug race conditions.

**State transitions:**

```
                    API Request
                        │
                        ▼
                ┌───────────────┐
                │   INITIATED   │  Balance pre-deducted, slot reserved
                └───────┬───────┘
                        │ originate A-leg
                        ▼
                ┌───────────────┐
                │  A_DIALING    │  Waiting for A to answer
                └───────┬───────┘
                   ┌────┴────┐
                   │         │
              A answers   A fails/timeout
                   │         │
                   ▼         ▼
           ┌──────────┐  ┌──────────┐
           │ A_PARKED  │  │ A_FAILED │──→ FINALIZED (refund)
           └─────┬─────┘  └──────────┘
                 │ originate B-leg
                 ▼
           ┌──────────┐
           │ B_DIALING │  A is parked, waiting for B
           └─────┬─────┘
              ┌──┴──┐
              │     │
         B answers  B fails/timeout
              │     │
              ▼     ▼
        ┌─────────┐ ┌──────────┐
        │ BRIDGED  │ │ B_FAILED │──→ hangup A ──→ FINALIZED (partial refund)
        └────┬─────┘ └──────────┘
             │ either side hangs up
             ▼
        ┌──────────┐
        │ COMPLETED │  CDR generated, duration calculated
        └────┬─────┘
             │ finalize billing, upload recording, send webhook
             ▼
        ┌──────────┐
        │ FINALIZED │  Slot released, balance adjusted, recording archived
        └──────────┘
```

**Example:**
```go
type CallState int

const (
    StateInitiated CallState = iota
    StateADialing
    StateAParked
    StateAFailed
    StateBDialing
    StateBridged
    StateBFailed
    StateCompleted
    StateFinalized
)

type CallStateMachine struct {
    mu         sync.Mutex
    callID     string
    state      CallState
    aLegUUID   string
    bLegUUID   string
    clientID   string
    startTime  time.Time
    answerTime time.Time
    bridgeTime time.Time
    endTime    time.Time
    hangupCause string
}

func (csm *CallStateMachine) Transition(event ESLEvent) (CallState, []Action, error) {
    csm.mu.Lock()
    defer csm.mu.Unlock()
    // Validate transition is legal, return actions to execute
}
```

### Pattern 3: Atomic Balance Pre-Deduction with PG Row Locks

**What:** Before originating any call, deduct estimated cost from client balance using `SELECT ... FOR UPDATE` + `UPDATE` in a single transaction. If call fails or is shorter than estimated, refund the difference in finalization.

**When to use:** Every call initiation.

**Trade-offs:** Row-level lock means concurrent calls for the same client serialize on balance check. This is correct behavior -- it prevents overdraft. For different clients, there is zero contention.

**Example:**
```go
func (s *BillingService) PreDeduct(ctx context.Context, clientID string, estimatedMinutes float64, ratePerMin float64) (txID string, err error) {
    return sqldb.ExecTx(ctx, s.db, func(tx *sqldb.Tx) error {
        var balance float64
        err := tx.QueryRow(ctx,
            `SELECT balance FROM users WHERE id = $1 FOR UPDATE`, clientID,
        ).Scan(&balance)
        if err != nil { return err }

        cost := estimatedMinutes * ratePerMin
        if balance < cost {
            return errs.B().Code(errs.FailedPrecondition).Msg("insufficient balance").Err()
        }

        _, err = tx.Exec(ctx,
            `UPDATE users SET balance = balance - $1 WHERE id = $2`, cost, clientID)
        // Also INSERT into transactions table for audit
        return err
    })
}
```

### Pattern 4: Redis Concurrent Slot Control

**What:** Use Redis INCR/DECR as a distributed counting semaphore. Each client has a key `slots:{clientID}` with their current active call count. INCR before call, DECR in finalizeCall (not defer).

**When to use:** Every call initiation, alongside balance check.

**Trade-offs:** Redis failure = cannot initiate calls (fail-closed, which is correct). Must guarantee DECR happens exactly once per call -- hence the "finalizeCall releases slot" decision from PROJECT.md.

**Critical detail:** Use a Lua script for atomic check-and-increment to prevent races:

```lua
-- KEYS[1] = slots:{clientID}
-- ARGV[1] = max_slots
local current = redis.call('GET', KEYS[1]) or 0
if tonumber(current) >= tonumber(ARGV[1]) then
    return -1  -- at capacity
end
return redis.call('INCR', KEYS[1])
```

### Pattern 5: Webhook Delivery with Exponential Backoff + DLQ

**What:** Call events published via Encore PubSub. Webhook service subscribes, looks up client's webhook URL, delivers HTTP POST. On failure: exponential backoff (5s, 30s, 2m, 15m, 1h, 6h) up to 6 attempts. After exhausting retries, mark as dead-lettered for manual inspection.

**When to use:** All client-facing call events (call.started, call.answered, call.bridged, call.completed, call.failed).

**Trade-offs:** Separate webhook_deliveries table avoids lock contention on callback_calls. Each delivery attempt is a row update (attempt_count, next_retry_at, last_error).

## Data Flow

### Primary Call Flow (Happy Path)

```
Client API Request (POST /callback/initiate)
    │
    ▼
[Callback Service]
    │ 1. Validate request (A/B numbers, client auth)
    │ 2. Check concurrent slots (Redis INCR)
    │ 3. Pre-deduct balance (PG row lock)
    │ 4. Create callback_calls record (state=INITIATED)
    │
    ▼
[Call Engine Service]
    │ 5. Originate A-leg via ESL bgapi
    │    → FS sends CHANNEL_CREATE, CHANNEL_ANSWER events
    │ 6. On A answer: park A-leg, update state=A_PARKED
    │ 7. Route B-leg (Routing Service → gateway selection)
    │ 8. Originate B-leg via ESL bgapi
    │    → FS sends CHANNEL_CREATE, CHANNEL_ANSWER events
    │ 9. On B answer: uuid_bridge(A, B), update state=BRIDGED
    │ 10. Start recording (Execute record on bridge event)
    │
    ▼
[FreeSWITCH]
    │ 11. Media flows between A and B
    │ 12. Either party hangs up → CHANNEL_HANGUP event
    │
    ▼
[Call Engine Service]
    │ 13. On hangup: update state=COMPLETED
    │ 14. finalizeCall():
    │     a. Calculate actual duration
    │     b. Finalize billing (refund excess pre-deduction)
    │     c. Release Redis slot (DECR)
    │     d. Trigger recording merge + upload
    │     e. Publish call.completed event (PubSub)
    │     f. Update state=FINALIZED
    │
    ▼
[Webhook Service] (async, via PubSub subscription)
    │ 15. Deliver webhook to client endpoint
    │ 16. Retry with exponential backoff on failure
```

### ESL Event Flow

```
FreeSWITCH ESL (:8021)
    │
    │ persistent TCP connection (inbound mode)
    │
    ▼
[Call Engine - ESL Event Loop]
    │
    │ Subscribed events:
    │   CHANNEL_CREATE, CHANNEL_ANSWER, CHANNEL_BRIDGE,
    │   CHANNEL_UNBRIDGE, CHANNEL_HANGUP, CHANNEL_HANGUP_COMPLETE,
    │   BACKGROUND_JOB, RECORD_START, RECORD_STOP
    │
    │ Event routing:
    │   1. Extract Unique-ID (channel UUID)
    │   2. Look up CallStateMachine by UUID (A-leg or B-leg)
    │   3. Feed event to state machine → get actions
    │   4. Execute actions (bridge, hangup, record, finalize)
    │
    ▼
[Per-Call State Machine]
    │ Produces actions:
    │   - ESL commands (bridge, hangup, record)
    │   - Service calls (billing finalize, webhook publish)
    │   - State updates (DB writes)
```

### Recording Pipeline

```
[CHANNEL_BRIDGE event]
    │
    ▼
[Call Engine]
    │ Execute: uuid_record A-leg (A-side audio)
    │ Execute: uuid_record B-leg (B-side audio)
    │
    ▼
[CHANNEL_HANGUP event]
    │
    ▼
[Recording Service]
    │ 1. Wait for FS to finalize recording files
    │ 2. ffmpeg merge A + B tracks → stereo or mixed mono
    │ 3. Upload to S3/Minio with call_id as key
    │ 4. Update callback_calls with recording_url
    │ 5. Clean up local temp files
```

## Database Schema Patterns

### Core Tables

```sql
-- Users / Clients with embedded balance
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT UNIQUE NOT NULL,
    password    TEXT NOT NULL,  -- bcrypt
    role        TEXT NOT NULL CHECK (role IN ('admin', 'client')),
    api_key     TEXT UNIQUE,
    balance     DECIMAL(12,4) NOT NULL DEFAULT 0,
    max_slots   INT NOT NULL DEFAULT 5,
    webhook_url TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The main call record, doubles as live state + CDR
CREATE TABLE callback_calls (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id       UUID NOT NULL REFERENCES users(id),
    a_number        TEXT NOT NULL,
    b_number        TEXT NOT NULL,
    a_leg_uuid      TEXT,
    b_leg_uuid      TEXT,
    state           TEXT NOT NULL DEFAULT 'initiated',
    a_gateway_id    UUID REFERENCES fs_gateways(id),
    b_gateway_id    UUID REFERENCES fs_gateways(id),
    initiated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    a_answer_at     TIMESTAMPTZ,
    b_answer_at     TIMESTAMPTZ,
    bridge_at       TIMESTAMPTZ,
    end_at          TIMESTAMPTZ,
    duration_sec    INT,
    billable_sec    INT,
    hangup_cause    TEXT,
    hangup_side     TEXT,  -- 'a' or 'b'
    recording_url   TEXT,
    pre_deduct_amt  DECIMAL(12,4),
    final_cost      DECIMAL(12,4),
    rate_per_min    DECIMAL(8,4),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_calls_client ON callback_calls(client_id, created_at DESC);
CREATE INDEX idx_calls_state ON callback_calls(state) WHERE state NOT IN ('finalized');

-- Transactions for audit trail
CREATE TABLE transactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id   UUID NOT NULL REFERENCES users(id),
    call_id     UUID REFERENCES callback_calls(id),
    type        TEXT NOT NULL, -- 'pre_deduct', 'finalize', 'refund', 'topup'
    amount      DECIMAL(12,4) NOT NULL,
    balance_after DECIMAL(12,4) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Webhook delivery tracking (separate from calls to avoid lock contention)
CREATE TABLE webhook_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id       UUID NOT NULL REFERENCES users(id),
    call_id         UUID NOT NULL REFERENCES callback_calls(id),
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    url             TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending', -- pending, delivered, failed, dead_letter
    attempt_count   INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 6,
    next_retry_at   TIMESTAMPTZ,
    last_error      TEXT,
    last_status_code INT,
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_webhooks_pending ON webhook_deliveries(next_retry_at)
    WHERE status IN ('pending', 'failed');
```

### Schema Design Rationale

- **callback_calls is both live state and CDR:** No separate CDR table. The call record accumulates timestamps as the call progresses. Once finalized, it IS the CDR. Simpler than dual-table sync.
- **transactions as append-only ledger:** Never update; always insert. This gives full audit trail. balance_after field enables point-in-time balance reconstruction.
- **webhook_deliveries separate from calls:** Key decision from PROJECT.md. Retry UPDATE cycles would cause lock contention on the hot callback_calls table.
- **Partial index on active calls:** `WHERE state NOT IN ('finalized')` keeps the index small -- most calls are finalized.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 0-50 concurrent calls | Single FreeSWITCH, single Encore instance. All in one PG database. Redis optional (can use PG advisory locks). |
| 50-500 concurrent calls | Single FreeSWITCH (handles thousands), single Encore. Redis required for slot counting. Monitor PG connection pool. |
| 500-5000 concurrent calls | FreeSWITCH HA pair (active-standby). PG read replicas for dashboard queries. Recording pipeline may need queue (ffmpeg is CPU-bound). |
| 5000+ concurrent calls | Multiple FreeSWITCH nodes with call engine routing. Shard by client. Recording offloaded to worker pool. |

### Scaling Priorities

1. **First bottleneck: ffmpeg recording merge.** CPU-intensive, blocks if done synchronously. Solution: async job queue from day one (Encore PubSub topic for merge jobs).
2. **Second bottleneck: PG row locks on hot clients.** A client making 50 calls/sec will serialize on balance check. Solution: batch pre-deduction or credit-based billing at scale.
3. **Third bottleneck: ESL event throughput.** Single ESL connection handles ~1000 events/sec. Solution: multiple ESL connections with event sharding by UUID prefix.

## Anti-Patterns

### Anti-Pattern 1: Blocking API Originate

**What people do:** The `/callback/initiate` API blocks until the call is fully bridged (or fails), returning the final status.
**Why it's wrong:** A call can take 30-60 seconds to complete A+B dialing. This ties up HTTP connections, causes timeouts, and makes the API unusable under load.
**Do this instead:** Return immediately with call_id after initiating. Client polls or receives webhook for status updates. The API response is "call initiated" not "call completed."

### Anti-Pattern 2: Per-Call ESL Connections

**What people do:** Open a new ESL connection for each call, manage it in a goroutine.
**Why it's wrong:** TCP connection overhead, FreeSWITCH connection limits, impossible to correlate events across calls.
**Do this instead:** Single persistent ESL inbound connection (or small pool), multiplex all calls over it using bgapi + event correlation by UUID.

### Anti-Pattern 3: Using defer for Slot Release

**What people do:** `defer releaseSlot(clientID)` at the top of the call handler.
**Why it's wrong:** ESL events are asynchronous. The initiating goroutine may return long before the call ends. Defer fires on function return, not on call completion.
**Do this instead:** Release slot in `finalizeCall()` which is triggered by CHANNEL_HANGUP_COMPLETE event. This is explicitly called out in PROJECT.md.

### Anti-Pattern 4: Separate CDR Table with Sync

**What people do:** Maintain a live `calls` table and a separate `cdr` table, copying data on completion.
**Why it's wrong:** Data sync bugs, missing fields, dual-write complexity. Two sources of truth.
**Do this instead:** Single `callback_calls` table that accumulates data as the call progresses. Once finalized, it IS the CDR. Query by state for live vs historical.

### Anti-Pattern 5: Synchronous Webhook Delivery in Call Flow

**What people do:** Deliver webhooks inside the call finalization path. If webhook delivery is slow, call finalization is delayed.
**Why it's wrong:** External HTTP calls are unreliable. A slow/down webhook endpoint should never delay slot release or billing finalization.
**Do this instead:** Publish event via Encore PubSub. Webhook service subscribes and delivers asynchronously. Call finalization completes independently.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| FreeSWITCH ESL | Persistent TCP inbound connection on :8021 | Reconnect with backoff on disconnect. Health check via `status` command. |
| S3/Minio | Standard SDK upload after ffmpeg merge | Use presigned URLs for client download in portal |
| Redis | Connection pool, Lua scripts for atomic ops | Fail-closed: if Redis down, reject new calls |
| ffmpeg | CLI subprocess for A/B track merge | Run async, not in call path. Monitor for zombie processes. |

### Internal Boundaries (Encore Service-to-Service)

| Boundary | Communication | Notes |
|----------|---------------|-------|
| Callback -> Call Engine | Direct API call | Sync: returns call_id immediately |
| Call Engine -> Billing | Direct API call | Sync: pre-deduct must complete before originate |
| Call Engine -> Routing | Direct API call | Sync: need gateway before originate |
| Call Engine -> Recording | PubSub event | Async: recording merge is slow, must not block |
| Call Engine -> Webhook | PubSub event | Async: webhook delivery is external, unreliable |
| Admin/Client -> all | Direct API calls | Read-only queries, no call-flow dependency |

### FreeSWITCH HA Pattern

```
                ┌──────────────┐
                │  Call Engine  │
                │              │
                │ health_check │
                │   timer(5s)  │
                └──┬───────┬───┘
                   │       │
              primary  standby
                   │       │
                   ▼       ▼
            ┌─────────┐ ┌─────────┐
            │  FS-1   │ │  FS-2   │
            │ (active)│ │(standby)│
            └─────────┘ └─────────┘
```

- Call Engine sends `status` command every 5s to both FS nodes
- If primary fails 3 consecutive health checks, promote standby
- Active calls on failed node are lost (SIP is not stateful across nodes) -- CDR records marked as `failed` with cause `node_failure`
- New calls route to the newly promoted primary

## Build Order Implications

Based on component dependencies, the recommended build sequence:

1. **Auth + Database Schema** -- Everything depends on users table and authentication. Build first.
2. **Call Engine with Mock FSClient** -- Core state machine logic, event handling, can be fully unit-tested without FreeSWITCH.
3. **Billing Service** -- Pre-deduct/finalize pattern. Call Engine depends on this.
4. **Routing Service** -- Gateway selection. Call Engine depends on this.
5. **Callback Service** -- Thin API layer wiring Auth + Billing + Call Engine. Needs 2-4.
6. **Real FSClient (eslgo integration)** -- Swap mock for real ESL. Needs running FreeSWITCH.
7. **Recording Service** -- Depends on real FS for actual recording files.
8. **Webhook Service** -- Independent of call flow, can be built in parallel with 6-7.
9. **Admin/Client Services** -- Dashboard APIs, read-heavy, depend on data from all above.
10. **Frontend** -- React app, depends on API surface being stable.
11. **FreeSWITCH HA** -- Polish item, needs two FS nodes.

**Key insight:** Steps 1-5 can be built and fully tested with zero FreeSWITCH dependency thanks to Mock FSClient. This is the Phase 1a from PROJECT.md.

## Sources

- [FreeSWITCH Event Socket Library Documentation](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Client-and-Developer-Interfaces/Event-Socket-Library/)
- [FreeSWITCH Channel States](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Dialplan/Channel-States_7144639/)
- [FreeSWITCH Call States](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Dialplan/Call-States_32178212/)
- [FreeSWITCH Event List](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Introduction/Event-System/Event-List_7143557/)
- [FreeSWITCH mod_event_socket](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Modules/mod_event_socket_1048924/)
- [eslgo - Go FreeSWITCH ESL Library](https://github.com/percipia/eslgo)
- [FreeSWITCH Originate Examples](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Examples/Originate-Example_10682745/)
- [Originating Calls in FreeSWITCH (Nick vs Networking)](https://nickvsnetworking.com/originating-calls-in-freeswitch/)
- [Redis INCR for Rate Limiting](https://redis.io/commands/incr)
- [Redis Counting Semaphore Pattern](https://medium.com/@17sheetalsharma/redis-counting-semaphore-2db63d5807d3)
- [Webhook Retry Best Practices (Hookdeck)](https://hookdeck.com/webhooks/guides/webhook-retry-best-practices)
- [Webhook Retry Best Practices (Svix)](https://www.svix.com/resources/webhook-best-practices/retries/)

---
*Architecture research for: BOS3000 API Callback System*
*Researched: 2026-03-09*
