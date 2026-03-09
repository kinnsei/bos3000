# Feature Research

**Domain:** API Callback / Click-to-Call (B2B Wholesale Telecom)
**Researched:** 2026-03-09
**Confidence:** HIGH (domain well-established, patterns clear from Twilio/wholesale VoIP platforms)

## Feature Landscape

### Table Stakes (Users Expect These)

Features B2B telecom clients assume exist. Missing these = platform is not production-ready.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| API-initiated A/B dual-outbound callback | Core product. Without it there is no product. | HIGH | A-leg originate, park/hold, B-leg originate, bridge. The entire call flow engine. |
| Call Detail Records (CDR) | Every telecom platform generates CDRs. Clients need them for reconciliation. | MEDIUM | Must capture: caller, callee, start/end time, duration, disposition, gateway used, cost. Written after call ends. |
| Per-second / per-minute billing with rate plans | Clients expect accurate, transparent billing. Incorrect billing = immediate churn. | HIGH | Rate plan per client, prefix-based rating, prepaid balance model with atomic deduction. |
| Balance management (prepaid) | B2B wholesale always works on prepaid credit. No balance = calls blocked. | MEDIUM | Top-up, deduction, refund on failed calls, transaction ledger. PG row-lock for atomicity. |
| Concurrent call limits | Prevents a single client from exhausting platform capacity. Standard in wholesale. | LOW | Redis INCR/DECR per client. Reject calls when limit hit. |
| Call recording | Regulatory requirement in many jurisdictions. Clients expect recordings for QA. | HIGH | A/B split-track recording, post-call merge via ffmpeg, S3 upload, playback/download via portal. |
| Webhook notifications | Clients integrate callbacks into their own systems. Must know call status in real-time. | MEDIUM | Events: call.initiated, a_leg.answered, b_leg.answered, call.bridged, call.ended, recording.ready. Retry with exponential backoff + DLQ. |
| Admin portal (operations dashboard) | Platform operators need visibility into all clients, calls, gateways, billing. | HIGH | Client CRUD, gateway management, CDR search, financial overview, system health. |
| Client self-service portal | Clients expect to see their own calls, balance, recordings without contacting support. | MEDIUM | Dashboard, CDR search, balance history, recording playback, API key management. |
| API key authentication | B2B API platforms authenticate via API keys. Table stakes for developer experience. | LOW | Key generation, rotation, per-key rate limiting. HMAC signature for webhooks. |
| Blacklist / number blocking | Prevent calls to blocked numbers (regulatory, fraud prevention). | LOW | Global + per-client blacklists. Check before originate. |
| Basic call routing (prefix-based) | Route B-leg to correct gateway based on destination prefix (130=mobile, 150=unicom, etc.). | MEDIUM | Prefix table lookup, gateway selection. Must handle overlapping prefixes (longest match). |

### Differentiators (Competitive Advantage)

