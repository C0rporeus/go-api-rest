#!/usr/bin/env bash
# Canary for the DNSBL providers, run against a deployed API (resolvers differ
# between a laptop and Cloud Run, so only the deployed result is meaningful).
#   127.0.0.2 must be "listed" and 127.0.0.1 must be "not_listed" for a provider
#   to be trustworthy. Providers answering "error" are reported and, if too many
#   fail, the script exits non-zero so the list can be pruned.
# Usage: scripts/check_dnsbl_canary.sh [API_URL] [MIN_HEALTHY]
set -euo pipefail

API_URL="${1:-https://api.yonathangutierrez.dev}"
MIN_HEALTHY="${2:-5}"

fetch() { curl -fsS --max-time 20 "${API_URL}/api/tools/dns/blacklist?ip=$1"; }

python3 - "$(fetch 127.0.0.2)" "$(fetch 127.0.0.1)" "$MIN_HEALTHY" <<'PY'
import json, sys
listed, clean, minimum = json.loads(sys.argv[1]), json.loads(sys.argv[2]), int(sys.argv[3])
clean_by = {r["provider"]: r for r in clean["results"]}
healthy, unhealthy = [], []
for r in listed["results"]:
    c = clean_by[r["provider"]]
    ok = r["status"] == "listed" and c["status"] == "not_listed"
    (healthy if ok else unhealthy).append((r["provider"], r["status"], r.get("error", ""), c["status"]))
print(f"healthy ({len(healthy)}):")
for p in healthy: print("  OK  ", p[0])
print(f"unhealthy ({len(unhealthy)}):")
for p in unhealthy: print(f"  FAIL {p[0]}  canary={p[1]} {p[2]}  clean={p[3]}")
sys.exit(0 if len(healthy) >= minimum else 1)
PY
