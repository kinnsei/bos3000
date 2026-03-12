#!/usr/bin/env bash
set -euo pipefail

# BOS3000 Admin Password Reset
# Run on the server where BOS3000 is running.

API_URL="${BOS3000_API_URL:-http://localhost:4000}"

echo "=== BOS3000 Admin Password Reset ==="
echo ""

read -rp "Admin email: " EMAIL
if [[ -z "$EMAIL" ]]; then
  echo "Error: email is required"
  exit 1
fi

read -rsp "New password (min 8 chars): " PASSWORD
echo ""
if [[ ${#PASSWORD} -lt 8 ]]; then
  echo "Error: password must be at least 8 characters"
  exit 1
fi

read -rsp "Confirm password: " PASSWORD2
echo ""
if [[ "$PASSWORD" != "$PASSWORD2" ]]; then
  echo "Error: passwords do not match"
  exit 1
fi

echo ""
echo "Resetting password for $EMAIL ..."

RESPONSE=$(curl -s -w "\n%{http_code}" \
  --noproxy localhost \
  -X POST "${API_URL}/auth/admin/reset-password" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"new_password\":\"${PASSWORD}\"}")

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [[ "$HTTP_CODE" == "200" ]]; then
  echo "Password reset successfully."
else
  echo "Failed (HTTP $HTTP_CODE): $BODY"
  exit 1
fi
