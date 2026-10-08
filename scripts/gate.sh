#!/usr/bin/env bash
# The local gate: CI's checks minus opt-in e2e. Biome, knip, tsc and bun test read the generated bindings, so they
# start once those exist; the Go steps and the rest never read them and start at once, so a cold build cache is
# not spent waiting on the bindings. The gate takes as long as the slowest step (the race tests).
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
export GORACE=atexit_sleep_ms=0

# One folder per run holds the step logs and every step's scratch (Go test binaries, Chromium profiles, bun's temp
# files), so a run killed mid-way leaks one folder that the next run reaps, not scattered go-build and chromium
# dirs. On /var/tmp, not tmpfs: the race test binaries are large. The name stays short because a test binds a unix
# socket under it, and a socket path holds 107 bytes.
base=/var/tmp
for old in "$base"/.mg-*; do
  pid=${old#"$base"/.mg-}
  if [[ $pid =~ ^[0-9]+$ ]] && [ -d "$old" ] && [ ! -L "$old" ] && ! kill -0 "$pid" 2>/dev/null; then
    rm -rf -- "$old"
  fi
done
logs=$base/.mg-$$
mkdir -p "$logs"
trap 'rm -rf "$logs"' EXIT
export TMPDIR=$logs GOTMPDIR=$logs
# go test caches a result only while the TMPDIR and GOTMPDIR it ran under stay the same, so the race tests get one
# fixed folder; what a killed run leaves in it is reaped once it is an hour old.
gotmp=$base/.mg-go
mkdir -p "$gotmp"
find "$gotmp" -mindepth 1 -maxdepth 1 -mmin +60 -exec rm -rf -- {} +

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
# golangci-lint spends a quarter of its CPU in GC at the default GOGC, and a cold gate is CPU-bound, so lint trades
# memory for it. The updatetest pass runs after the linux one so it reuses that pass's cache for every dependency.
{
  export GOGC=400
  golangci-lint run --allow-parallel-runners
  a=$?
  golangci-lint run --allow-parallel-runners --build-tags updatetest ./internal/updatesvc/... && exit "$a"
} >"$logs/golangci-linux" 2>&1 &
pids[golangci-linux]=$!
step golangci-windows env GOGC=400 GOOS=windows golangci-lint run --allow-parallel-runners
step version go run ./cmd/version -check
step site-games bun scripts/site-games.ts --check
step i18n-dupes bun scripts/i18n-dupes.ts
step metainfo appstreamcli validate --strict --no-net build/linux/tech.rethunk.Mortar.metainfo.xml
step go-test env TMPDIR="$gotmp" GOTMPDIR="$gotmp" go test -race ./...
step vuln scripts/vulncheck.sh

wait "$bindings_pid" || { cat "$logs/bindings"; exit 1; }
step biome bun run --silent lint
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
# A passing step prints nothing, so a skipped vuln check is the one line worth showing.
grep -h '^govulncheck skipped' "$logs/vuln" 2>/dev/null
exit $failed
