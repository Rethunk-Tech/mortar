#!/usr/bin/env bash
# Runs the fresh-clone gate in one reusable clean tree instead of a new clone per build.
#
#   scripts/clean-gate.sh [--package] [<rev>]   (rev defaults to HEAD of this checkout)
#
# The tree ($MORTAR_CLEAN_GATE_DIR, default /var/tmp/mortar-clean-gate) is cloned from this checkout once; each run
# fetches, checks <rev> out detached and runs git clean -ffdx, keeping only node_modules so bun install reuses it.
# Then: bun install --frozen-lockfile, bindings, frontend build and the gate; with --package also wails3 task package,
# and the AppImage path is printed. All output goes to $TREE.log, overwritten each run.
set -euo pipefail

REPO=$(cd "$(dirname "$0")/.." && pwd)
TREE=${MORTAR_CLEAN_GATE_DIR:-/var/tmp/mortar-clean-gate}
LOG=$TREE.log
# Kept inside .git, which git clean never touches: it marks the tree as this script's own before anything is cleaned.
MARKER=.git/mortar-clean-gate
# Scratch for every step lives in the tree, which the next run's git clean empties, so a killed run leaks nothing.
export TMPDIR=$TREE/.tmp GOTMPDIR=$TREE/.tmp

package=""
if [ "${1:-}" = --package ]; then
  package=1
  shift
fi
[ $# -le 1 ] || {
  sed -n '2,9p' "$0"
  exit 2
}
sha=$(git -C "$REPO" rev-parse --verify "${1:-HEAD}^{commit}")

if [ ! -e "$TREE" ]; then
  git clone --quiet --no-checkout "$REPO" "$TREE"
  touch "$TREE/$MARKER"
fi
[ -f "$TREE/$MARKER" ] || {
  echo "$TREE exists but is not a clean-gate tree (no $MARKER); refusing to clean it" >&2
  exit 1
}

cd "$TREE"
echo "clean gate at $sha in $TREE (log $LOG)"
# Chained with && because set -e does not apply inside a command list that is followed by ||.
{
  git fetch --quiet origin &&
    git checkout --quiet --force --detach "$sha" &&
    git clean -ffdxq -e node_modules &&
    mkdir -p "$TMPDIR" &&
    bun install --frozen-lockfile &&
    bun run bindings &&
    bun run --cwd frontend build &&
    bun run gate
} >"$LOG" 2>&1 || {
  echo "gate FAILED at $sha; see $LOG" >&2
  exit 1
}
echo "gate passed at $sha"

if [ -n "$package" ]; then
  wails3 task package >>"$LOG" 2>&1 || {
    echo "package FAILED at $sha; see $LOG" >&2
    exit 1
  }
  echo "$TREE/bin/mortar-linux-$(uname -m).AppImage"
fi
