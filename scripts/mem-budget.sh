#!/usr/bin/env bash
# Memory budget: peak PSS of Mortar's Go server and of the browser (headless Chromium standing in for the WebView) over
# browse, a large install, a Problems check and a launch, in a throwaway self-test sandbox. Exits 1 over the budget.
# Knobs and their meaning: frontend/e2e/mem-budget.ts. Opt-in (`gate --e2e`), never the default gate.
#
#   MORTAR_MEM_DATA=/path/to/mortar-data scripts/mem-budget.sh      all four scenarios on that data
#   MORTAR_MEM_ONLY=browse scripts/mem-budget.sh                    one scenario on the seeded sandbox (smoke)
set -euo pipefail
cd "$(dirname "$0")/.."
exec bun frontend/e2e/mem-budget.ts
