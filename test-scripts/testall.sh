#!/usr/bin/env bash
set -euo pipefail

BASE_URL="http://localhost:8080"
ADMIN_TOKEN="replace-with-admin-api-token"
ADMIN_AUTH_HEADER="X-Admin-Token: ${ADMIN_TOKEN}"

TENANT_ID="tenant-001"
CLIENT_ID="client-001"
CLIENT_SECRET="replace-with-client-secret"
REDIRECT_URI="http://localhost:3000/callback"
USERNAME="demo@example.com"
PASSWORD="changeme" # if password grant is enabled in your setup

# 1) Create tenant (admin)
curl -sS -X POST "$BASE_URL/admin/tenants" \
  -H "$ADMIN_AUTH_HEADER" -H "Content-Type: application/json" \
  -d "{\"id\":\"$TENANT_ID\",\"name\":\"Demo Tenant\"}" | jq .

# 2) Create client (admin)
curl -sS -X POST "$BASE_URL/admin/clients" \
  -H "$ADMIN_AUTH_HEADER" -H "Content-Type: application/json" \
  -d "{\"id\":\"$CLIENT_ID\",\"tenant_id\":\"$TENANT_ID\",\"redirect_uris\":[\"$REDIRECT_URI\"],\"secret\":\"$CLIENT_SECRET\"}" | jq .

# 3) Start authorization (authorization code)
AUTH_CODE=$(
  curl -sS -X POST "$BASE_URL/auth/authorize" \
    -H "Content-Type: application/json" \
    -d "{\"client_id\":\"$CLIENT_ID\",\"redirect_uri\":\"$REDIRECT_URI\",\"response_type\":\"code\",\"scope\":\"openid profile\",\"state\":\"xyz\"}" \
  | jq -r '.code'
)
echo "AUTH_CODE=$AUTH_CODE"

# 4) Grant consent for requested purposes (after login/authz context)
curl -sS -X POST "$BASE_URL/auth/consent" \
  -H "Content-Type: application/json" \
  -d '{"purposes":["login","profile"]}' | jq .

# 5) Exchange code for tokens
TOKENS=$(
  curl -sS -X POST "$BASE_URL/auth/token" \
    -H "Content-Type: application/json" \
    -d "{\"grant_type\":\"authorization_code\",\"code\":\"$AUTH_CODE\",\"client_id\":\"$CLIENT_ID\",\"client_secret\":\"$CLIENT_SECRET\",\"redirect_uri\":\"$REDIRECT_URI\"}"
)
echo "$TOKENS" | jq .

ACCESS_TOKEN=$(echo "$TOKENS" | jq -r '.access_token')
REFRESH_TOKEN=$(echo "$TOKENS" | jq -r '.refresh_token')

# 6) Call userinfo
curl -sS -X GET "$BASE_URL/auth/userinfo" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .

# 7) List sessions
curl -sS -X GET "$BASE_URL/auth/sessions" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .

# 8) Revoke specific consent
curl -sS -X POST "$BASE_URL/auth/consent/revoke" \
  -H "Content-Type: application/json" \
  -d '{"purposes":["profile"]}' | jq .

# 9) Revoke all consents
curl -sS -X POST "$BASE_URL/auth/consent/revoke-all" | jq .

# 10) Refresh token
curl -sS -X POST "$BASE_URL/auth/token" \
  -H "Content-Type: application/json" \
  -d "{\"grant_type\":\"refresh_token\",\"refresh_token\":\"$REFRESH_TOKEN\",\"client_id\":\"$CLIENT_ID\",\"client_secret\":\"$CLIENT_SECRET\"}" | jq .

# 11) Revoke tokens
curl -sS -X POST "$BASE_URL/auth/revoke" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"$ACCESS_TOKEN\",\"client_id\":\"$CLIENT_ID\"}" | jq .

# 12) Delete all consents
curl -sS -X DELETE "$BASE_URL/auth/consent" | jq .
