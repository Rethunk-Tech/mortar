#!/usr/bin/env bash
# render.sh BIN_DIR OUT_DIR: writes OUT_DIR/PKGBUILD with the real sha256sums of the release assets in BIN_DIR
# (and of the tagged source archive) and OUT_DIR/.SRCINFO beside it.
set -euo pipefail
bin="${1:?usage: render.sh BIN_DIR OUT_DIR}"
out="${2:?usage: render.sh BIN_DIR OUT_DIR}"
here="$(cd "$(dirname "$0")" && pwd)"
mkdir -p "$out"

pkgver="$(sed -n "s/^pkgver=//p" "$here/PKGBUILD")"
src="$(mktemp)"
trap 'rm -f "$src"' EXIT
# The AUR installs from the public archive URL, which a private repo does not serve; the tarball then stays SKIP
# and the other sources still get real sums.
src_sum=SKIP
if curl -fsSL "https://github.com/${GITHUB_REPOSITORY:-Rethunk-Tech/mortar}/archive/refs/tags/v${pkgver}.tar.gz" -o "$src"; then
  src_sum="$(sha256sum "$src" | cut -d' ' -f1)"
else
  echo "source archive not publicly downloadable; leaving its sha256sum as SKIP" >&2
fi

sum() { sha256sum "$1" | cut -d' ' -f1; }
sed -e "s/^sha256sums=.*/sha256sums=('$src_sum')/" \
    -e "s/^sha256sums_x86_64=.*/sha256sums_x86_64=('$(sum "$bin/mortar-aur-linux-amd64")')/" \
    -e "s/^sha256sums_aarch64=.*/sha256sums_aarch64=('$(sum "$bin/mortar-aur-linux-arm64")')/" \
    "$here/PKGBUILD" >"$out/PKGBUILD"
python3 "$here/srcinfo.py" "$out/PKGBUILD" >"$out/.SRCINFO"
