#!/usr/bin/env bash
# Generates the frontend bindings unless the Go sources and the generator are what the last generated set came from.
# A frontend-only change, or a rerun on an unchanged tree, keeps the set instead of spending seconds regenerating it.
set -euo pipefail
cd "$(dirname "$0")/.."
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
