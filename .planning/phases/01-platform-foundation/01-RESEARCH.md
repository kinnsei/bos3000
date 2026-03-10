# Phase 1: 平台基础 - Research

**Researched:** 2026-03-10
**Domain:** Encore.go backend services (auth, billing, routing, compliance, infrastructure)
**Confidence:** HIGH

## Summary

Phase 1 builds the foundational platform services for BOS3000: authentication (dual-mode admin/client), billing engine (atomic balance with pre-deduction), routing engine (A-leg weighted round-robin, B-leg prefix matching), compliance (blacklist, audit, rate limiting), and infrastructure standards (error codes, docker-compose, prefix validation). All services are built on Encore.go with PostgreSQL (managed by Encore) and Redis (via Encore cache).

The Encore.go framework provides strong primitives for this phase: `//encore:authhandler` with structured params supports the dual-auth model (JWT Cookie for admin, JWT Bearer/API Key for client); `encore.dev/storage/sqldb` handles database migrations and queries including `SELECT FOR UPDATE` for atomic balance; `encore.dev/storage/cache` provides Redis-backed `IntKeyspace` for concurrent slot control and daily call limits; `encore.dev/beta/errs` gives structured error codes mapping directly to HTTP status codes.

**Primary recommendation:** Structure as 5 Encore services (`auth`, `billing`, `routing`, `compliance`, `gateway`) each with its own database, plus a shared `pkg/` for error codes and common types. Use a single auth handler that inspects multiple credential sources (cookie, header, query) and returns a unified `AuthData` containing role and user_id.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- A 路按 6 秒块计费（不足 6 秒按 6 秒计），B 路按 60 秒块计费（不足 60 秒按 60 秒计）
- 费率模板支持两种计费精度，管理员按客户选择
- 呼叫发起时按固定时长（30 分钟）预扣余额，通话结束后精确结算，多退少补
- 客户只看到一笔总费用（单价x时长），管理员后台可看 A/B 路分别的成本明细和毛利分析
- 费率模板支持两种模式：统一单价（所有目的地同价）和按前缀计价（不同运营商前缀不同价格），管理员按客户选择模式
- 错误响应包含：业务错误码 + 可读描述 + 处理建议，不暴露内部实现细节
- 业务错误码采用英文常量格式（如 INSUFFICIENT_BALANCE、BLACKLISTED_NUMBER、RATE_LIMIT_EXCEEDED）
- 错误信息仅英文，前端负责国际化
- API 响应结构遵循 Encore 默认规范：成功直接返回数据，失败返回 {code, message, details}
- 黑名单采用精确号码匹配（完整手机号），不支持前缀或正则
- 黑名单双层结构：先查全局黑名单（管理员维护，不可覆盖），再查客户级黑名单
- 日呼叫量限制按每客户统一限额（daily_limit 字段），管理员可按客户调整
- 审计日志保留 90 天，过期自动清理
- DID 外显号选取优先级：客户专属号码池 > 公共号码池随机选取
- B 路前缀表由管理员在网关管理界面手动维护
- A 路网关采用简单整数权重轮询（如 weight=3），按比例分配流量
- 网关容灾基于定时健康检查，连续 N 次失败标记不健康，路由时自动跳过，恢复后自动重新加入

### Claude's Discretion
- JWT Token 有效期和刷新策略
- API Key 的格式和生成方式
- 健康检查的具体间隔和失败阈值
- 预扣金额的具体计算公式
- 数据库 schema 的字段级设计
- 审计日志的清理机制实现方式

