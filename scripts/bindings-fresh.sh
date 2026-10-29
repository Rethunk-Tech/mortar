#!/usr/bin/env bash
# Generates what the frontend imports but git does not track: the Lingui catalogs, compiled from the .po files every
# time (under a second), and the bindings, unless the Go sources and the generator are what the last set came from.
# A frontend-only change, or a rerun on an unchanged tree, keeps the bindings instead of spending seconds on them.
set -euo pipefail
cd "$(dirname "$0")/.."
bun run --cwd frontend catalogs
stamp=frontend/bindings/.inputs
want=$({
  wails3 version 2>&1
  git ls-files -co --exclude-standard -z -- '*.go' go.mod go.sum scripts/erasable-bindings.ts | xargs -0 sha1sum
} | sha1sum)
if [ "$(cat "$stamp" 2>/dev/null)" = "$want" ]; then
  exit 0
fi
bun run bindings
printf '%s\n' "$want" >"$stamp"
