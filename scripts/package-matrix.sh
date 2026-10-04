#!/usr/bin/env bash
# Install, smoke-run and uninstall every Linux release package in throwaway containers.
# Usage: scripts/package-matrix.sh [tag]   (default: latest release; needs gh, docker or podman, flatpak for the bundle)
set -uo pipefail

TAG=${1:-}
REPO=Rethunk-AI/mortar
CT=$(command -v podman || command -v docker) || { echo "no podman or docker" >&2; exit 2; }
case $(uname -m) in x86_64) DEB=amd64 ARCH=x86_64 ;; aarch64) DEB=arm64 ARCH=aarch64 ;; *) echo "unsupported arch" >&2; exit 2 ;; esac
WORK=$(mktemp -d "${TMPDIR:-/var/tmp}/mortar-pkgmatrix.XXXXXX")
trap 'rm -rf "$WORK"' EXIT
A=$WORK/assets; mkdir -p "$A"
gh release download ${TAG:+"$TAG"} --repo "$REPO" -D "$A" \
  -p "*_${DEB}.deb" -p "*.${ARCH}.rpm" -p "*-${ARCH}.pkg.tar.zst" -p "*-${ARCH}.AppImage" -p "*-${ARCH}.flatpak" || exit 2
VERSION=$(basename "$A"/*.deb); VERSION=${VERSION#mortar_}; VERSION=${VERSION%%_*}

# Runs in the container with INSTALL and REMOVE set; the exit status is the verdict.
cat >"$WORK/inner.sh" <<'INNER'
FILES="/usr/bin/mortar /usr/share/applications/tech.rethunk.Mortar.desktop /usr/share/metainfo/tech.rethunk.Mortar.metainfo.xml
/usr/share/icons/hicolor/256x256/apps/tech.rethunk.Mortar.png /usr/share/icons/hicolor/scalable/apps/tech.rethunk.Mortar.svg /usr/share/mime/packages/mortar.xml"
fail() { echo "$*"; exit 1; }
sh -c "$INSTALL" >/tmp/install.log 2>&1 || { tail -5 /tmp/install.log; fail "install failed"; }
for f in $FILES; do [ -e "$f" ] || fail "missing after install: $f"; done
out=$(mortar version 2>&1) || fail "mortar version exit $?: $out"
case $out in *"$VERSION"*) ;; *) fail "unexpected version output: $out" ;; esac
grep -q '^Exec=mortar' /usr/share/applications/tech.rethunk.Mortar.desktop || fail "desktop Exec wrong"
sh -c "$REMOVE" >/tmp/remove.log 2>&1 || { tail -5 /tmp/remove.log; fail "uninstall failed"; }
# x-mortar.xml is the shared-mime-info trigger's cache; the distro, not the package, removes it.
left=$(find /usr /etc /root /var/lib -iname '*mortar*' ! -path '/usr/share/mime/application/x-mortar.xml' 2>/dev/null)
[ -z "$left" ] || fail "left after uninstall: $left"
INNER

RESULTS=(); FAIL=0
record() { RESULTS+=("$(printf '%-30s %-5s %s' "$1" "$2" "$3")"); [ "$2" = PASS ] || FAIL=1; }

pkgcase() { # label image install remove
  local out
  if out=$($CT run --rm -v "$A":/a:ro -v "$WORK/inner.sh":/inner.sh:ro -e VERSION="$VERSION" -e INSTALL="$3" -e REMOVE="$4" "$2" bash /inner.sh 2>&1); then
    record "$1" PASS ""
  else
    record "$1" FAIL "$(printf '%s' "$out" | tail -3 | tr '\n' ' ')"
  fi
}

DEB_I="apt-get update && apt-get install -y /a/*.deb"
for img in debian:stable ubuntu:latest; do
  pkgcase ".deb on $img" "$img" "$DEB_I" "apt-get purge -y mortar && apt-get autoremove -y"
done
pkgcase ".rpm on fedora:latest" fedora:latest "dnf install -y /a/*.rpm" "dnf remove -y mortar"
pkgcase ".pkg.tar.zst on archlinux:latest" archlinux:latest "pacman -Sy --noconfirm && pacman -U --noconfirm /a/*.pkg.tar.zst" "pacman -Rns --noconfirm mortar"

# AppImage: extract-and-run needs no FUSE; the extracted tree must carry the desktop entry, icon, licence and notices.
if out=$($CT run --rm -v "$A":/a:ro -e VERSION="$VERSION" ubuntu:latest bash -c '
  set -e
  cp /a/*.AppImage /tmp/m.AppImage && chmod +x /tmp/m.AppImage && cd /tmp
  out=$(./m.AppImage --appimage-extract-and-run version 2>&1) || { echo "version exit $?: $out"; exit 1; }
  case $out in *"$VERSION"*) ;; *) echo "unexpected: $out"; exit 1 ;; esac
  ./m.AppImage --appimage-extract >/dev/null
  ls squashfs-root/*.desktop squashfs-root/*.png >/dev/null || { echo "no desktop/icon in AppImage"; exit 1; }
  for n in LICENSE THIRD_PARTY_NOTICES; do [ -s squashfs-root/usr/share/doc/mortar/$n ] || { echo "no $n in AppImage"; exit 1; }; done
  [ -z "$(find /root -iname "*mortar*" 2>/dev/null)" ] || { echo "left in HOME"; exit 1; }' 2>&1); then
  record "AppImage on ubuntu:latest" PASS ""
else
  record "AppImage on ubuntu:latest" FAIL "$(printf '%s' "$out" | tail -3 | tr '\n' ' ')"
fi

# Flatpak needs the host (a container would have to pull the GNOME runtime and run bwrap nested); a throwaway
# FLATPAK_USER_DIR keeps the host's installations untouched.
if command -v flatpak >/dev/null; then
  export FLATPAK_USER_DIR=$WORK/flatpak
  APP=tech.rethunk.Mortar
  if out=$( { flatpak --user install -y --noninteractive --bundle "$A"/*.flatpak \
      && v=$(flatpak run "$APP" version 2>&1) && case $v in *"$VERSION"*) ;; *) echo "unexpected: $v"; false ;; esac \
      && for f in applications/$APP.desktop metainfo/$APP.metainfo.xml icons/hicolor/256x256/apps/$APP.png; do
           [ -e "$FLATPAK_USER_DIR/exports/share/$f" ] || { echo "missing export: $f"; false; }; done \
      && flatpak --user uninstall -y --noninteractive "$APP" \
      && [ -z "$(find "$FLATPAK_USER_DIR/exports" -iname '*mortar*' 2>/dev/null)" ]; } 2>&1); then
    record "flatpak bundle (host, temp dir)" PASS ""
  else
    record "flatpak bundle (host, temp dir)" FAIL "$(printf '%s' "$out" | tail -3 | tr '\n' ' ')"
  fi
else
  record "flatpak bundle" SKIP "flatpak not installed"
fi

printf '\nmortar %s package matrix\n' "$VERSION"
printf '%s\n' "${RESULTS[@]}"
exit $FAIL
