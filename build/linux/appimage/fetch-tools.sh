#!/usr/bin/env bash
# Usage: fetch-tools.sh ARCH. Puts the pinned linuxdeploy and type2 runtime for ARCH (x86_64, aarch64) in the cache and
# prints the cache directory. A file that is present and matches its pinned SHA-256 is never fetched again, so a warm
# cache builds offline.
set -euo pipefail
arch="$1"
here="$(cd "$(dirname "$0")" && pwd)"
cache="${MORTAR_BUILD_CACHE:-${XDG_CACHE_HOME:-$HOME/.cache}/mortar-build}/appimage"
mkdir -p "$cache"
while read -r name sha url; do
  case "$name" in "#"* | "") continue ;; esac
  [ "${name##*-}" = "$arch" ] || continue
  dest="$cache/$name"
  if [ -f "$dest" ] && echo "$sha  $dest" | sha256sum -c --status -; then continue; fi
  part="$(mktemp "$dest.XXXXXX")"
  trap 'rm -f "$part"' EXIT
  curl -fsSL --retry 3 -o "$part" "$url"
  echo "$sha  $part" | sha256sum -c --status - || { echo "$name: SHA-256 mismatch for $url" >&2; exit 1; }
  chmod 755 "$part"
  mv "$part" "$dest"
  trap - EXIT
done <"$here/pins.txt"
echo "$cache"
