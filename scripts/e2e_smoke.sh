#!/usr/bin/env bash
set -euo pipefail
BASE_URL="${BASE_URL:-http://localhost:8080}"
EMAIL="${E2E_EMAIL:-}"
PASSWORD="${E2E_PASSWORD:-}"

need(){ command -v "$1" >/dev/null || { echo "missing dependency: $1" >&2; exit 2; }; }
need curl
need python3

curl -fsS "$BASE_URL/healthz" >/dev/null
echo "PASS healthz"
curl -fsS "$BASE_URL/readyz" >/dev/null
echo "PASS readyz"

if [[ -z "$EMAIL" || -z "$PASSWORD" ]]; then
  echo "SKIP authenticated smoke checks: set E2E_EMAIL and E2E_PASSWORD"
  exit 0
fi

login="$(curl -fsS "$BASE_URL/api/v1/auth/login" -H 'Content-Type: application/json' \
  --data "$(python3 - "$EMAIL" "$PASSWORD" <<'PY'
import json,sys
print(json.dumps({"email":sys.argv[1],"password":sys.argv[2]}))
PY
)")"
token="$(printf '%s' "$login" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')"
[[ -n "$token" ]]
echo "PASS login"

curl -fsS "$BASE_URL/api/v1/users/me" -H "Authorization: Bearer $token" >/dev/null
echo "PASS current user"
curl -fsS "$BASE_URL/api/v1/trainers" -H "Authorization: Bearer $token" >/dev/null
echo "PASS trainers"
curl -fsS "$BASE_URL/api/v1/shop/products" -H "Authorization: Bearer $token" >/dev/null
echo "PASS products"

echo "E2E smoke checks complete (AI intentionally not exercised)."
