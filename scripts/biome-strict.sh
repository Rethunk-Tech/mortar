#!/usr/bin/env bash
# Biome has no switch that fails on infos, and the preset's size and style rules report at that level, so the
# summary line is the check.
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
out=$(bun run --silent lint:biome 2>&1)
status=$?
printf '%s\n' "$out"
[[ $status -eq 0 ]] && ! grep -Eq '^Found [0-9]+ infos?\.' <<<"$out"