### Deferred Ideas (OUT OF SCOPE)
无
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| AUTH-01 | 管理员通过 JWT Cookie（HttpOnly, Secure）认证，数据范围全平台 | Encore authhandler with structured params supports `cookie:"session"` tag; golang-jwt/jwt/v5 for token creation/validation |
| AUTH-02 | 客户通过 JWT Token 或 API Key（IP 白名单校验）认证，数据严格隔离 | Single authhandler inspects header + query; API Key with prefix format + bcrypt/argon2 hash storage |
| AUTH-03 | 客户可管理 API Key（查看 prefix、重置密钥）和 IP 白名单 | CRUD APIs in auth service; store hashed keys, expose only prefix for display |
| BILL-01 | 呼叫发起前原子预扣余额（PG 行锁），余额不足返回 402 | PostgreSQL `SELECT FOR UPDATE` + `UPDATE` in single transaction via `sqldb.Exec` |
| BILL-02 | 并发槽位通过 Redis INCR 控制，超限返回 429 | Encore `cache.IntKeyspace` with `Increment()` for atomic counter |
| BILL-03 | 通话结束后精确计算实际费用，退还预扣差额，写入 transactions 流水 | Billing service finalize API with transaction table insert |
| BILL-04 | 费率优先级：用户级 a_leg_rate > 费率模板 rate_plans > B 路前缀费率 | Multi-tier rate resolution logic in billing service |
| BILL-05 | 管理员可创建/编辑费率模板，为客户指定费率模板或用户级费率 | Admin-only CRUD APIs with rate_plans + rate_plan_prefixes tables |
| ROUT-01 | A 路网关通过加权轮询选择，跳过不健康网关 | Weighted round-robin algorithm with health status filter |
| ROUT-02 | B 路网关通过被叫前 3 位精确前缀匹配，降级到同运营商其他网关 | Prefix lookup table + failover chain |
| ROUT-03 | 网关配置容灾关系（failover_gateway_id），主网关 DOWN 自动切备用 | Gateway table with failover_gateway_id FK; routing checks health then falls back |
| ROUT-04 | DID 号码池管理：导入、分配专属/公共池、自动选取外显号码 | did_numbers table with user_id nullable (null=public pool); selection logic |
| COMP-01 | 全局 + 每客户黑名单，呼叫前检查被叫号码 | blacklisted_numbers table with nullable user_id; exact match lookup |
| COMP-02 | 所有管理操作写入审计日志 | audit_logs table; middleware or explicit logging in admin endpoints |
| COMP-03 | 日呼叫量限制（daily_limit），超限返回 429 | Encore cache IntKeyspace with daily TTL + user daily_limit field |
| INFR-01 | 所有 Web 服务统一端口 12345 | Encore run with `--listen` flag or encore.app config |
| INFR-02 | docker-compose.dev.yml 一键启动 FreeSWITCH 开发环境 | Docker Compose with FreeSWITCH 1.10 image + entrypoint env injection |
| INFR-03 | 启动时校验 fs_gateways 前缀与 rate_plans 前缀一致性 | Service init function queries both tables, logs warnings on mismatches |
| INFR-04 | 错误码规范统一实现 | Shared error code constants using encore.dev/beta/errs package |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| encore.dev | v1.52.1 | Framework (services, APIs, DB, cache) | Project constraint - already in go.mod |
| encore.dev/storage/sqldb | (bundled) | PostgreSQL database management | Encore's native DB primitive with auto-provisioning |
| encore.dev/storage/cache | (bundled) | Redis-backed caching and counters | Encore's native cache primitive, type-safe IntKeyspace |
| encore.dev/beta/errs | (bundled) | Structured error codes | Maps to HTTP status codes, supports details and metadata |
| encore.dev/beta/auth | (bundled) | Authentication framework | Single auth handler with structured params |
| github.com/golang-jwt/jwt/v5 | v5.2+ | JWT token creation and validation | De facto standard Go JWT library, 12,500+ importers |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| golang.org/x/crypto/bcrypt | latest | Password hashing | Admin/client login password verification |
| crypto/rand (stdlib) | Go 1.25 | API key generation | Generating cryptographically secure random API keys |
| encoding/base64 (stdlib) | Go 1.25 | API key encoding | URL-safe encoding of random bytes for API keys |
| net (stdlib) | Go 1.25 | IP address parsing | IP whitelist validation for API key auth |
| crypto/sha256 (stdlib) | Go 1.25 | API key hash storage | Fast hash for API key lookup (not passwords) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| golang-jwt/jwt | Clerk/Auth0 | External dependency, unnecessary for self-managed JWT |
| bcrypt | argon2id | Argon2id is newer/stronger but bcrypt is sufficient and simpler for admin passwords |
| PostgreSQL row locks | Redis distributed locks | PG row locks are simpler and transactionally consistent for balance ops |

