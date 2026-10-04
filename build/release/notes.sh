#!/usr/bin/env bash
# notes.sh [--metainfo FILE] TAG [REF]: user-facing release notes for TAG from the Conventional Commits between the
# previous v* tag and REF (default TAG; a dry run passes HEAD before the tag exists). Only feat, fix and perf commits
# outside internal scopes are kept, as sentence-case lines without the type or scope, grouped New and Fixed and
# capped at $NOTES_CAP lines each, with the full changelog linked. With --metainfo, the same lines replace the
# <description> of TAG's <release> in the AppStream FILE instead of printing Markdown.
set -euo pipefail
metainfo=""
if [ "${1:-}" = --metainfo ]; then
  metainfo="${2:?usage: notes.sh --metainfo FILE TAG [REF]}"
  shift 2
fi
tag="${1:?usage: notes.sh [--metainfo FILE] TAG [REF]}"
ref="${2:-$tag}"
cap="${NOTES_CAP:-15}"
repo="${GITHUB_REPOSITORY:-Rethunk-AI/mortar}"
prev="$(git describe --tags --match 'v*' --abbrev=0 "${ref}^" 2>/dev/null || true)"
subjects="$(git log --no-merges --format=%s "${prev:+$prev..}$ref")"

# Scopes that name code, tooling or the automation CLI rather than something a user sees, and subjects that name code
# (camelCase or snake_case identifiers, backticks) or tooling.
internal='^([a-z-]*svc|main|release|gate|lint|deps|build|ci|tests?|selftest|dev|fsx|datadir|data|meta|usererr|github|site|sampler|notices|credits|migrate|config|components|nativehost|errors|logs?|cli)$'
codeish='[a-z][A-Z]|[a-z]_[a-z]|`|(^|[^a-z])(selftest|gate|lint|biome|knip|golangci|gui-design|frontend build|api types)([^a-z]|$)'

lines() { # types regex -> one sentence-case line per user-facing subject
  printf '%s\n' "$subjects" | sed -nE "s/^($1)(\(([^)]*)\))?!?: (.+)\$/\3\x1f\4/p" |
    while IFS=$'\x1f' read -r scope text; do
      if { [ -n "$scope" ] && [[ "$scope" =~ $internal ]]; } || [[ "$text" =~ $codeish ]]; then
        continue
      fi
      text="${text%.}"
      printf '%s\n' "${text^}"
    done
}

new="$(lines feat)"
fixed="$(lines 'fix|perf')"
if [ -n "$prev" ]; then
  changelog="https://github.com/$repo/compare/$prev...$tag"
else
  changelog="https://github.com/$repo/commits/$tag"
fi

capped() { # lines -> at most $cap of them, then a count of the rest
  local n
  n="$(printf '%s\n' "$1" | wc -l)"
  printf '%s\n' "$1" | head -n "$cap"
  if [ "$n" -gt "$cap" ]; then
    printf '…and %d more\n' "$((n - cap))"
  fi
}

markdown() {
  local title body
  for title in New Fixed; do
    body="$fixed"
    [ "$title" = New ] && body="$new"
    [ -n "$body" ] && printf '## %s\n\n%s\n\n' "$title" "$(capped "$body" | sed 's/^/- /')"
  done
  if [ -z "$new$fixed" ]; then
    printf 'No user-facing changes.\n\n'
  fi
  printf 'The Linux AppImage and portable program need glibc 2.38 or newer (Ubuntu 24.04, Debian 13, Fedora 39 or later); older systems use the Flatpak.\n\n'
  printf '[Full changelog](%s)\n' "$changelog"
}

xml_escape() { sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g'; }

description() { # the AppStream <description>, indented for the <release> element
  local title body
  printf '      <description>\n'
  for title in New Fixed; do
    body="$fixed"
    [ "$title" = New ] && body="$new"
    if [ -n "$body" ]; then
      printf '        <p>%s</p>\n        <ul>\n' "$title"
      capped "$body" | xml_escape | sed 's|.*|          <li>&</li>|'
      printf '        </ul>\n'
    fi
  done
  if [ -z "$new$fixed" ]; then
    printf '        <p>No user-facing changes.</p>\n'
  fi
  printf '      </description>\n'
}

if [ -z "$metainfo" ]; then
  markdown
  exit 0
fi

version="${tag#v}"
if ! grep -q "<release version=\"$version\"" "$metainfo"; then
  echo "notes.sh: $metainfo has no <release version=\"$version\">" >&2
  exit 1
fi
desc="$(description)"
# Rewrites TAG's <release> as an open element holding only the new description, dropping any earlier one.
awk -v ver="$version" -v desc="$desc" '
  skip && /<\/release>/ { skip = 0; next }
  skip { next }
  index($0, "<release version=\"" ver "\"") {
    line = $0
    open = (line !~ /\/>[[:space:]]*$/)
    sub(/[[:space:]]*\/>[[:space:]]*$/, ">", line)
    print line
    print desc
    match($0, /^[[:space:]]*/)
    print substr($0, 1, RLENGTH) "</release>"
    skip = open
    next
  }
  { print }
' "$metainfo" >"$metainfo.tmp"
mv "$metainfo.tmp" "$metainfo"
