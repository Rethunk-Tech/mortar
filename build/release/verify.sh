#!/usr/bin/env bash
# verify.sh TAG DIR      downloads every asset of release TAG (a draft too) and checks it against the staged files in
#                        DIR and against the signed manifest.json (size and sha512 digest, as `wails3 updater manifest`
#                        writes them).
# verify.sh --staged DIR checks the staged files in DIR against their own manifest.json (a dry run, no release).
# verify.sh --public TAG fetches the published release's manifest and its assets with no token.
# Every mode also fetches, with no token, the component assets the bundled components.json pins and the newest signed
# components manifest, which is what a stranger's Mortar downloads. While the repo serving a URL is private an
# unreachable URL is a warning; once it is public it fails the check.
set -euo pipefail
usage="usage: verify.sh TAG DIR | --staged DIR | --public TAG"
repo="${GITHUB_REPOSITORY:-Rethunk-Tech/mortar}"
root="$(cd "$(dirname "$0")/../.." && pwd)"
mode=release
case "${1:-}" in
  --staged | --public)
    mode="${1#--}"
    shift
    ;;
esac
fail=0

# public OWNER/REPO: whether the repo is public; a lookup the token cannot make counts as private.
public() { [ "$(gh api "repos/$1" --jq .private 2>/dev/null || true)" = false ]; }

# anon URL: fetch the first byte of URL without credentials, as a stranger's Mortar would.
anon() {
  local url="$1" owner_repo
  if curl -fsL --retry 2 -r 0-0 -o /dev/null "$url"; then
    return 0
  fi
  owner_repo="$(printf '%s\n' "$url" | sed -E 's#^https://(api\.)?github\.com/(repos/)?([^/]+/[^/]+)/.*#\3#')"
  if public "$owner_repo"; then
    echo "UNREACHABLE without a token: $url" >&2
    fail=1
  else
    echo "WARNING: unreachable without a token while $owner_repo is private: $url" >&2
  fi
}

check_components() {
  local url
  while IFS= read -r url; do
    anon "$url"
  done < <(jq -r '.components[] | select(.source.host == "github.com")
    | "https://github.com/\(.source.owner)/\(.source.repo)/releases/download/\(.tag)/\(.asset)"' \
    "$root/internal/components/components.json")
  url="$(curl -fsSL "https://api.github.com/repos/$repo/releases?per_page=100" 2>/dev/null |
    jq -r '[.[] | select(.draft | not) | select(.tag_name | startswith("components-"))][0].assets[]?
      | select(.name == "components.json") | .browser_download_url' 2>/dev/null || true)"
  if [ -n "$url" ]; then
    anon "$url"
    anon "$url.sig"
  elif public "$repo"; then
    echo "NO components-* release is reachable without a token" >&2
    fail=1
  else
    echo "WARNING: no components-* release is visible without a token while $repo is private" >&2
  fi
}

# check_manifest DL: every artifact the manifest in DL names is in DL with the manifest's size and digest.
check_manifest() {
  local dl="$1" url size digest name got_size got_digest
  while IFS=$'\t' read -r url size digest; do
    name="$(basename "$url")"
    if [ ! -f "$dl/$name" ]; then
      echo "manifest lists $name but there is no such asset" >&2
      fail=1
      continue
    fi
    got_size="$(stat -c %s "$dl/$name")"
    got_digest="$(openssl dgst -sha512 -binary "$dl/$name" | base64 -w0)"
    if [ "$got_size" != "$size" ] || [ "$got_digest" != "$digest" ]; then
      echo "manifest mismatch for $name (size $got_size vs $size)" >&2
      fail=1
    fi
  done < <(jq -r '.artifacts[] | [.url, .size, .digest] | @tsv' "$dl/manifest.json")
}

# check_appimages DIR: the AppImage this host can run carries the AGPL text and the third-party notices.
check_appimages() {
  local f tmp
  [ "$(uname -m)" = x86_64 ] || return 0
  for f in "$1"/*-x86_64.AppImage; do
    [ -f "$f" ] || continue
    tmp="$(mktemp -d)"
    cp "$f" "$tmp/a.AppImage"
    chmod +x "$tmp/a.AppImage"
    (cd "$tmp" && ./a.AppImage --appimage-extract 'usr/share/doc/mortar/*' >/dev/null 2>&1) || true
    for n in LICENSE THIRD_PARTY_NOTICES; do
      if [ ! -s "$tmp/squashfs-root/usr/share/doc/mortar/$n" ]; then
        echo "$(basename "$f") has no usr/share/doc/mortar/$n" >&2
        fail=1
      fi
    done
    rm -rf "$tmp"
  done
}

case "$mode" in
  release)
    tag="${1:?$usage}"
    dir="${2:?$usage}"
    dl="$(mktemp -d)"
    trap 'rm -rf "$dl"' EXIT
    gh release download "$tag" --repo "$repo" --dir "$dl"
    for f in "$dir"/*; do
      name="$(basename "$f")"
      if [ ! -f "$dl/$name" ]; then
        echo "MISSING from release: $name" >&2
        fail=1
      elif ! cmp -s "$f" "$dl/$name"; then
        echo "DIFFERS from staged file: $name" >&2
        fail=1
      fi
    done
    check_manifest "$dl"
    check_appimages "$dl"
    summary="release $tag: $(find "$dir" -mindepth 1 -maxdepth 1 | wc -l) staged files present, manifest digests match"
    ;;
  staged)
    dir="${1:?$usage}"
    check_manifest "$dir"
    check_appimages "$dir"
    summary="staged files in $dir match their manifest"
    ;;
  public)
    tag="${1:?$usage}"
    base="https://github.com/$repo/releases/download/$tag"
    anon "$base/manifest.json"
    anon "https://github.com/$repo/releases/latest/download/manifest.json"
    while IFS= read -r url; do
      anon "$url"
    done < <(curl -fsSL "$base/manifest.json" 2>/dev/null | jq -r '.artifacts[].url' 2>/dev/null || true)
    summary="release $tag is reachable without a token"
    ;;
esac
check_components

if [ "$fail" != 0 ]; then
  echo "verification failed" >&2
  exit 1
fi
echo "verified: $summary"