**Installation:**
```bash
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
```

## Architecture Patterns

### Recommended Project Structure
```
bos3000/
  encore.app
  go.mod
  auth/                    # AUTH service - authentication & user management
    auth.go                # Service struct + init
    handler.go             # //encore:authhandler (single handler, multi-method)
    login.go               # Login/register endpoints
    apikey.go              # API key CRUD endpoints
    migrations/
      1_create_users.up.sql
      2_create_api_keys.up.sql
  billing/                 # BILLING service - balance, rates, transactions
    billing.go             # Service struct + init
    balance.go             # Pre-deduct, finalize, refund
    rates.go               # Rate plan CRUD, rate resolution
    migrations/
      1_create_rate_plans.up.sql
      2_create_transactions.up.sql
  routing/                 # ROUTING service - gateway selection, DID management
    routing.go             # Service struct + init
    aleg.go                # A-leg weighted round-robin
    bleg.go                # B-leg prefix matching + failover
    did.go                 # DID pool management
    health.go              # Gateway health checker (cron)
    migrations/
      1_create_gateways.up.sql
      2_create_did_numbers.up.sql
  compliance/              # COMPLIANCE service - blacklist, audit, rate limiting
    compliance.go          # Service struct + init
    blacklist.go           # Blacklist check + CRUD
    audit.go               # Audit log writer + query
    ratelimit.go           # Daily call limit check
    migrations/
      1_create_blacklist.up.sql
      2_create_audit_logs.up.sql
  pkg/                     # Shared package (not a service)
    errcode/
      codes.go             # Business error code constants
    types/
      types.go             # Shared types (Money, PhoneNumber, etc.)
  docker-compose.dev.yml   # FreeSWITCH dev environment
```

### Pattern 1: Single Auth Handler with Multi-Method Dispatch
**What:** One `//encore:authhandler` that checks cookie, then bearer token, then API key in order.
**When to use:** Always -- Encore supports exactly one auth handler per app.
**Example:**
```go
// Source: https://encore.dev/docs/go/develop/auth
type AuthParams struct {
    SessionCookie *http.Cookie `cookie:"session"`
    Authorization string       `header:"Authorization"`
    APIKey        string       `query:"api_key"`
}

type AuthData struct {
    UserID   int64  `json:"user_id"`
    Role     string `json:"role"` // "admin" or "client"
    Username string `json:"username"`
}

//encore:authhandler
func (s *Service) AuthHandler(ctx context.Context, p *AuthParams) (auth.UID, *AuthData, error) {
    // 1. Check session cookie (admin JWT in HttpOnly cookie)
    if p.SessionCookie != nil {
        return s.validateSessionCookie(ctx, p.SessionCookie.Value)
    }
    // 2. Check Authorization header (client JWT Bearer token)
    if strings.HasPrefix(p.Authorization, "Bearer ") {
        return s.validateBearerToken(ctx, strings.TrimPrefix(p.Authorization, "Bearer "))
    }
    // 3. Check API key (query param or could also be in Authorization)
    if p.APIKey != "" {
        return s.validateAPIKey(ctx, p.APIKey)
    }
    return "", nil, &errs.Error{Code: errs.Unauthenticated, Message: "no credentials provided"}
}
```

