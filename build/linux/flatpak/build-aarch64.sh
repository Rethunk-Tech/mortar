#!/usr/bin/env bash
# Builds bin/mortar-linux-aarch64.flatpak on an x86_64 host without root.
# flatpak-builder runs build-commands through the aarch64 SDK's /bin/sh, so aarch64 ELFs must execute.
# Since Linux 6.7 a user namespace can mount its own binfmt_misc, so qemu-aarch64-static is registered
# there instead of system-wide; the F flag opens the interpreter at registration, so the nested build
# sandboxes run it without seeing the host's /usr/bin.
set -euo pipefail

root="$(cd "$(dirname "$0")/../../.." && pwd)"
tmp="$root/tmp"
cd "$root"
mkdir -p "$tmp"

# The outer namespace is root only to mount binfmt_misc; the inner one maps back to the caller's uid so
# flatpak and the build own their files as usual. Nested namespaces inherit the outer binfmt_misc.
if [[ "${1:-}" == --binfmt ]]; then
  mount -t binfmt_misc binfmt_misc /proc/sys/fs/binfmt_misc
  magic='\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\xb7\x00'
  mask='\xff\xff\xff\xff\xff\xff\xff\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff'
  printf ':qemu-aarch64:M::%s:%s:%s:F' "$magic" "$mask" "$4" >/proc/sys/fs/binfmt_misc/register
  if [[ -n "$5" ]]; then
    mount -t overlay overlay -o "lowerdir=$5/libexec:/usr/libexec" /usr/libexec
    export LD_LIBRARY_PATH="$5/lib64"
  fi
  exec unshare --user --map-user="$2" --map-group="$3" -- bash build/linux/cross-arm64.sh linux:flatpak ARCH=arm64
fi

# Tools the host lacks come from Fedora's packages, unpacked under tmp/ rather than installed.
unpacked() {
  local dir="$tmp/rpm/$1"
  if [[ ! -f "$dir/.done" ]]; then
    command -v dnf >/dev/null || { echo "Fedora packages $* (or their tools on PATH) are required" >&2; exit 1; }
    rm -rf "$dir"
    mkdir -p "$dir"
    (cd "$dir" && dnf download -q --arch=x86_64 --disablerepo='*' --enablerepo=fedora --enablerepo=updates "$@" >&2 </dev/null &&
      for rpm in *.rpm; do rpm2cpio "$rpm" | cpio -idm --quiet; done && touch .done)
  fi
  echo "$dir/usr"
}

qemu="$(command -v qemu-aarch64-static || echo "$(unpacked qemu-user-static-aarch64)/bin/qemu-aarch64-static")"
# The host's builder, not Flathub's sandboxed one: that one runs build-commands through the session helper
# outside this script's namespaces, where the binfmt entry does not exist.
builder="$(command -v flatpak-builder || echo "$(unpacked flatpak-builder)/bin/flatpak-builder")"
# appstreamcli looks for its compose plugin only at /usr/libexec, so an unpacked one is overlaid there.
compose=
[[ -x /usr/libexec/appstreamcli-compose ]] || compose="$(unpacked appstream-compose blake3)"
mkdir -p "$tmp/bin"
# setuid fusermount cannot mount rofiles inside a user namespace.
printf '#!/bin/sh\nexec %s --disable-rofiles-fuse "$@"\n' "$builder" >"$tmp/bin/flatpak-builder"
chmod +x "$tmp/bin/flatpak-builder"
export PATH="$tmp/bin:$PATH"
flatpak remote-add --user --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo

exec unshare --user --map-root-user --mount -- "$0" --binfmt "$(id -u)" "$(id -g)" "$qemu" "$compose"
