#!/usr/bin/env bash
# govulncheck for the local gate (`bun run vuln`). Its findings change with go.mod, go.sum and bun.lock (the
# vulnerability database moves too, which CI catches), so locally it reruns only when those differ from the last
# passing run. CI, which sets CI, and MORTAR_GATE_VULNCHECK=1 always run it.
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1

stamp=tmp/gate/vulncheck.stamp
sum=$(cat go.mod go.sum bun.lock | sha256sum | cut -d' ' -f1)

if [ -z "${CI:-}" ] && [ -z "${MORTAR_GATE_VULNCHECK:-}" ] && [ "$(cat "$stamp" 2>/dev/null)" = "$sum" ]; then
  echo "govulncheck skipped: go.mod, go.sum and bun.lock are unchanged since the last passing run (MORTAR_GATE_VULNCHECK=1 forces it)"
  exit 0
fi

# x/vuln reads the database with http.DefaultClient, whose requests carry no deadline, and it prints nothing until the
# scan ends: a database that takes the connection and then stalls waits forever, which is how a CI gate run sat silent
# for its whole 35-minute job budget. The scan is bounded so a stall fails this step and names itself.
limit=${MORTAR_VULNCHECK_TIMEOUT:-600}
govulncheck ./... &
pid=$!
started=$SECONDS
while kill -0 "$pid" 2>/dev/null; do
  if [ $((SECONDS - started)) -ge "$limit" ]; then
    kill "$pid" 2>/dev/null
    wait "$pid" 2>/dev/null
    echo "govulncheck: the vulnerability database gave no answer within ${limit}s, so it is stalled (MORTAR_VULNCHECK_TIMEOUT)" >&2
    exit 1
  fi
  sleep 0.2
done
wait "$pid" || exit 1
mkdir -p "$(dirname "$stamp")"
printf '%s\n' "$sum" >"$stamp"