Features that set BOS3000 apart from generic callback platforms.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Wastage analysis (A-through-B-fail / bridge-drop) | Most platforms only show CDRs. BOS3000 analyzes WHY calls fail: A answered but B unreachable, bridge connected but dropped in <3s. Directly maps to route quality and cost savings. | MEDIUM | Categorize: A-fail, B-fail-after-A-answer (wastage), short-duration (<6s = likely not real conversation). Dashboard with ASR/ACD per route, per gateway, per time period. |
| ASR/ACD quality metrics per gateway/route | Wholesale operators live and die by ASR (Answer Seizure Ratio) and ACD (Average Call Duration). Real-time visibility enables instant route switching. | MEDIUM | Calculate from CDRs. Alert when ASR drops below threshold. Historical trend charts. |
| A-leg round-robin routing | Distribute A-leg calls across multiple gateways to avoid single-gateway overload and maintain quality. | LOW | Weighted round-robin with health-aware failover. Track per-gateway success rates. |
| DID number management | Assign specific caller-ID numbers to clients. Manage DID pool, allocation, release. | MEDIUM | DID inventory table, assignment to clients, auto-rotation to avoid number flagging. |
| FreeSWITCH HA (dual hot-standby) | Zero downtime for the media layer. Critical for production but rare in smaller platforms. | HIGH | Health check + automatic failover. ESL connection pool across FS instances. Session recovery is hard -- focus on new-call routing to healthy node. |
| WebSocket real-time call push | Live call status on dashboard without polling. Better UX than competitors using polling or periodic refresh. | MEDIUM | Push call state transitions to connected admin/client dashboards. |
| Webhook delivery audit trail | Full visibility into webhook delivery attempts, failures, retries. Most platforms hide this. | LOW | webhook_deliveries table with attempt count, response codes, next retry time. Client can see delivery status. |
| Rate plan versioning | Change rates without affecting in-progress billing periods. Enable scheduled rate changes. | MEDIUM | Effective date on rate plans. In-progress calls use rate at call start time. |
| Multi-tier client hierarchy | Reseller model: admin creates resellers, resellers create sub-clients. | HIGH | Defer to v2. Adds complexity to every query (data isolation at 3 levels). |
| Automated number hygiene / HLR lookup | Validate numbers before dialing to reduce wastage on disconnected/invalid numbers. | MEDIUM | Defer to v2. Requires external HLR provider integration. Phase 1 uses prefix routing which accepts some mismatch. |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Inbound call routing / IVR | "We want a full phone system" | Fundamentally different architecture (requires registration, NAT traversal, queue management). Doubles scope. BOS3000 is outbound-only by design. | Keep scope pure outbound. Refer inbound needs to dedicated PBX/CCaaS. |
| Real-time transcription / AI call analysis | Trendy AI feature | Massive infrastructure cost (speech-to-text per call), storage, latency. Not core to callback billing platform. | Provide recording download API; let clients use their own transcription service. |
| Mobile app (iOS/Android) | "Our agents use phones" | Native apps double frontend effort, require app store management, push notification infra. Low value for B2B API platform. | Responsive web portal works on mobile browsers. API-first means clients build their own UX if needed. |
| OAuth / social login | "Modern auth" | B2B platform with API keys. Social login adds complexity (provider outages, token refresh) for zero value. API clients never use social login. | Email/password + API key. JWT for portal sessions. Simple, reliable. |
| Video calling | "Competitors have video" | 10x bandwidth/storage cost. Completely different media pipeline. Not the product. | Out of scope. Different product category entirely. |
| Real-time call monitoring (listen-in/whisper) | "Supervisors want to listen" | Requires media forking on FreeSWITCH, additional RTP streams, WebRTC gateway for browser playback. High complexity for niche use case in API callback context. | Provide call recordings. If monitoring needed later, it is a v3 feature with dedicated architecture. |
| Per-second billing granularity below 1 second | "We want millisecond billing" | Industry standard is 1-second or 6-second blocks. Sub-second creates rounding disputes and CDR storage overhead. | 1-second billing (already precise). Configurable minimum billing increment (e.g., 6s, 30s, 60s). |
| Automatic route optimization (AI-based) | "Auto-switch to best route" | Black-box routing decisions are dangerous in telecom. Operators need control. Unexpected route changes cause compliance issues. | Surface ASR/ACD data clearly. Alert on degradation. Let operators make routing decisions with good data. |

## Feature Dependencies

```
[API Callback Engine (A/B originate + bridge)]
    |
    +--requires--> [FreeSWITCH ESL Integration]
    +--requires--> [Gateway Management]
    +--requires--> [Prefix-based Routing]
    +--requires--> [Balance Management (prepaid)]
    +--requires--> [Concurrent Call Limits]
    |
    +--produces--> [CDR Records]
    |                 |
    |                 +--enables--> [Billing / Rating Engine]
    |                 +--enables--> [Wastage Analysis]
    |                 +--enables--> [ASR/ACD Metrics]
    |
    +--produces--> [Call State Events]
    |                 |
    |                 +--enables--> [Webhook Notifications]
    |                 +--enables--> [WebSocket Real-time Push]
    |
    +--triggers--> [Recording (post-bridge)]
                      |
                      +--requires--> [Object Storage (S3)]
                      +--requires--> [ffmpeg for merge]

[Admin Portal]
    +--requires--> [All backend APIs]
    +--requires--> [Auth system (email/password + roles)]

[Client Portal]
    +--requires--> [Client-scoped backend APIs]
    +--requires--> [API Key Management]

[DID Management]
    +--enhances--> [API Callback Engine] (caller-ID selection)
    +--independent of--> [Routing Engine]

[Blacklist]
    +--enhances--> [API Callback Engine] (pre-call check)

[FreeSWITCH HA]
    +--enhances--> [FreeSWITCH ESL Integration]
    +--independent of--> [all business logic]
```

