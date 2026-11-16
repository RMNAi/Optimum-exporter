#!/usr/bin/env bash
# Smoke-test json_exporter extraction against fixture /health responses.
# Run from anywhere: bash tests/smoke.sh   (needs python3 + docker)
set -euo pipefail
cd "$(dirname "$0")/.."

python3 tests/mock_gateway.py 8000 &
MOCK_PID=$!
docker rm -f og-smoke-json >/dev/null 2>&1 || true
docker run -d --name og-smoke-json --network host \
  -v "$PWD/json_exporter/config.yml:/etc/json/config.yml:ro" \
  prometheuscommunity/json-exporter:v0.6.0 \
  --config.file=/etc/json/config.yml >/dev/null
cleanup() { docker rm -f og-smoke-json >/dev/null 2>&1 || true; kill "$MOCK_PID" 2>/dev/null || true; }
trap cleanup EXIT

for _ in $(seq 1 30); do curl -sf http://localhost:7979/ >/dev/null && break; sleep 1; done

probe()  { curl -sf "http://localhost:7979/probe?module=optimum_health&target=http://localhost:8000$1" \
           || { echo "FAIL: probe $1 errored (503 acceptance regression?)"; exit 1; }; }
has()    { printf '%s\n' "$1" | grep -Eq "^$2 $3\$" \
           || { echo "FAIL: expected $2 = $3"; printf '%s\n' "$1"; exit 1; }; }
absent() { if printf '%s\n' "$1" | grep -Eq "^$2([ {]|\$)"; then
             echo "FAIL: $2 should be absent"; printf '%s\n' "$1"; exit 1; fi; }

out=$(probe /stream-only)
has "$out" optimum_mump2p_peers 44
has "$out" optimum_uptime_seconds 3018
absent "$out" optimum_cl_peers
echo "ok: stream-only (200) - CL series absent"

out=$(probe /cl-healthy)
has "$out" optimum_cl_peers 2
echo "ok: CL-healthy (200)"

out=$(probe /stall)
has "$out" optimum_last_block_age_seconds 95
echo "ok: stall (503) - gauges survive degraded"

out=$(probe /cl-fail)
has "$out" optimum_cl_peers 0
echo "ok: CL-fail (503) - alert input present"

echo "smoke: all fixtures passed"