### Pattern 2: Atomic Balance Pre-Deduction with PG Row Lock
**What:** Use `SELECT ... FOR UPDATE` to lock the user's balance row, check sufficiency, deduct, all in one transaction.
**When to use:** Every call initiation (BILL-01).
**Example:**
```go
// Source: PostgreSQL docs + Encore sqldb patterns
func (s *Service) PreDeduct(ctx context.Context, p *PreDeductParams) (*PreDeductResponse, error) {
    tx, err := s.db.Begin(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    var balance int64 // stored in cents/fen
    err = tx.QueryRow(ctx,
        `SELECT balance FROM users WHERE id = $1 FOR UPDATE`, p.UserID,
    ).Scan(&balance)
    if err != nil {
        return nil, err
    }

    // Pre-deduct: 30min * rate
    preDeductAmount := calculatePreDeduct(p.Rate, 30*60)
    if balance < preDeductAmount {
        return nil, &errs.Error{Code: errs.FailedPrecondition, Message: "INSUFFICIENT_BALANCE"}
        // Note: errs.FailedPrecondition maps to 400; for 402 we need custom HTTP status
    }

    _, err = tx.Exec(ctx,
        `UPDATE users SET balance = balance - $1 WHERE id = $2`,
        preDeductAmount, p.UserID,
    )
    if err != nil {
        return nil, err
    }

    // Record transaction
    _, err = tx.Exec(ctx,
        `INSERT INTO transactions (user_id, type, amount, balance_after, reference_id, created_at)
         VALUES ($1, 'pre_deduct', $2, $3, $4, NOW())`,
        p.UserID, preDeductAmount, balance-preDeductAmount, p.CallID,
    )
    if err != nil {
        return nil, err
    }

    if err := tx.Commit(); err != nil {
        return nil, err
    }
    return &PreDeductResponse{Amount: preDeductAmount, TxID: txID}, nil
}
```

### Pattern 3: Weighted Round-Robin for A-Leg Gateway Selection
**What:** Maintain in-memory weighted counter for gateway selection, skip unhealthy.
**When to use:** Every A-leg routing decision (ROUT-01).
**Example:**
```go
type WeightedGateway struct {
    ID              int64
    Name            string
    Weight          int
    CurrentWeight   int
    Healthy         bool
}

func (s *Service) PickALegGateway(ctx context.Context) (*Gateway, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    healthyGateways := make([]*WeightedGateway, 0)
    totalWeight := 0
    for _, gw := range s.aLegGateways {
        if gw.Healthy {
            healthyGateways = append(healthyGateways, gw)
            totalWeight += gw.Weight
        }
    }
    if len(healthyGateways) == 0 {
        return nil, &errs.Error{Code: errs.Unavailable, Message: "NO_HEALTHY_GATEWAY"}
    }

    // Smooth weighted round-robin (nginx-style)
    var best *WeightedGateway
    for _, gw := range healthyGateways {
        gw.CurrentWeight += gw.Weight
        if best == nil || gw.CurrentWeight > best.CurrentWeight {
            best = gw
        }
    }
    best.CurrentWeight -= totalWeight
    return toGateway(best), nil
}
```

### Pattern 4: B-Leg Prefix Matching with Failover
**What:** Match callee number prefix (first 3 digits) to gateway, with failover chain.
**When to use:** Every B-leg routing decision (ROUT-02, ROUT-03).
**Example:**
```go
func (s *Service) PickBLegGateway(ctx context.Context, calledNumber string) (*Gateway, error) {
    prefix := calledNumber[:3]

    // Query gateways for this prefix, ordered by priority
    rows, err := s.db.Query(ctx,
        `SELECT g.id, g.name, g.healthy, g.failover_gateway_id
         FROM fs_gateways g
         JOIN gateway_prefixes gp ON g.id = gp.gateway_id
         WHERE gp.prefix = $1
         ORDER BY gp.priority ASC`, prefix,
    )
    // ... iterate, pick first healthy, follow failover chain if needed
}
```