### Dependency Notes

- **Callback Engine requires ESL + Gateway + Routing + Balance**: Cannot make a single call without all four. These must be built together in the first phase.
- **CDR enables Billing, Wastage, ASR/ACD**: CDR is the foundation for all analytics and billing. Build CDR generation in the call engine phase, analysis features can follow.
- **Recording requires bridge event**: Recording starts at CHANNEL_BRIDGE, so recording depends on the call engine being functional.
- **Portals require backend APIs**: Admin and Client portals are pure consumers of backend APIs. Backend first, frontend second.
- **FS HA is independent of business logic**: Can be added to any phase without changing business logic. Pure infrastructure concern.
- **Wastage Analysis enhances CDR**: Same data, different aggregation. Low marginal cost once CDRs exist.

## MVP Definition

### Launch With (v1)

Minimum viable product -- what is needed to onboard the first paying client.

- [x] API-initiated A/B callback with FreeSWITCH ESL (mock-first, then real) -- core product
- [x] Gateway management (CRUD + health status) -- need at least one gateway to route calls
- [x] Prefix-based B-leg routing + A-leg round-robin -- calls must reach correct carrier
- [x] Prepaid balance with atomic deduction + concurrent call limits -- prevent revenue leakage
- [x] CDR generation on call completion -- billing and reconciliation foundation
- [x] Per-minute/per-second billing with rate plans -- clients need to know what they pay
- [x] Call recording (A/B split + merge + S3 upload) -- regulatory and QA requirement
- [x] Webhook notifications with retry + DLQ -- clients need programmatic call status
- [x] Blacklist checking -- regulatory compliance
- [x] Admin portal (operations dashboard) -- operators must manage the platform
- [x] Client portal (self-service dashboard) -- clients must see their own data
- [x] API key authentication + dual-mode permissions (admin/client) -- security foundation

### Add After Validation (v1.x)

Features to add once the core is stable and first clients are onboarded.

- [ ] Wastage analysis dashboard (A-through-B-fail categorization) -- add once enough CDR data exists to make analysis meaningful
- [ ] ASR/ACD metrics per gateway/route with alerting -- needs traffic volume to be useful
- [ ] DID number management + caller-ID assignment -- first clients can use fixed caller-ID; pool management adds value at scale
- [ ] WebSocket real-time call push -- polling works initially; WebSocket improves UX
- [ ] Rate plan versioning with effective dates -- first clients get one rate plan; versioning needed when rates change
- [ ] Webhook delivery audit trail in client portal -- expose webhook_deliveries data to clients
- [ ] FreeSWITCH HA (dual hot-standby + health-aware failover) -- single FS works for initial load; HA before scaling

### Future Consideration (v2+)

Features to defer until product-market fit is established and client base grows.

- [ ] Multi-tier client hierarchy (reseller model) -- only needed when resellers exist; adds query complexity everywhere
- [ ] HLR lookup for number validation -- reduces wastage but requires external provider contract and per-lookup cost
- [ ] Automated alerting (ASR threshold alerts via email/SMS) -- manual monitoring sufficient initially
- [ ] Batch callback API (upload CSV of A/B pairs) -- niche use case; clients can loop API calls
- [ ] Call cost estimation API (pre-call price check) -- nice UX but not blocking

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| A/B Callback Engine | HIGH | HIGH | P1 |
| Gateway Management | HIGH | MEDIUM | P1 |
| Prefix-based Routing | HIGH | MEDIUM | P1 |
| Balance Management | HIGH | MEDIUM | P1 |
| Concurrent Call Limits | HIGH | LOW | P1 |
| CDR Generation | HIGH | MEDIUM | P1 |
| Billing / Rating | HIGH | HIGH | P1 |
| Call Recording | HIGH | HIGH | P1 |
| Webhook Notifications | HIGH | MEDIUM | P1 |
| Blacklist | MEDIUM | LOW | P1 |
| Admin Portal | HIGH | HIGH | P1 |
| Client Portal | HIGH | MEDIUM | P1 |
| API Key Auth + Permissions | HIGH | MEDIUM | P1 |
| Wastage Analysis | HIGH | MEDIUM | P2 |
| ASR/ACD Metrics | HIGH | MEDIUM | P2 |
| DID Management | MEDIUM | MEDIUM | P2 |
| WebSocket Push | MEDIUM | MEDIUM | P2 |
| Rate Plan Versioning | MEDIUM | MEDIUM | P2 |
| Webhook Audit Trail | LOW | LOW | P2 |
| FreeSWITCH HA | HIGH | HIGH | P2 |
| Multi-tier Hierarchy | MEDIUM | HIGH | P3 |
| HLR Lookup | MEDIUM | MEDIUM | P3 |
| Batch Callback API | LOW | LOW | P3 |
| Cost Estimation API | LOW | LOW | P3 |

