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

govulncheck ./... || exit 1
mkdir -p "$(dirname "$stamp")"
printf '%s\n' "$sum" >"$stamp"