### Anti-Patterns to Avoid
- **Multiple auth handlers:** Encore allows exactly ONE auth handler. Do NOT try to create separate admin/client handlers -- use one handler with role-based dispatch.
- **Storing API keys in plaintext:** Always hash API keys before storage. Store prefix (first 8 chars) separately for display.
- **Using `defer tx.Rollback()` without checking commit:** This is actually correct -- `Rollback()` after `Commit()` is a no-op. But never forget to call `Commit()`.
- **Floating-point money:** Store all monetary values as `int64` in smallest unit (fen/cents). Never use `float64` for money.
- **Global variables for mutable state in routing:** Use service struct fields with mutex protection for weighted round-robin counters.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JWT token creation/validation | Custom JWT parser | `github.com/golang-jwt/jwt/v5` | Edge cases in JWT spec (alg confusion, clock skew, claim validation) |
| Password hashing | Custom hash scheme | `golang.org/x/crypto/bcrypt` | Bcrypt handles salt, work factor, timing-safe comparison |
| HTTP error codes | Manual HTTP status setting | `encore.dev/beta/errs` with ErrCode | Encore maps codes to HTTP status automatically |
| Database migrations | Manual SQL execution | `encore.dev/storage/sqldb` with migrations dir | Encore auto-runs migrations, handles versioning |
| Redis operations | Raw Redis client | `encore.dev/storage/cache` | Type-safe, auto-provisioned, no connection management |
| Cron cleanup jobs | Manual goroutine timers | `encore.dev/cron` | Reliable scheduling, only runs in deployed environments |

**Key insight:** Encore provides infrastructure primitives as code. Fighting the framework (e.g., using raw Redis client instead of cache package) loses type safety and auto-provisioning benefits.

## Common Pitfalls

### Pitfall 1: HTTP 402 Not in Encore's Error Codes
**What goes wrong:** `errs.FailedPrecondition` maps to HTTP 400, not 402 (Payment Required). There's no built-in 402 code in Encore's errs package.
**Why it happens:** The errs package follows gRPC error code conventions which don't include 402.
**How to avoid:** Use `errs.FailedPrecondition` with a clear business error code like `INSUFFICIENT_BALANCE` in the message. The frontend/client can check the business code rather than HTTP status. Alternatively, use the `Details` field to communicate the specific situation.
**Warning signs:** Requirements specifying HTTP 402 -- acknowledge this limitation in API docs.

### Pitfall 2: Single Auth Handler Complexity
**What goes wrong:** The single auth handler becomes a complex dispatcher with hard-to-test branching logic.
**Why it happens:** Encore mandates one auth handler, but the app needs cookie auth (admin), bearer token (client), and API key (client).
**How to avoid:** Extract each auth method into a separate private function (`validateSessionCookie`, `validateBearerToken`, `validateAPIKey`). Test each independently. The handler itself is just dispatch logic.
**Warning signs:** Auth handler function exceeding 50 lines.

### Pitfall 3: Concurrent Balance Race Conditions
**What goes wrong:** Two simultaneous calls for the same user both pass the balance check and cause negative balance.
**Why it happens:** Without `SELECT FOR UPDATE`, two transactions read the same balance before either writes.
**How to avoid:** Always use `SELECT ... FOR UPDATE` within a transaction for balance operations. This serializes concurrent access to the same user's row.
**Warning signs:** Balance going negative in production, duplicate pre-deduction for same call.

### Pitfall 4: Weighted Round-Robin State Loss on Restart
**What goes wrong:** Service restart resets round-robin counters, causing temporary traffic imbalance.
**Why it happens:** In-memory state is ephemeral.
**How to avoid:** Accept this as tolerable -- counters converge quickly. Do NOT persist RR state to database (overhead not worth it for a few seconds of imbalance).
**Warning signs:** Trying to store round-robin state in PostgreSQL per-request.

