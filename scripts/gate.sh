#!/usr/bin/env bash
# The local gate: CI's checks minus opt-in e2e. Biome, knip, tsc and bun test read the generated bindings, so they
# start once those exist; the Go steps and the rest never read them and start at once, so a cold build cache is
# not spent waiting on the bindings. The gate takes as long as the slowest step (the race tests).
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
export GORACE=atexit_sleep_ms=0

profile=0
if [[ "${1:-}" == --profile ]]; then
  profile=1
fi
gate_t0=$(date +%s%N)
profile_log() {
  [ "$profile" = 1 ] || return 0
  local name=$1 start=$2 rc=$3
  local end elapsed
  end=$(date +%s%N)
  elapsed=$(awk -v s="$start" -v e="$end" 'BEGIN { printf "%.2f", (e-s)/1e9 }')
  printf '%s %ss rc=%s\n' "$name" "$elapsed" "$rc" >>"$logs/profile"
}

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
# The steps run in the background, so a gate that is itself killed (a CI step timeout, Ctrl-C, a stopped runner)
# would leave them running, and on CI holding the runner until the job's own timeout. Its exit ends every process
# it started, listed by parent before any is signalled.
# shellcheck disable=SC2329 # called by cleanup
descendants() {
  local c
  for c in $(pgrep -P "$1"); do
    [ "$c" = "$BASHPID" ] && continue
    descendants "$c"
    echo "$c"
  done
}
# shellcheck disable=SC2329 # run by the EXIT trap
cleanup() {
  local left=()
  mapfile -t left < <(descendants "$$")
  [ ${#left[@]} -eq 0 ] || kill "${left[@]}" 2>/dev/null
  rm -rf "$logs"
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM
export TMPDIR=$logs GOTMPDIR=$logs
# go test caches a result only while the TMPDIR and GOTMPDIR it ran under stay the same, so the race tests get one
# fixed folder; what a killed run leaves in it is reaped once it is an hour old.
gotmp=$base/.mg-go
mkdir -p "$gotmp"
find "$gotmp" -mindepth 1 -maxdepth 1 -mmin +60 -exec rm -rf -- {} +

# The module cache is primed here, before any step starts and under a bounded wait: cmd/go fetches a module the cache
# lacks through a client with no deadline, and a step writes its output to its own log until it ends, so a proxy that
# takes the connection and then stalls leaves the whole gate silent with nothing to read. Primed, every step below
# reads the cache instead of the network.
modules=$logs/modules
go mod download >"$modules" 2>&1 &
dl=$!
started=$SECONDS
while kill -0 "$dl" 2>/dev/null; do
  if [ $((SECONDS - started)) -ge 300 ]; then
    kill "$dl" 2>/dev/null
    wait "$dl" 2>/dev/null
    cat "$modules"
    echo "=== FAILED: go mod download (300s and no module arrived, so the module proxy is stalled)"
    exit 1
  fi
  sleep 0.2
done
if ! wait "$dl"; then
  cat "$modules"
  echo "=== FAILED: go mod download"
  exit 1
fi

declare -A pids
step() {
  local name=$1
  shift
  (
    local t rc=0
    t=$(date +%s%N)
    "$@" || rc=$?
    profile_log "$name" "$t" "$rc"
    exit "$rc"
  ) >"$logs/$name" 2>&1 &
  pids[$name]=$!
}

(
  t=$(date +%s%N)
  rc=0
  scripts/bindings-fresh.sh || rc=$?
  profile_log bindings "$t" "$rc"
  exit "$rc"
) >"$logs/bindings" 2>&1 &
bindings_pid=$!

step shellcheck bash -c "set -o pipefail; git ls-files -co --exclude-standard -z '*.sh' | xargs -0 -r shellcheck -x"
# The race build, linux lint, govulncheck and the race tests all start at once and share GOCACHE: GOFLAGS=-race
# makes lint and govulncheck load packages from the same instrumented compile the tests link, so the Wails cgo
# step and sqlite compile once, not once per tool. Waiting for the build first stacked the test-binary compile
# after that compile. The windows lint cannot share it (another GOOS), so it starts at once beside this chain.
# golangci-lint spends a quarter of its CPU in GC at the default GOGC, and a cold gate is CPU-bound, so lint trades
# memory for it. The updatetest pass runs after the linux one so it reuses that pass's cache for every dependency.
{
  rc=0
  (
    t=$(date +%s%N)
    go build -race ./...
    a=$?
    profile_log race-build "$t" "$a"
    exit "$a"
  ) >"$logs/race-build" 2>&1 &
  build_pid=$!
  (
    t=$(date +%s%N)
    export GOFLAGS=-race GOGC=400
    golangci-lint run --allow-parallel-runners
    a=$?
    golangci-lint run --allow-parallel-runners --build-tags updatetest ./internal/updatesvc/...
    b=$?
    if [ "$b" != 0 ]; then
      a=$b
    fi
    profile_log golangci-linux "$t" "$a"
    exit "$a"
  ) >"$logs/golangci-linux" 2>&1 &
  lint_pid=$!
  (
    t=$(date +%s%N)
    GOFLAGS=-race scripts/vulncheck.sh
    a=$?
    profile_log vuln "$t" "$a"
    exit "$a"
  ) >"$logs/vuln" 2>&1 &
  vuln_pid=$!
  (
    t=$(date +%s%N)
    mapfile -t pkgs < <(go list ./...)
    # One shard per core, up to 8: each shard holds race-instrumented test binaries in memory, and a runner with few
    # cores has little memory too.
    shards=$(nproc)
    [ "$shards" -le 8 ] || shards=8
    printf '%s\0' "${pkgs[@]}" | xargs -0 -n 16 -P "$shards" env TMPDIR="$gotmp" GOTMPDIR="$gotmp" go test -race
    a=$?
    profile_log go-test "$t" "$a"
    exit "$a"
  ) >"$logs/go-test" 2>&1 &
  test_pid=$!
  for child in "$build_pid" "$lint_pid" "$vuln_pid" "$test_pid"; do
    wait "$child" || rc=1
  done
  if [ "$rc" = 1 ]; then
    cat "$logs/race-build" "$logs/golangci-linux" "$logs/vuln" "$logs/go-test"
  fi
  exit "$rc"
} >"$logs/racechain" 2>&1 &
pids[racechain]=$!
step golangci-windows env GOGC=400 GOOS=windows golangci-lint run --allow-parallel-runners
step vet-darwin env GOOS=darwin go vet ./internal/...
step version go run ./cmd/version -check
step i18n-dupes bun scripts/i18n-dupes.ts
# Locally a dependency bump rewrites the credits for its commit to carry (the lefthook credits job stages them); CI
# only checks that the commit did.
if [ -n "${CI:-}" ]; then
  step credits bun scripts/gen-credits.ts --check
else
  step credits bun scripts/gen-credits.ts
fi
step metainfo appstreamcli validate --strict --no-net build/linux/tech.rethunk.Mortar.metainfo.xml

wait "$bindings_pid" || {
  cat "$logs/bindings"
  exit 1
}
step biome bun run --silent lint
# The site and its demo need the bindings too; the build checks every link and picture it writes.
step site bun run --silent site
step site-types bun run --cwd web --silent check
step knip bunx knip
step tsc bun run --cwd frontend tsc --noEmit
# A CI runner shares its few cores with the race build and both lints running beside this, so tests that finish in
# milliseconds here can pass bun's 5s default there.
# A random file order each run, so a test that leaks shared state fails here, not only under CI's file order.
step bun-test bun test --randomize ${CI:+--timeout 20000}

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
if [ "$profile" = 1 ]; then
  profile_log wall "$gate_t0" "$failed"
  echo "=== gate profile (slowest first)"
  sort -k2 -nr "$logs/profile"
fi
exit $failed
