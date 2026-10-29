#!/usr/bin/env bash
# notes.sh [--metainfo FILE] TAG [REF]: user-facing release notes for TAG from the Conventional Commits between the
# previous v* tag and REF (default TAG; a dry run passes HEAD before the tag exists). Only feat, fix and perf commits
# outside internal scopes are kept, as sentence-case lines without the type or scope, grouped New and Fixed, each
# under the area its scope belongs to, and capped at $NOTES_CAP lines each, with the full changelog linked. With --metainfo, the same lines replace the
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
repo="${GITHUB_REPOSITORY:-Rethunk-Tech/mortar}"
prev="$(git describe --tags --match 'v*' --abbrev=0 "${ref}^" 2>/dev/null || true)"
subjects="$(git log --no-merges --format=%s "${prev:+$prev..}$ref")"

# Scopes that name code, tooling or the automation CLI rather than something a user sees, and subjects that name code
# (camelCase or snake_case identifiers, backticks) or tooling.
internal='^([a-z-]*svc|main|release|gate|lint|deps|build|ci|tests?|selftest|regress|e2e|dotnet|taskfile|scripts|server|control|doctor|dev|safefs|datadir|data|meta|usererr|github|site|sampler|notices|credits|migrate|config|components|nativehost|errors|logs?|cli|opener|docs|docscheck|i18n)$'
codeish='[a-z][A-Z]|[A-Z]+_[A-Z]|[a-z]_[a-z]|`|(^|[^a-z])(selftest|gate|lint|biome|knip|golangci|gui-design|frontend build|api types)([^a-z]|$)'
# Product names spelled in camel case are words a player knows, not identifiers.
products='BepInEx|WebKit|GitHub|SpaceCore|ModConfig|PlayStation|YouTube|macOS|iOS'

# The areas the app's What's new groups a release by, in the order it shows them.
areas=(Games 'Mods and browsing' 'Profiles and sharing' 'Launching and loaders' 'Problems and diagnostics' 'Downloads and updates' Saves 'Interface and settings' Other)
area_of() { # scope -> area
  case "$1" in
    game | games | steam | catalog | store | stores) echo Games ;;
    browse | mods | mod | source | sources | templates | bundles | thunderstore | modrinth | nexus | curseforge) echo 'Mods and browsing' ;;
    profile | profiles | share | packs | pack | import | history | sync | lan) echo 'Profiles and sharing' ;;
    launch | loader | smapi | bepinex | bepinex5 | deploy) echo 'Launching and loaders' ;;
    problems | diag | console | support) echo 'Problems and diagnostics' ;;
    queue | updates | update | watch | nxm) echo 'Downloads and updates' ;;
    saves | backup | backups) echo Saves ;;
    settings | ui | ux | theme | shell | frontend | firstrun | tour | a11y) echo 'Interface and settings' ;;
    *) echo Other ;;
  esac
}

lines() { # types regex -> one "area<US>sentence-case line" per user-facing subject
  printf '%s\n' "$subjects" | sed -nE "s/^($1)(\(([^)]*)\))?!?: (.+)\$/\3\x1f\4/p" |
    while IFS=$'\x1f' read -r scope text; do
      if { [ -n "$scope" ] && [[ "$scope" =~ $internal ]]; } || [[ "$(sed -E "s/(^|[^A-Za-z])($products)([^A-Za-z]|\$)/\1\3/g" <<<"$text")" =~ $codeish ]]; then
        continue
      fi
      text="${text%.}"
      printf '%s\x1f%s\n' "$(area_of "$scope")" "${text^}"
    done
}

by_area() { # "area<US>line" lines -> each area's heading (RS-marked, as a notes file's "### Area") and its lines
  local a body
  for a in "${areas[@]}"; do
    body="$(printf '%s\n' "$1" | awk -F'\037' -v a="$a" '$1 == a { print $2 }')"
    if [ -n "$body" ]; then
      printf '\x1e%s\n%s\n' "$a" "$body"
    fi
  done
}

new="$(by_area "$(lines feat)")"
changed=""
fixed="$(by_area "$(lines 'fix|perf')")"
# A release with notes written for players (build/release/notes/TAG.md: "## New", "## Changed" and "## Fixed" lists)
# uses them whole instead of commit subjects, which cannot rank a large release's headlines first. A "### Area"
# heading inside a list groups the lines under it: the release page keeps it (the app's What's new groups by it) and
# the AppStream description, which has no sub-headings, leaves it out.
curated="${NOTES_FILE:-$(dirname "$0")/notes/$tag.md}"
if [ -f "$curated" ]; then
  section() { awk -v want="## $1" '/^## /{ on = ($0 == want); next } on && /^### / { print "\x1e" substr($0, 5); next } on && /^- /{ print substr($0, 3) }' "$curated"; }
  new="$(section New)"
  changed="$(section Changed)"
  fixed="$(section Fixed)"
  cap=100000
fi
if [ -n "$prev" ]; then
  changelog="https://github.com/$repo/compare/$prev...$tag"
else
  changelog="https://github.com/$repo/commits/$tag"
fi

capped() { # lines -> at most $cap of them (area headings not counted), then a count of the rest
  printf '%s\n' "$1" | awk -v cap="$cap" '
    index($0, "\036") == 1 { if (n < cap) print; next }
    { n++; if (n <= cap) print }
    END { if (n > cap) printf "…and %d more\n", n - cap }'
}

group() { # title -> its lines
  case "$1" in
    New) printf '%s' "$new" ;;
    Changed) printf '%s' "$changed" ;;
    Fixed) printf '%s' "$fixed" ;;
  esac
}

markdown() {
  local title body
  for title in New Changed Fixed; do
    body="$(group "$title")"
    [ -n "$body" ] && printf '## %s\n\n%s\n\n' "$title" "$(capped "$body" | sed -e '/^\x1e/!s/^/- /' -e 's/^\x1e\(.*\)/\n### \1\n/')"
  done
  if [ -z "$new$changed$fixed" ]; then
    printf 'No user-facing changes.\n\n'
  fi
  printf 'The Linux AppImage and portable program need glibc 2.35 or newer (Ubuntu 22.04, Linux Mint 21, Debian 12, Fedora 36 or later); older systems use the Flatpak.\n\n'
  printf '[Full changelog](%s)\n' "$changelog"
}

xml_escape() { sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g'; }

description() { # the AppStream <description>, indented for the <release> element
  local title body
  printf '      <description>\n'
  for title in New Changed Fixed; do
    body="$(group "$title")"
    if [ -n "$body" ]; then
      printf '        <p>%s</p>\n        <ul>\n' "$title"
      capped "$body" | grep -v $'^\x1e' | xml_escape | sed 's|.*|          <li>&</li>|'
      printf '        </ul>\n'
    fi
  done
  if [ -z "$new$changed$fixed" ]; then
    printf '        <p>No user-facing changes.</p>\n'
  fi
  printf '      </description>\n'
}

if [ -z "$metainfo" ]; then
  markdown | cat -s
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
