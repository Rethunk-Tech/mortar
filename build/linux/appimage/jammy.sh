#!/usr/bin/env bash
# Builds the AppImage and the portable program beside it inside ubuntu:22.04 for the host's architecture, so both run on
# glibc 2.35 and newer. 22.04 has no WebKitGTK 6.0, so this is the GTK 3 / WebKitGTK 4.1 build (Wails tag gtk3).
# The host's Go toolchain, module and build caches, Bun and the AppImage tool cache are mounted in, and the build runs as
# the host user so nothing in the checkout or the caches ends up owned by root.
set -euo pipefail

if [ "${1:-}" != --inside ]; then
  root="$(git rev-parse --show-toplevel)"
  cache="${MORTAR_BUILD_CACHE:-${XDG_CACHE_HOME:-$HOME/.cache}/mortar-build}"
  mkdir -p "$cache" "$(go env GOCACHE)"
  exec docker run --rm \
    -v "$root:$root" -w "$root" \
    -v "$(go env GOROOT):/opt/go:ro" \
    -v "$(go env GOMODCACHE):/opt/gomod" \
    -v "$(go env GOCACHE):/opt/gocache" \
    -v "$cache:/opt/mortar-build" \
    -v "$(command -v bun):/usr/local/bin/bun:ro" \
    -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" \
    -e CURSEFORGE_API_KEY -e APPIMAGE_ARCH \
    ubuntu:22.04 bash build/linux/appimage/jammy.sh --inside
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq --no-install-recommends build-essential pkg-config git ca-certificates curl file \
  libgtk-3-dev libwebkit2gtk-4.1-dev >/dev/null

home="$(mktemp -d)"
chown "$HOST_UID:$HOST_GID" "$home"
as_user() {
  setpriv --reuid="$HOST_UID" --regid="$HOST_GID" --clear-groups env \
    HOME="$home" PATH="$home/bin:/opt/go/bin:$PATH" GOTOOLCHAIN=local GOMODCACHE=/opt/gomod GOCACHE=/opt/gocache \
    MORTAR_BUILD_CACHE=/opt/mortar-build APPIMAGE_EXTRACT_AND_RUN=1 "$@"
}
# The CLI's tasks need no cgo, and a cgo build would want GTK 4.
wails_dir="$(as_user go list -m -f '{{.Dir}}' github.com/wailsapp/wails/v3)"
as_user env --chdir="$wails_dir" CGO_ENABLED=0 go build -o "$home/bin/wails3" ./cmd/wails3
as_user wails3 task linux:create:appimage
