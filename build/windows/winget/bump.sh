#!/usr/bin/env bash
# Usage: bump.sh VERSION [OUT] — writes the winget-pkgs manifests for VERSION into OUT/manifests/r/RethunkTech/Mortar/VERSION
# (OUT defaults to a temp dir; point it at a winget-pkgs checkout), hashes taken from the release's signed SHA256SUMS.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); repo=$(cd "$here/../../.." && pwd)
v=$1; out=${2:-$(mktemp -d)}; dst=$out/manifests/r/RethunkTech/Mortar/$v
old=$(awk '/^PackageVersion:/{print $2}' "$here/RethunkTech.Mortar.yaml")
base=https://github.com/Rethunk-Tech/mortar/releases/download/v$v
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
curl -fsSLo "$tmp/S" "$base/SHA256SUMS"; curl -fsSLo "$tmp/S.asc" "$base/SHA256SUMS.asc"
export GNUPGHOME=$tmp/g; install -d -m 700 "$GNUPGHOME"
gpg --batch --quiet --import "$repo/build/linux/repo/mortar-archive-keyring.asc"
gpg --batch --status-fd 1 --verify "$tmp/S.asc" "$tmp/S" 2>/dev/null | grep -q "VALIDSIG 3283604606CAE2295D476F9883BC8751EE6F773D"
h() { awk -v f="$1" '$2==f{print toupper($1)}' "$tmp/S"; }
amd=$(h mortar-amd64-installer.exe); arm=$(h mortar-arm64-installer.exe); [ -n "$amd" ] && [ -n "$arm" ]
mkdir -p "$dst"
for f in "$here"/*.yaml; do sed "s/${old//./\\.}/$v/g" "$f" >"$dst/$(basename "$f")"; done
sed -i "s/^ReleaseDate: .*/ReleaseDate: $(date -u +%F)/" "$dst/RethunkTech.Mortar.installer.yaml"
python3 - "$dst/RethunkTech.Mortar.installer.yaml" "$amd" "$arm" <<'PY'
import sys,re
p,amd,arm=sys.argv[1:]; s=open(p).read()
hs=iter([amd,arm]); s=re.sub(r'InstallerSha256: [0-9A-F]+', lambda m: 'InstallerSha256: '+next(hs), s)
open(p,'w').write(s)
PY
echo "$dst"; grep -E "InstallerUrl|InstallerSha256|PackageVersion" "$dst"/*.installer.yaml
