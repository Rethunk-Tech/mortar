#!/usr/bin/env bash
# Cross-compile linux/arm64 against an Ubuntu 24.04 arm64 sysroot extracted from
# .deb files. The debs are fetched in an amd64 container (no binfmt, no sudo).
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"
tmp="$root/tmp"
sysroot="$tmp/linux-arm64-sysroot"
debs="$tmp/linux-arm64-debs"
image="ubuntu:24.04"

mkdir -p "$tmp" bin

if ! command -v zig >/dev/null; then
  echo "zig is required (zig cc -target aarch64-linux-gnu.2.39)" >&2
  exit 1
fi
if ! command -v docker >/dev/null; then
  echo "docker is required to fetch arm64 .debs" >&2
  exit 1
fi
if ! command -v nfpm >/dev/null; then
  echo "nfpm is required (go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.47.0)" >&2
  exit 1
fi

pkgs="libgtk-4-dev:arm64 libwebkitgtk-6.0-dev:arm64 shared-mime-info iso-codes"
stamp="$sysroot/.stamp"
stamp_ver=4
if [[ ! -f "$stamp" ]] || [[ "$(cat "$stamp")" != "$stamp_ver" ]] || [[ ! -f "$debs/.packages" ]] || [[ "$(cat "$debs/.packages")" != "$pkgs" ]]; then
  rm -rf "$sysroot"
  mkdir -p "$sysroot" "$debs"
  if [[ ! -f "$debs/.packages" ]] || [[ "$(cat "$debs/.packages")" != "$pkgs" ]]; then
    docker pull --platform linux/amd64 "$image"
    docker run --rm --platform linux/amd64 \
      -v "$debs:/debs" \
      -e pkgs="$pkgs" \
      "$image" \
      bash -euo pipefail -c '
        dpkg --add-architecture arm64
        cat >/etc/apt/sources.list.d/ubuntu.sources <<EOF
Types: deb
URIs: http://archive.ubuntu.com/ubuntu
Suites: noble noble-updates noble-backports
Components: main universe restricted multiverse
Architectures: amd64
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg

Types: deb
URIs: http://security.ubuntu.com/ubuntu
Suites: noble-security
Components: main universe restricted multiverse
Architectures: amd64
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg

Types: deb
URIs: http://ports.ubuntu.com/ubuntu-ports
Suites: noble noble-updates noble-backports noble-security
Components: main universe restricted multiverse
Architectures: arm64
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg
EOF
        apt-get update -qq
        apt-get install --download-only -y --no-install-recommends $pkgs
        rm -rf /debs/*
        find /var/cache/apt/archives -maxdepth 1 -name '*.deb' ! -name '*_amd64.deb' -exec cp -t /debs {} +
        cd /debs
        apt-get download shared-mime-info iso-codes
      '
    printf '%s\n' "$pkgs" >"$debs/.packages"
  fi
  shopt -s nullglob
  for deb in "$debs"/*_arm64.deb "$debs"/*_all.deb "$debs"/shared-mime-info_*.deb; do
    dpkg-deb -x "$deb" "$sysroot"
  done
  printf '%s\n' "$stamp_ver" >"$stamp"
fi
# Ubuntu's usrmerge /lib symlink comes from base-files, which is not extracted, so the
# ELF interpreter path /lib/ld-linux-aarch64.so.1 needs its own link for qemu -L.
[[ -e "$sysroot/lib/ld-linux-aarch64.so.1" ]] || ln -s ../usr/lib/ld-linux-aarch64.so.1 "$sysroot/lib/ld-linux-aarch64.so.1"

export GOOS=linux
export GOARCH=arm64
export CGO_ENABLED=1
export PKG_CONFIG_SYSROOT_DIR="$sysroot"
export PKG_CONFIG_LIBDIR="$sysroot/usr/lib/aarch64-linux-gnu/pkgconfig:$sysroot/usr/share/pkgconfig"
export PKG_CONFIG_PATH=""
# zig ships the target glibc itself; giving it --sysroot as well makes it prefix the
# sysroot onto pkg-config's already-prefixed -L paths and find no GTK libraries.
export CC="zig cc -target aarch64-linux-gnu.2.39"
export CXX="zig c++ -target aarch64-linux-gnu.2.39"
export CGO_LDFLAGS="-L$sysroot/usr/lib/aarch64-linux-gnu -Wl,-rpath-link,$sysroot/usr/lib/aarch64-linux-gnu -Wl,-rpath-link,$sysroot/lib/aarch64-linux-gnu"

if ! pkg-config --exists gtk4 webkitgtk-6.0; then
  pkg-config --print-errors --exists gtk4 webkitgtk-6.0
  exit 1
fi

if [[ ! -f "$root/frontend/dist/index.html" ]]; then
  echo "frontend/dist is missing; build the UI first (bun run bindings && bun run --cwd frontend build)" >&2
  exit 1
fi

cd "$root"
# The packaged build, nfpm packages and bin/mortar-aur-linux-arm64 come from the same task the native build uses;
# the exported CC/CGO env above makes it cross-compile.
wails3 task linux:nfpm ARCH=arm64 NFPM_ARCH=arm64

file bin/mortar-aur-linux-arm64
# No --version flag; the dynamic linker exiting after a load trace is the non-GUI smoke.
qemu-aarch64 -L "$sysroot" -E LD_TRACE_LOADED_OBJECTS=1 ./bin/mortar-aur-linux-arm64

missing=0
while read -r lib; do
  [[ -z "$lib" ]] && continue
  if [[ ! -e "$sysroot/lib/aarch64-linux-gnu/$lib" && ! -e "$sysroot/usr/lib/aarch64-linux-gnu/$lib" && ! -e "$sysroot/lib/$lib" ]]; then
    case "$lib" in
      ld-linux-aarch64.so.1|libc.so.6|libm.so.6|libpthread.so.0|libdl.so.2|librt.so.1|libresolv.so.2)
        ;;
      *)
        echo "NEEDED $lib not in sysroot" >&2
        missing=1
        ;;
    esac
  fi
done < <(readelf -d bin/mortar-aur-linux-arm64 | sed -n 's/.*NEEDED.*\[\(.*\)\]/\1/p')
if [[ "$missing" -ne 0 ]]; then
  exit 1
fi
