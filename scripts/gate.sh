#!/usr/bin/env bash
# The local gate: CI's checks minus opt-in e2e. Biome, knip, tsc and bun test read the generated bindings, so they
# start once those exist; the Go steps and the rest never read them and start at once, so a cold build cache is
# not spent waiting on the bindings. The gate takes as long as the slowest step (the race tests).
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
export GOTMPDIR="${GOTMPDIR:-/var/tmp}" TMPDIR="${TMPDIR:-/var/tmp}" GORACE=atexit_sleep_ms=0

logs=$(mktemp -d "$TMPDIR/gate.XXXXXX")
trap 'rm -rf "$logs"' EXIT

declare -A pids
step() {
  local name=$1
  shift
  "$@" >"$logs/$name" 2>&1 &
  pids[$name]=$!
}

bun run bindings >"$logs/bindings" 2>&1 &
bindings_pid=$!

step shellcheck bash -c "set -o pipefail; git ls-files -co --exclude-standard -z '*.sh' | xargs -0 -r shellcheck -x"
step golangci-linux golangci-lint run --allow-parallel-runners
step golangci-windows env GOOS=windows golangci-lint run --allow-parallel-runners
step golangci-updatetest golangci-lint run --allow-parallel-runners --build-tags updatetest ./internal/updatesvc/...
step version go run ./cmd/version -check
step site-games bun scripts/site-games.ts --check
step i18n-dupes bun scripts/i18n-dupes.ts
step metainfo appstreamcli validate --strict --no-net build/linux/tech.rethunk.Mortar.metainfo.xml
step go-test go test -race ./...
step vuln govulncheck ./...

wait "$bindings_pid" || { cat "$logs/bindings"; exit 1; }
step biome scripts/biome-strict.sh
step knip bunx knip
step tsc bun run --cwd frontend tsc --noEmit
step bun-test bun test

failed=0
for name in "${!pids[@]}"; do
  if ! wait "${pids[$name]}"; then
    failed=1
    echo "=== FAILED: $name"
    cat "$logs/$name"
  fi
done
exit $failed
