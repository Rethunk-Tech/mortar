#!/usr/bin/env bash
# Memory budget: peak PSS of Mortar's Go server and of the browser (headless Chromium standing in for the WebView) over
# idle, browse, a large install, a Problems check and a launch, in a throwaway self-test sandbox. Exits 1 over the budget.
# Knobs and their meaning: frontend/e2e/mem-budget.ts. Opt-in (`bun run mem`), in no gate: it needs the real 811-mod data to mean anything and runs for many minutes.
#
#   MORTAR_MEM_DATA=/path/to/mortar-data scripts/mem-budget.sh      every scenario on that data; report.md in MORTAR_MEM_OUT
#   MORTAR_MEM_ONLY=idle scripts/mem-budget.sh                      one scenario on the seeded sandbox (smoke)
set -euo pipefail
cd "$(dirname "$0")/.."
exec bun frontend/e2e/mem-budget.ts
