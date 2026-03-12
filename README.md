# BOS3000

Enterprise SIP callback platform built on [Encore.go](https://encore.dev) + [FreeSWITCH](https://freeswitch.org).

Automated A/B leg dialing, real-time billing, weighted gateway routing, compliance enforcement, and webhook event delivery — with admin dashboard and client self-service portal.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        BOS3000                              │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
│  │   Auth   │  │ Billing  │  │ Callback │  │Compliance │  │
│  │  users   │  │  rates   │  │  calls   │  │ blacklist │  │
│  │ api-keys │  │ balance  │  │  state   │  │  audit    │  │
│  └──────────┘  └──────────┘  └──────────┘  └───────────┘  │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
│  │ Routing  │  │Recording │  │ Webhook  │  │  Gateway  │  │
│  │ gateways │  │  merge   │  │ delivery │  │ admin SPA │  │
│  │   DIDs   │  │ storage  │  │   DLQ    │  │portal SPA │  │
│  └──────────┘  └──────────┘  └──────────┘  └───────────┘  │
│                                                             │
│  PostgreSQL ──── Redis ──── FreeSWITCH (ESL)               │
└─────────────────────────────────────────────────────────────┘
```

**8 services** · **75 API endpoints** · **2 React SPAs** · monorepo

## Tech Stack

| Layer | Stack |
|-------|-------|
| Backend | Go 1.25, Encore.go, PostgreSQL, Redis |
| Frontend | React 19, React Router 7, TanStack Query, shadcn/ui, TailwindCSS 4, Vite 7 |
| VoIP | FreeSWITCH ESL (percipia/eslgo) |
| Auth | JWT (session cookie for admin, bearer token for portal, API key for REST) |

## Services

### Auth (`auth/`)
User management, login, API key lifecycle, IP whitelist.

| Method | Path | Access | Description |
|--------|------|--------|-------------|
| POST | `/auth/admin/login` | public | Admin login → session cookie |
| POST | `/auth/client/login` | public | Client login → bearer token |
| GET | `/auth/me` | auth | Current user profile |
| GET | `/auth/admin/users` | admin | List users (paginated, search) |
| GET | `/auth/admin/users/:id` | admin | User detail |
| POST | `/auth/admin/users` | admin | Create client user |
| POST | `/auth/admin/users/:id/freeze` | admin | Freeze account |
| POST | `/auth/admin/users/:id/unfreeze` | admin | Unfreeze account |
| POST | `/auth/admin/reset-password` | private | Reset admin password (CLI) |
| PUT | `/auth/profile` | auth | Update own profile |
| POST | `/auth/profile/password` | auth | Change password |
| POST | `/auth/api-keys` | auth | Create API key |
| GET | `/auth/api-keys` | auth | List API keys |
| DELETE | `/auth/api-keys/:id` | auth | Revoke API key |
| POST | `/auth/api-keys/:id/ips` | auth | Add IP to whitelist |

### Billing (`billing/`)
Rate plans, balance pre-deduction, call finalization, transactions.

| Method | Path | Access | Description |
|--------|------|--------|-------------|
| POST | `/billing/rate-plans` | admin | Create rate plan |
| PUT | `/billing/rate-plans/:id` | admin | Update rate plan |
| GET | `/billing/rate-plans` | admin | List rate plans |
| POST | `/billing/resolve-rate` | private | Resolve rate (user → plan → prefix) |
| POST | `/billing/pre-deduct` | private | Pre-deduct balance (30 min estimate) |
| POST | `/billing/acquire-slot` | private | Acquire concurrent call slot |
| POST | `/billing/release-slot` | private | Release concurrent call slot |
| POST | `/billing/finalize` | private | Final cost reconciliation |
| POST | `/billing/accounts/:userId/topup` | admin | Add balance |
| POST | `/billing/accounts/:userId/deduct` | admin | Deduct balance |
| GET | `/billing/accounts/:userId` | auth | Account info (clients: own only) |
| GET | `/billing/accounts/:userId/transactions` | auth | Transaction history |

### Callback (`callback/`)
Core call state machine — initiates A/B legs via FreeSWITCH, tracks lifecycle.

| Method | Path | Access | Description |
|--------|------|--------|-------------|
| POST | `/callbacks` | auth | Initiate callback |
| GET | `/callbacks` | auth | List CDR (paginated) |
| GET | `/ws/calls` | ws+token | Real-time call events |

**Call lifecycle**: `pending` → `a_answered` → `b_dialing` → `b_answered` → `bridged` → `completed`

### Compliance (`compliance/`)
Blacklist, daily limits, audit trail.

| Method | Path | Access | Description |
|--------|------|--------|-------------|
| POST | `/compliance/blacklist` | auth | Add number to blacklist |
| DELETE | `/compliance/blacklist/:id` | auth | Remove entry |
| GET | `/compliance/blacklist` | auth | List blacklisted numbers |
| POST | `/compliance/check-blacklist` | private | Pre-call check |
| POST | `/compliance/check-daily-limit` | private | Pre-call check |
| GET | `/compliance/audit-logs` | admin | View audit logs |

### Routing (`routing/`)
Gateway selection (smooth weighted round-robin), DID management.

| Method | Path | Access | Description |
|--------|------|--------|-------------|
| POST | `/routing/pick-a-leg` | private | Select A-leg gateway (weighted RR) |
| POST | `/routing/pick-b-leg` | private | Select B-leg gateway (prefix match) |
| POST | `/routing/select-did` | private | Allocate DID |
| POST | `/routing/gateways` | admin | Create gateway |
| PUT | `/routing/gateways/:id` | admin | Update gateway |
| GET | `/routing/gateways` | admin | List gateways |
| POST | `/routing/gateways/:id/toggle` | admin | Enable/disable |
| POST | `/routing/did-import` | admin | Bulk import DIDs |
| GET | `/routing/dids` | admin | List DIDs |

### Webhook (`webhook/`)
Event delivery with exponential backoff, DLQ for failures.

| Method | Path | Access | Description |
|--------|------|--------|-------------|
| PUT | `/webhooks/config` | auth | Set webhook URL |
| GET | `/webhooks/deliveries` | auth | Delivery history |
| POST | `/webhooks/secret/reset` | auth | Rotate secret |
| GET | `/admin/webhooks/dlq` | admin | Dead-letter queue |
| POST | `/admin/webhooks/dlq/:id/retry` | admin | Retry failed delivery |

### Recording (`recording/`)
Call recording merge and download.

### Gateway (`gateway/`)
Serves admin and portal SPAs as embedded static files, plus version endpoint.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/version` | Running version |
| GET | `/admin/*` | Admin SPA |
| GET | `/portal/*` | Portal SPA |

## Frontend

### Admin Dashboard (`admin/` → port 5173)

| Route | Page |
|-------|------|
| `/login` | Admin login |
| `/` | Dashboard overview |
| `/customers` | Customer list & detail |
| `/gateways` | Gateway config & health |
| `/cdr` | Call detail records |
| `/wastage` | Wastage analysis |
| `/finance` | Rate plans & transactions |
| `/did` | DID management |
| `/compliance` | Blacklist & audit |
| `/settings` | System config |

### Client Portal (`portal/` → port 5174)

| Route | Page |
|-------|------|
| `/login` | Client login |
| `/` | Dashboard (balance, trends) |
| `/callback` | Initiate callback |
| `/cdr` | Call records |
| `/wastage` | Wastage breakdown |
| `/finance` | Balance & transactions |
| `/api-integration` | API keys, webhooks, docs |
| `/settings` | Profile & security |

## Quick Start

```bash
# Prerequisites: Go 1.25+, Node 20+, PostgreSQL, Encore CLI

# Start backend (auto-creates DBs, runs migrations)
encore run
# → API: http://localhost:4000
# → Dashboard: http://localhost:9400

# Start frontends (separate terminals)
cd admin && npm install && npm run dev    # http://localhost:5173
cd portal && npm install && npm run dev   # http://localhost:5174
```

**Default accounts** (dev seed data):

| Role | Email | Password |
|------|-------|----------|
| Admin | `admin@bos3000.local` | `admin123` |
| Client | `client@bos3000.local` | `client123` |

## Production Deployment

```bash
# 1. Build
bash scripts/build.sh v1.0.0

# 2. Upload & install on server
scp dist/bos3000-v1.0.0.tar.gz root@<server>:/tmp/
ssh root@<server>
tar xzf /tmp/bos3000-v1.0.0.tar.gz -C /tmp/bos3000-deploy
cd /tmp/bos3000-deploy
sudo bash deploy/install.sh --version v1.0.0 --eip <public-ip>

# 3. Start
sudo systemctl start bos3000
```

**Production admin**: `admin@localhost` / `changeme123` (change immediately)

**Reset forgotten password**:
```bash
bash /opt/bos3000/reset-admin-password.sh
```

See [`deploy/README.md`](deploy/README.md) for full deployment guide.

## Testing

```bash
# Backend
encore test ./...

# Frontend
cd admin && npm test
cd portal && npm test
```

## Project Structure

```
bos3000/
├── auth/              # Auth service + migrations
├── billing/           # Billing service + migrations
├── callback/          # Callback service + FreeSWITCH ESL
├── compliance/        # Compliance service + migrations
├── routing/           # Routing service + migrations
├── recording/         # Recording service + migrations
├── webhook/           # Webhook service + migrations
├── gateway/           # SPA gateway (embedded frontend)
├── pkg/errcode/       # Shared error codes
├── admin/             # Admin dashboard (React)
├── portal/            # Client portal (React)
├── scripts/           # Build & utility scripts
├── deploy/            # Production deployment
└── encore.app         # Encore config
```

## Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 4000 | TCP | API + embedded web UI |
| 5060 | UDP/TCP | SIP signaling |
| 5061 | TCP | SIP TLS |
| 8021 | TCP | FreeSWITCH ESL |
| 16384–32768 | UDP | RTP media |
