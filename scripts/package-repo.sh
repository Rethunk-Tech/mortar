#!/usr/bin/env bash
# Build the signed apt, dnf and pacman repository tree served at https://mortar.rethunk.tech/packages.
# Usage: MORTAR_REPO_KEY=<armored private key file> scripts/package-repo.sh ASSET_DIR OUT_DIR
#   ASSET_DIR holds a release's .deb, .rpm and .pkg.tar.zst files; OUT_DIR is created empty.
#   MORTAR_REPO_URL overrides the base URL written into the client snippets (a local test server).
# The metadata tools run in throwaway containers (podman, else docker); signing runs on the host with gpg in a temporary
# GNUPGHOME. The key must be the one in build/linux/repo/mortar-archive-keyring.asc, so a wrong secret cannot publish a
# repo that installed clients reject.
# RPMs are served unsigned, byte-identical to the release assets: dnf verifies the signed repomd.xml (repo_gpgcheck=1),
# which pins every package's sha256, so gpgcheck=0 loses nothing.
set -euo pipefail
ASSETS=$(cd "${1:?usage: package-repo.sh ASSET_DIR OUT_DIR}" && pwd)
OUT=${2:?usage: package-repo.sh ASSET_DIR OUT_DIR}
URL=${MORTAR_REPO_URL:-https://mortar.rethunk.tech/packages}
ROOT=$(cd "$(dirname "$0")/.." && pwd)
PUBKEY=$ROOT/build/linux/repo/mortar-archive-keyring.asc
: "${MORTAR_REPO_KEY:?MORTAR_REPO_KEY must name the armored private key file}"
CT=$(command -v podman || command -v docker) || {
  echo "no podman or docker" >&2
  exit 2
}

GNUPGHOME=$(mktemp -d "${TMPDIR:-/var/tmp}/mortar-repo-gpg.XXXXXX")
export GNUPGHOME
trap 'gpgconf --kill all >/dev/null 2>&1 || true; rm -rf "$GNUPGHOME"' EXIT
gpg --batch --quiet --import "$MORTAR_REPO_KEY" 2>/dev/null
FPR=$(gpg --with-colons --list-secret-keys | awk -F: '/^fpr/{print $10; exit}')
WANT=$(gpg --with-colons --show-keys "$PUBKEY" | awk -F: '/^fpr/{print $10; exit}')
if [ -z "$FPR" ] || [ "$FPR" != "$WANT" ]; then
  echo "signing key ${FPR:-none} is not $WANT from $PUBKEY" >&2
  exit 1
fi
sign() { gpg --batch --yes --local-user "$FPR" "$@"; }

[ ! -e "$OUT" ] || [ -z "$(ls -A "$OUT")" ] || {
  echo "$OUT is not empty" >&2
  exit 1
}
mkdir -p "$OUT"
OUT=$(cd "$OUT" && pwd)
# Rootful containers write as root, so the tree is handed back to the caller afterwards. Under rootless podman the
# container's root already is the caller, and a chown to the caller's id would map to a subordinate id instead.
handback="chown -R $(id -u):$(id -g) /out"
if [ "$("$CT" info --format '{{.Host.Security.Rootless}}' 2>/dev/null)" = true ]; then
  handback=true
fi
run() {
  {
    cat
    echo "$handback"
  } | $CT run --rm -i -v "$OUT":/out -w /out "$1" sh -es
}

shopt -s nullglob
debs=("$ASSETS"/*.deb) rpms=("$ASSETS"/*.rpm) pkgs=("$ASSETS"/*.pkg.tar.zst)
if [ ${#debs[@]} -eq 0 ] || [ ${#rpms[@]} -eq 0 ] || [ ${#pkgs[@]} -eq 0 ]; then
  echo "$ASSETS lacks .deb, .rpm or .pkg.tar.zst files" >&2
  exit 1
fi

# apt: one suite, one component, both architectures.
mkdir -p "$OUT/deb/pool/main/m/mortar"
cp "${debs[@]}" "$OUT/deb/pool/main/m/mortar/"
run debian:stable <<'EOF'
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq && apt-get install -y -qq apt-utils >/dev/null
cd deb
  for a in amd64 arm64; do
    d=dists/stable/main/binary-$a; mkdir -p $d
    apt-ftparchive --arch $a packages pool >$d/Packages; gzip -9kn $d/Packages
  done
  apt-ftparchive -o APT::FTPArchive::Release::Origin=Mortar -o APT::FTPArchive::Release::Label=Mortar \
    -o APT::FTPArchive::Release::Suite=stable -o APT::FTPArchive::Release::Codename=stable \
    -o APT::FTPArchive::Release::Architectures="amd64 arm64" -o APT::FTPArchive::Release::Components=main \
    release dists/stable >/tmp/Release && mv /tmp/Release dists/stable/Release
EOF
sign --armor --detach-sign -o "$OUT/deb/dists/stable/Release.gpg" "$OUT/deb/dists/stable/Release"
sign --clearsign -o "$OUT/deb/dists/stable/InRelease" "$OUT/deb/dists/stable/Release"

# dnf: one repo per $basearch.
for f in "${rpms[@]}"; do
  a=${f%.rpm}
  a=${a##*.}
  mkdir -p "$OUT/rpm/$a" && cp "$f" "$OUT/rpm/$a/"
done
run fedora:latest <<'EOF'
dnf install -y -q createrepo_c >/dev/null
for d in rpm/*/; do createrepo_c -q "$d"; done
EOF
for d in "$OUT"/rpm/*/repodata; do sign --armor --detach-sign -o "$d/repomd.xml.asc" "$d/repomd.xml"; done

# pacman: package signatures first, so repo-add embeds them in the database, then the database itself.
for f in "${pkgs[@]}"; do
  a=${f%.pkg.tar.zst}
  a=${a##*-}
  mkdir -p "$OUT/arch/$a" && cp "$f" "$OUT/arch/$a/"
  sign --detach-sign -o "$OUT/arch/$a/$(basename "$f").sig" "$OUT/arch/$a/$(basename "$f")"
done
run archlinux:latest <<'EOF'
for d in arch/*/; do (cd "$d" && repo-add -q mortar.db.tar.gz ./*.pkg.tar.zst); done
EOF
for d in "$OUT"/arch/*; do
  for f in mortar.db.tar.gz mortar.files.tar.gz; do sign --detach-sign -o "$d/$f.sig" "$d/$f"; done
  # repo-add's mortar.db and mortar.files are symlinks a static host may not follow.
  for f in mortar.db mortar.files; do
    rm -f "$d/$f" "$d/$f.sig"
    cp "$d/$f.tar.gz" "$d/$f"
    cp "$d/$f.tar.gz.sig" "$d/$f.sig"
  done
done

cp "$PUBKEY" "$OUT/mortar-archive-keyring.asc"
gpg --dearmor <"$PUBKEY" >"$OUT/mortar-archive-keyring.gpg"
echo "deb [signed-by=/usr/share/keyrings/mortar-archive-keyring.gpg] $URL/deb stable main" >"$OUT/mortar.list"
cat >"$OUT/mortar.repo" <<EOF
[mortar]
name=Mortar
baseurl=$URL/rpm/\$basearch
enabled=1
gpgcheck=0
repo_gpgcheck=1
gpgkey=$URL/mortar-archive-keyring.asc
EOF
cat >"$OUT/pacman.conf" <<EOF
[mortar]
SigLevel = Required
Server = $URL/arch/\$arch
EOF
echo "package repo for key $FPR in $OUT"
