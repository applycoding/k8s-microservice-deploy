#!/usr/bin/env bash
# HTTP smoke checks for a running trips API. Used by GitHub Actions after Helm install.
set -euo pipefail

BASE="${BASE_URL:-http://127.0.0.1:8080}"

need() {
  local code
  code="$(curl -sS -o /tmp/trips-body -w "%{http_code}" "$@" || true)"
  echo "HTTP ${code} $(cat /tmp/trips-body)"
  if [[ "$code" != "${EXPECT}" ]]; then
    echo "expected HTTP ${EXPECT}" >&2
    exit 1
  fi
}

echo "== healthz"
EXPECT=200 need "${BASE}/healthz"

echo "== readyz"
ready=0
for _ in $(seq 1 30); do
  code="$(curl -sS -o /tmp/trips-body -w "%{http_code}" "${BASE}/readyz" || true)"
  if [[ "$code" == "200" ]]; then
    ready=1
    echo "HTTP ${code} $(cat /tmp/trips-body)"
    break
  fi
  sleep 2
done
if [[ "$ready" != "1" ]]; then
  echo "readyz did not become ready; last body: $(cat /tmp/trips-body 2>/dev/null || true)" >&2
  exit 1
fi

echo "== create"
EXPECT=201 need -X POST "${BASE}/v1/trips" \
  -H 'content-type: application/json' \
  -d '{"rider_id":"rider-42","pickup":"1 Market St","dropoff":"Ferry Building"}'
id="$(python3 -c 'import json; print(json.load(open("/tmp/trips-body"))["id"])')"
if [[ -z "$id" ]]; then
  echo "create response did not include an id" >&2
  exit 1
fi

echo "== get"
EXPECT=200 need "${BASE}/v1/trips/${id}"

echo "== list"
EXPECT=200 need "${BASE}/v1/trips"
python3 -c 'import json,sys; body=json.load(open("/tmp/trips-body")); ids=[t["id"] for t in body["trips"]]; sys.exit(0 if "'"${id}"'" in ids else 1)'

echo "== accept"
EXPECT=200 need -X PATCH "${BASE}/v1/trips/${id}" \
  -H 'content-type: application/json' \
  -d '{"status":"accepted","driver_id":"driver-7"}'
python3 -c 'import json,sys; body=json.load(open("/tmp/trips-body")); sys.exit(0 if body["status"]=="accepted" and body["driver_id"]=="driver-7" else 1)'

echo "== illegal transition"
EXPECT=409 need -X PATCH "${BASE}/v1/trips/${id}" \
  -H 'content-type: application/json' \
  -d '{"status":"requested"}'

echo "== missing"
EXPECT=404 need "${BASE}/v1/trips/missing-trip"

echo "== bad request"
EXPECT=400 need -X POST "${BASE}/v1/trips" \
  -H 'content-type: application/json' \
  -d '{}'

echo "== metrics"
metrics="$(curl -fsS "${BASE}/metrics")"
echo "${metrics}" | grep -q 'http_requests_total'
echo "${metrics}" | grep -E '^trips_created_total( |\{).*[1-9]'

echo "smoke ok"
