#!/usr/bin/env bash
set -euo pipefail

BASE_URL="http://localhost:8080"
ADMIN_TOKEN="replace-with-admin-api-token"
ACCESS_TOKEN="replace-with-valid-access-token"   # must be a bearer token for protected routes
CONSENT_BODY='{"purposes":["login"]}'
CLASS_HITS=20   # how many times to hit each endpoint

log_hit() { printf "%s | class=%s | req=%s | status=%s\n" "$1" "$2" "$3" "$4"; }

# Auth class: /auth/authorize
for i in $(seq 1 $CLASS_HITS); do
  status=$(curl -w "%{http_code}" -o /tmp/rl-auth.out -sS \
    -X POST "$BASE_URL/auth/authorize" \
    -H "Content-Type: application/json" \
    -d '{"client_id":"client-001","redirect_uri":"http://localhost:3000/callback","response_type":"code","scope":"openid","state":"s"}')
  log_hit "auth" "$i" "$status"
done

# Read class: /auth/userinfo
for i in $(seq 1 $CLASS_HITS); do
  status=$(curl -w "%{http_code}" -o /tmp/rl-read.out -sS \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    "$BASE_URL/auth/userinfo")
  log_hit "read" "$i" "$status"
done

# Sensitive class: /auth/consent
for i in $(seq 1 $CLASS_HITS); do
  status=$(curl -w "%{http_code}" -o /tmp/rl-sensitive.out -sS \
    -X POST "$BASE_URL/auth/consent" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$CONSENT_BODY")
  log_hit "sensitive" "$i" "$status"
done

# Write class: /admin/tenants (admin token required)
for i in $(seq 1 $CLASS_HITS); do
  status=$(curl -w "%{http_code}" -o /tmp/rl-write.out -sS \
    -X GET "$BASE_URL/admin/tenants" \
    -H "X-Admin-Token: $ADMIN_TOKEN")
  log_hit "write" "$i" "$status"
done
