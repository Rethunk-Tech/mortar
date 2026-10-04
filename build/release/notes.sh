#!/usr/bin/env bash
# Release notes for TAG from the Conventional Commits since the previous tag: feat and fix only, grouped.
set -euo pipefail
tag="${1:?usage: notes.sh TAG}"
prev="$(git describe --tags --abbrev=0 "${tag}^" 2>/dev/null || true)"
range="${prev:+$prev..}$tag"
subjects="$(git log --no-merges --format=%s "$range")"

section() { # title, type
  local lines
  lines="$(printf '%s\n' "$subjects" | sed -nE -e "s/^$2\(([^)]*)\)!?: (.+)\$/- **\1:** \2/p" -e "s/^$2!?: (.+)\$/- \1/p")"
  if [ -n "$lines" ]; then
    printf '## %s\n\n%s\n\n' "$1" "$lines"
  fi
}

out="$(section Features feat; section Fixes fix)"
if [ -z "$out" ]; then
  out="No user-facing changes."
fi
printf '%s\n' "$out"
if [ -n "$prev" ]; then
  printf '\n**Full changelog:** https://github.com/%s/compare/%s...%s\n' "${GITHUB_REPOSITORY:-Rethunk-AI/mortar}" "$prev" "$tag"
fi
