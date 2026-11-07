#!/usr/bin/env bash
# repro-check.sh [REV] builds the Linux binary of REV (default HEAD) twice, each in its own fresh clone at a different
# path with an empty Go build cache, and fails when the binaries or the embedded frontend bundles differ.
# Needs the release job's tools (setup-mortar action): GTK4/WebKitGTK headers, Go, bun, wails3. Keep it off the gate:
# two cold builds take several minutes.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
rev="$(git -C "$root" rev-parse "${1:-HEAD}")"
work="$(mktemp -d /var/tmp/mortar-repro.XXXXXX)"
trap 'rm -rf "$work"' EXIT
SOURCE_DATE_EPOCH="$(git -C "$root" log -1 --format=%ct "$rev")"
export SOURCE_DATE_EPOCH
# The clones are built at paths of different length, which -trimpath must make irrelevant.
paths=("$work/a" "$work/second-build-tree")

for i in 0 1; do
  dir="${paths[$i]}"
  git clone -q --no-hardlinks "$root" "$dir"
  git -C "$dir" -c advice.detachedHead=false checkout -q "$rev"
  (
    cd "$dir"
    export GOCACHE="$work/gocache-$i" GOFLAGS=-mod=readonly TZ=UTC LC_ALL=C
    bun install --frozen-lockfile >/dev/null
    scripts/bindings-fresh.sh >/dev/null
    wails3 task linux:build >/dev/null
  )
  (cd "$dir/frontend/dist" && find . -type f -print0 | sort -z | xargs -0 sha256sum) >"$work/dist-$i.sha"
  sha256sum "$dir/bin/mortar" | cut -d' ' -f1 >"$work/bin-$i.sha"
done

status=0
if ! diff -u "$work/dist-0.sha" "$work/dist-1.sha"; then
  echo "frontend bundle differs between builds" >&2
  status=1
fi
if ! cmp -s "$work/bin-0.sha" "$work/bin-1.sha"; then
  echo "linux binary differs: $(cat "$work/bin-0.sha") vs $(cat "$work/bin-1.sha")" >&2
  status=1
fi
[ "$status" = 0 ] && echo "reproducible: $(cat "$work/bin-0.sha") mortar-linux at ${rev:0:8}"
exit "$status"