**Priority key:**
- P1: Must have for launch (core call flow, billing, recording, portals)
- P2: Should have, add after first clients onboarded (analytics, HA, DID management)
- P3: Nice to have, future consideration (reseller model, HLR, batch operations)

## Competitor Feature Analysis

| Feature | Twilio (CPaaS) | Wholesale VoIP Platforms (ASTPP/MOR) | Knowlarity/DeepCall (India/CN) | BOS3000 Approach |
|---------|----------------|--------------------------------------|-------------------------------|------------------|
| Call initiation | REST API, TwiML | Web portal + API | Web portal + API | REST API primary, portal secondary |
| Billing | Per-minute, post-paid | Prepaid + postpaid, prefix-based | Prepaid packages | Prepaid, prefix-based, per-second/minute configurable |
| CDR | Real-time API + console | CSV export + portal | Portal download | API + portal + webhook delivery |
| Recording | Mono/dual channel, cloud stored | Server-side, manual download | Cloud stored | A/B split + merged, S3 stored, API accessible |
| Webhooks | Comprehensive (initiated/ringing/answered/completed) | Limited or none | Basic status callbacks | Full lifecycle events with retry + DLQ + audit trail |
| Wastage analysis | None (DIY from CDRs) | Basic ASR/ACD reports | None | Built-in: A-fail, B-fail, short-duration categorization |
| Routing | Programmable (developer builds it) | LCR + prefix routing | Fixed routes | Prefix-based B-leg + round-robin A-leg with health awareness |
| DID management | Full API + console | Portal-based | Assigned by provider | Portal + API, pool management |
| Self-service portal | Console (developer-focused) | Basic reseller portal | Client dashboard | Purpose-built admin + client portals |
| HA | Built-in (cloud) | Manual FS clustering | Cloud (managed) | Dual FS hot-standby with ESL health checks |

**BOS3000's edge:** Purpose-built for the A/B callback use case with wastage analysis as a first-class feature. Wholesale VoIP platforms are generic (handle inbound, outbound, PBX). Twilio is a toolkit, not a solution. BOS3000 is an opinionated solution for callback operators who care about route quality and cost optimization.

## Sources

- [Twilio Voice API Documentation](https://www.twilio.com/docs/voice/api)
- [Twilio Webhooks Guide](https://www.twilio.com/docs/usage/webhooks)
- [FreeSWITCH CDR Documentation](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Configuration/CDR/)
- [FreeSWITCH Billing Solutions](https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Miscellaneous/Billing-Solutions_13173290/)
- [Kolmisoft ASR/ACD Wiki](https://wiki.kolmisoft.com/index.php/ASR/ACD)
- [ASTPP SIP Trunking Reports](https://astppbilling.org/sip-trunking-solution-advanced-reports-augment-quality-of-service-standards)
- [CloudTalk Click-to-Call Providers Review](https://www.cloudtalk.io/blog/click-to-call-providers/)
- [Knowlarity Click-to-Call Features](https://www.knowlarity.com/voice/click-to-call)
- [DeepCall Click-to-Call API](https://deepcall.com/api/click-to-call)
- [Voxvalley Click-to-Call Solution](https://voxvalley.com/click-to-call-solution/)
- [Hookdeck - Twilio Webhooks Best Practices](https://hookdeck.com/webhooks/platforms/twilio-webhooks-features-and-best-practices-guide)
- [VoIP Billing and Routing Systems](https://www.voip-info.org/voip-billing-and-routing-system/)

---
*Feature research for: BOS3000 API Callback / Click-to-Call Platform*
*Researched: 2026-03-09*