### Pitfall 5: API Key Timing Attack
**What goes wrong:** Comparing API keys with `==` leaks information about key prefix via timing.
**Why it happens:** String comparison short-circuits on first mismatched byte.
**How to avoid:** Hash the API key with SHA-256, then look up by hash. The database lookup is constant-time by nature (hash match or miss). Do NOT compare raw keys in application code.
**Warning signs:** Using `if apiKey == storedKey` anywhere.

### Pitfall 6: Audit Log Write Blocking Request
**What goes wrong:** Synchronous audit log insert slows down admin API responses.
**Why it happens:** Writing to audit_logs table in the same transaction as the business operation.
**How to avoid:** Write audit logs asynchronously via Encore Pub/Sub or in a separate goroutine. Audit log loss is tolerable for a few edge cases; request latency is not.
**Warning signs:** Admin API endpoints consistently slower than expected.

## Code Examples

### Business Error Codes (pkg/errcode/codes.go)
```go
// Source: CONTEXT.md locked decision on error format
package errcode

import "encore.dev/beta/errs"

// Business error codes as constants
const (
    InsufficientBalance = "INSUFFICIENT_BALANCE"
    BlacklistedNumber   = "BLACKLISTED_NUMBER"
    RateLimitExceeded   = "RATE_LIMIT_EXCEEDED"
    NoHealthyGateway    = "NO_HEALTHY_GATEWAY"
    InvalidCredentials  = "INVALID_CREDENTIALS"
    APIKeyRevoked       = "API_KEY_REVOKED"
    IPNotWhitelisted    = "IP_NOT_WHITELISTED"
    PrefixNotFound      = "PREFIX_NOT_FOUND"
    DailyLimitExceeded  = "DAILY_LIMIT_EXCEEDED"
)

// NewError creates a structured error with business code in details
func NewError(code errs.ErrCode, bizCode string, message string) error {
    return &errs.Error{
        Code:    code,
        Message: message,
        Details: ErrDetails{BizCode: bizCode, Suggestion: suggestFor(bizCode)},
    }
}

type ErrDetails struct {
    BizCode    string `json:"biz_code"`
    Suggestion string `json:"suggestion"`
}
```

### Daily Call Limit Check (compliance/ratelimit.go)
```go
// Source: Encore cache docs + CONTEXT.md
var complianceCache = cache.NewCluster("compliance", cache.ClusterConfig{
    EvictionPolicy: cache.AllKeysLRU,
})

var dailyCalls = cache.NewIntKeyspace[DailyCallKey](complianceCache, cache.KeyspaceConfig{
    KeyPattern:    "daily-calls/:UserID/:Date",
    DefaultExpiry: cache.ExpireIn(25 * time.Hour), // slightly longer than a day
})

type DailyCallKey struct {
    UserID int64
    Date   string // "2026-03-10"
}

func (s *Service) CheckDailyLimit(ctx context.Context, userID int64, dailyLimit int64) error {
    key := DailyCallKey{UserID: userID, Date: time.Now().Format("2006-01-02")}
    count, err := dailyCalls.Increment(ctx, key, 1)
    if err != nil {
        // Fail open -- allow call if Redis is down
        return nil
    }
    if count > dailyLimit {
        // Decrement back since we incremented optimistically
        dailyCalls.Increment(ctx, key, -1)
        return errcode.NewError(errs.ResourceExhausted, errcode.DailyLimitExceeded,
            "daily call limit exceeded")
    }
    return nil
}
```

### API Key Generation and Storage
```go
// Source: crypto/rand best practices
func GenerateAPIKey() (displayPrefix string, fullKey string, hashedKey string, err error) {
    // Generate 32 bytes of random data
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", "", "", err
    }

    prefix := "bos_"
    key := prefix + base64.URLEncoding.EncodeToString(b)

    // Hash for storage (SHA-256 is fine for API keys -- not passwords)
    hash := sha256.Sum256([]byte(key))
    hashedHex := hex.EncodeToString(hash[:])

    return key[:12], key, hashedHex, nil  // prefix for display, full key for user, hash for DB
}
```

