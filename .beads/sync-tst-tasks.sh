#!/bin/bash
# Sync TST test plan tasks to beads-rust
# Run: bash .beads/sync-tst-tasks.sh

set -e

# 1. Root epic
ROOT=$(br create "[TST] Test Coverage Expansion" -t epic -p 1 --silent)
echo "Root epic: $ROOT"

# 2. Phase epics
P1=$(br create "[TST.1] Backend Test Gap Coverage" -t epic -p 1 --parent "$ROOT" --silent)
P2=$(br create "[TST.2] Frontend Shared Code Tests" -t epic -p 1 --parent "$ROOT" --silent)
P3=$(br create "[TST.3] Admin Component Tests" -t epic -p 2 --parent "$ROOT" --silent)
P4=$(br create "[TST.4] Portal Component Tests" -t epic -p 2 --parent "$ROOT" --silent)
P5=$(br create "[TST.5] Cross-Cutting Tests" -t epic -p 3 --parent "$ROOT" --silent)
echo "Phase epics: P1=$P1 P2=$P2 P3=$P3 P4=$P4 P5=$P5"

# 3. Phase 1 tasks (all done in Wave A)
T11=$(br create "TST.1.1 Recording: API + worker tests" -t task -p 1 --parent "$P1" -s closed -d "12 tests: recording/api_test.go. Invalid type, empty callID, unauth, not found, client isolation, not ready." --silent)
T12=$(br create "TST.1.2 Webhook: DLQ + deliver tests" -t task -p 1 --parent "$P1" -s closed -d "6 tests: webhook/dlq_test.go. Admin-only retry, wrong status, not found, timeout, invalid URL, retry intervals." --silent)
T13=$(br create "TST.1.3 Gateway: SPA handler tests" -t task -p 1 --parent "$P1" -s closed -d "12 tests: gateway/gateway_test.go. Root serve, SPA fallback, cache headers, path stripping." --silent)
T14=$(br create "TST.1.4 Auth: edge case tests" -t task -p 1 --parent "$P1" -s closed -d "5 tests: auth/auth_test.go (extended). Dup email, freeze idempotency, non-admin freeze, pagination, wrong password." --silent)
T15=$(br create "TST.1.5 Billing: edge case tests" -t task -p 1 --parent "$P1" -s closed -d "6 tests: billing/billing_test.go (extended). Negative topup, dup plan name, empty txns, missing account, over-deduct, zero rates." --silent)
echo "P1 tasks: $T11 $T12 $T13 $T14 $T15"

# 4. Phase 2 tasks (done in Wave A)
T21=$(br create "TST.2.1 Admin API client tests" -t task -p 1 --parent "$P2" -s closed -d "14 tests: admin/src/lib/api/client.test.ts. Request helper, qs, auth, billing, callback, routing." --silent)
T22=$(br create "TST.2.2 Portal API client tests" -t task -p 1 --parent "$P2" -s closed -d "12 tests: portal/src/lib/api/client.test.ts. Token mgmt, bearer injection, error handling, auth, callback." --silent)
T23=$(br create "TST.2.3 CDR utils + export tests" -t task -p 1 --parent "$P2" -s closed -d "7 tests: portal/src/lib/export.test.ts. CSV export, escaping, transforms, XLSX, CDR columns." --silent)
T24=$(br create "TST.2.4 useCallWs hook tests" -t task -p 1 --parent "$P2" -s closed -d "10 tests: portal/src/hooks/use-call-ws.test.ts. WS lifecycle, reconnect backoff, auth failure, disconnect." --silent)
T25=$(br create "TST.2.5 Error/version parity" -t task -p 2 --parent "$P2" -d "4 tests: extend existing error tests + version display tests." --silent)
echo "P2 tasks: $T21 $T22 $T23 $T24 $T25"

# 5. Phase 3 tasks (Wave B+C - pending)
T31=$(br create "TST.3.1 Customers (table, create, balance, detail)" -t task -p 1 --parent "$P3" -d "14 tests: admin customer components." --silent)
T32=$(br create "TST.3.2 Gateways (table, config, health, test-call)" -t task -p 1 --parent "$P3" -d "10 tests: admin gateway components." --silent)
T33=$(br create "TST.3.3 CDR + live calls" -t task -p 1 --parent "$P3" -d "10 tests: admin CDR and live call components." --silent)
T34=$(br create "TST.3.4 Finance (transactions, rate plans, profit)" -t task -p 2 --parent "$P3" -d "8 tests: admin finance components." --silent)
T35=$(br create "TST.3.5 Compliance (blacklist, audit log)" -t task -p 2 --parent "$P3" -d "8 tests: admin compliance components." --silent)
T36=$(br create "TST.3.6 Dashboard, DID, wastage, ops, settings" -t task -p 3 --parent "$P3" -d "14 tests: admin remaining components." --silent)
echo "P3 tasks: $T31 $T32 $T33 $T34 $T35 $T36"

# 6. Phase 4 tasks (Wave B+C - pending)
T41=$(br create "TST.4.1 Callback form + active calls + call card" -t task -p 1 --parent "$P4" -d "12 tests: portal callback components." --silent)
T42=$(br create "TST.4.2 CDR table + detail + export + recording" -t task -p 1 --parent "$P4" -d "10 tests: portal CDR components." --silent)
T43=$(br create "TST.4.3 Finance (balance, transactions, rate query)" -t task -p 2 --parent "$P4" -d "8 tests: portal finance components." --silent)
T44=$(br create "TST.4.4 Settings, API integration, wastage, dashboard" -t task -p 3 --parent "$P4" -d "16 tests: portal remaining components." --silent)
echo "P4 tasks: $T41 $T42 $T43 $T44"

# 7. Phase 5 tasks (Wave D - pending)
T51=$(br create "TST.5.1 Race/concurrency (Go -race flag)" -t task -p 2 --parent "$P5" -d "6 tests: callback/race_test.go, billing/concurrent_test.go." --silent)
T52=$(br create "TST.5.2 Router/navigation tests" -t task -p 2 --parent "$P5" -d "8 tests: */src/app/router.test.tsx." --silent)
T53=$(br create "TST.5.3 WS status banner" -t task -p 3 --parent "$P5" -d "4 tests: admin/src/components/shared/ws-status-banner.test.tsx." --silent)
echo "P5 tasks: $T51 $T52 $T53"

# 8. Wire dependencies
br dep add "$T24" "$T22" 2>/dev/null || true  # TST.2.4 depends on TST.2.2
br dep add "$T31" "$T21" 2>/dev/null || true  # TST.3.x depends on TST.2.1
br dep add "$T32" "$T21" 2>/dev/null || true
br dep add "$T33" "$T21" 2>/dev/null || true
br dep add "$T41" "$T22" 2>/dev/null || true  # TST.4.x depends on TST.2.2
br dep add "$T42" "$T22" 2>/dev/null || true
br dep add "$T51" "$P1" 2>/dev/null || true   # TST.5.x depends on Phase 1+2
br dep add "$T51" "$P2" 2>/dev/null || true

echo "Done! All 23 tasks + 5 phase epics + 1 root epic created."
br stats