### Billing Calculation (6s / 60s blocks)
```go
// Source: CONTEXT.md locked decision
func CalculateCost(durationSec int64, ratePerMin int64, blockSec int64) int64 {
    if durationSec <= 0 {
        return 0
    }
    // Round up to nearest block
    blocks := (durationSec + blockSec - 1) / blockSec
    actualSeconds := blocks * blockSec
    // rate is per minute, convert to per second then multiply
    // Use integer math: (ratePerMin * actualSeconds + 59) / 60 for ceiling
    cost := (ratePerMin * actualSeconds) / 60
    return cost
}

// A-leg: CalculateCost(duration, aRate, 6)   -- 6-second blocks
// B-leg: CalculateCost(duration, bRate, 60)  -- 60-second blocks
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| github.com/dgrijalva/jwt-go | github.com/golang-jwt/jwt/v5 | 2021-2023 | Must use v5, old package abandoned |
| encore.dev/beta/errs manual codes | encore.dev/beta/errs with Details struct | v1.40+ | Can attach typed details to errors |
| Raw Redis client in Encore | encore.dev/storage/cache package | v1.20+ | Type-safe, auto-provisioned Redis |
| bcrypt only | bcrypt (passwords) + sha256 (API keys) | Best practice | Different threat models, different algorithms |

**Deprecated/outdated:**
- `github.com/dgrijalva/jwt-go`: Abandoned, use `github.com/golang-jwt/jwt/v5`
- Manual Redis connection in Encore: Use `encore.dev/storage/cache` instead

## Open Questions

1. **HTTP 402 status code**
   - What we know: Encore's errs package doesn't have a direct 402 mapping. `errs.FailedPrecondition` returns 400.
   - What's unclear: Whether Encore supports custom HTTP status codes or if we must accept 400 for insufficient balance.
   - Recommendation: Use `errs.FailedPrecondition` with business code `INSUFFICIENT_BALANCE` in Details. Document that clients should check `biz_code`, not HTTP status. This is acceptable since the CONTEXT.md says "API 响应结构遵循 Encore 默认规范".

2. **Encore.go port configuration (INFR-01)**
   - What we know: Encore run starts on a default port. The requirement says all services on port 12345.
   - What's unclear: Exact CLI flag or config for custom port in encore.app.
   - Recommendation: Use `encore run --listen :12345` for local dev. For production, Encore Cloud handles port binding.

3. **Audit log async write mechanism**
   - What we know: Encore has Pub/Sub primitives that could handle async writes.
   - What's unclear: Whether the overhead of Pub/Sub is justified for audit logs vs. simple goroutine.
   - Recommendation: Use Encore Pub/Sub topic for audit events -- it provides guaranteed delivery and is the framework-native approach. Create an `audit_events` topic and a subscription in the compliance service.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + encore test |
| Config file | none -- Encore uses standard Go test conventions |
| Quick run command | `encore test ./auth/... ./billing/... ./routing/... ./compliance/...` |
| Full suite command | `encore test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AUTH-01 | Admin JWT cookie auth validates correctly | unit | `encore test ./auth/... -run TestAdminCookieAuth -x` | No -- Wave 0 |
| AUTH-02 | Client JWT/API Key auth with IP whitelist | unit | `encore test ./auth/... -run TestClientAuth -x` | No -- Wave 0 |
| AUTH-03 | API Key CRUD operations | unit | `encore test ./auth/... -run TestAPIKey -x` | No -- Wave 0 |
| BILL-01 | Atomic pre-deduction with row lock | unit | `encore test ./billing/... -run TestPreDeduct -x` | No -- Wave 0 |
| BILL-02 | Concurrent slot control via Redis | unit | `encore test ./billing/... -run TestConcurrentSlot -x` | No -- Wave 0 |
| BILL-03 | Finalize billing with refund | unit | `encore test ./billing/... -run TestFinalize -x` | No -- Wave 0 |
| BILL-04 | Rate priority resolution | unit | `encore test ./billing/... -run TestRatePriority -x` | No -- Wave 0 |
| BILL-05 | Rate plan CRUD | unit | `encore test ./billing/... -run TestRatePlan -x` | No -- Wave 0 |
| ROUT-01 | A-leg weighted round-robin | unit | `encore test ./routing/... -run TestALegRoundRobin -x` | No -- Wave 0 |
| ROUT-02 | B-leg prefix matching | unit | `encore test ./routing/... -run TestBLegPrefix -x` | No -- Wave 0 |
| ROUT-03 | Gateway failover chain | unit | `encore test ./routing/... -run TestGatewayFailover -x` | No -- Wave 0 |
| ROUT-04 | DID pool selection | unit | `encore test ./routing/... -run TestDIDSelection -x` | No -- Wave 0 |
| COMP-01 | Blacklist check (global + client) | unit | `encore test ./compliance/... -run TestBlacklist -x` | No -- Wave 0 |
| COMP-02 | Audit log write | unit | `encore test ./compliance/... -run TestAuditLog -x` | No -- Wave 0 |
| COMP-03 | Daily call limit | unit | `encore test ./compliance/... -run TestDailyLimit -x` | No -- Wave 0 |
| INFR-03 | Prefix consistency validation | unit | `encore test ./routing/... -run TestPrefixConsistency -x` | No -- Wave 0 |
| INFR-04 | Error code constants and formatting | unit | `encore test ./pkg/errcode/... -run TestErrorCodes -x` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `encore test ./auth/... ./billing/... ./routing/... ./compliance/... ./pkg/...`
- **Per wave merge:** `encore test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `auth/auth_test.go` -- covers AUTH-01, AUTH-02, AUTH-03
- [ ] `billing/billing_test.go` -- covers BILL-01 through BILL-05
- [ ] `routing/routing_test.go` -- covers ROUT-01 through ROUT-04
- [ ] `compliance/compliance_test.go` -- covers COMP-01, COMP-02, COMP-03
- [ ] `pkg/errcode/codes_test.go` -- covers INFR-04
- [ ] Database migrations for all services -- required before tests can run

## Sources

### Primary (HIGH confidence)
- [Encore.go Auth Docs](https://encore.dev/docs/go/develop/auth) - auth handler patterns, structured params, cookie/header/query tags
- [Encore.go Caching Docs](https://encore.dev/docs/go/primitives/caching) - IntKeyspace, Increment, rate limiting patterns
- [Encore.go Error Docs](https://encore.dev/docs/go/primitives/api-errors) - errs package, error codes, Details struct
- [Encore.go App Structure](https://encore.dev/docs/go/primitives/app-structure) - service definition, monorepo patterns
- [Encore.go SQL Databases](https://encore.dev/docs/go/primitives/databases) - sqldb.NewDatabase, migrations, transactions

### Secondary (MEDIUM confidence)
- [golang-jwt/jwt v5](https://pkg.go.dev/github.com/golang-jwt/jwt/v5) - JWT library API, validation options
- [PostgreSQL Explicit Locking](https://www.postgresql.org/docs/current/explicit-locking.html) - SELECT FOR UPDATE semantics
- [Go API Key Authentication](https://oneuptime.com/blog/post/2026-01-30-go-api-key-authentication/view) - API key generation patterns

### Tertiary (LOW confidence)
- Port configuration for `encore run` -- inferred from CLI help, needs validation

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Encore.go primitives are well-documented, golang-jwt/jwt is de facto standard
- Architecture: HIGH - Follows Encore.go monorepo service patterns exactly
- Pitfalls: HIGH - Balance race conditions and JWT security are well-understood domains
- Auth handler: HIGH - Verified structured params with cookie/header/query tags from official docs
- Port config (INFR-01): LOW - Need to verify exact CLI flag or encore.app setting

**Research date:** 2026-03-10
**Valid until:** 2026-04-10 (stable ecosystem, 30-day validity)
