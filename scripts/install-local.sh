#!/usr/bin/env bash
# Builds the committed HEAD and installs it over the maintainer's Mortar (GearLever AppImage), after snapshotting the data folder.
#
#   scripts/install-local.sh [--windows]   --windows also builds the Windows installer and zips it into ~/Downloads
#
# Refuses a dirty tree, since the build stamps HEAD. Needs the CurseForge key in the keyring (secret-tool, service curseforge).
# Once the installed build reports HEAD, superseded /var/tmp/mortar-pre-* snapshots are deleted; older installer zips are listed.
set -euo pipefail

REPO=$(cd "$(dirname "$0")/.." && pwd)
cd "$REPO"
windows=0
[ "${1:-}" = "--windows" ] && windows=1

if [ -n "$(git status --porcelain)" ]; then
  echo "the tree is dirty; commit first so the build matches HEAD" >&2
  git status --short >&2
  exit 1
fi
sha=$(git rev-parse --short=8 HEAD)
log=${TMPDIR:-/var/tmp}/mortar-install-$sha.log

key=$(/usr/bin/secret-tool lookup service curseforge) || true
if [ -z "$key" ]; then
  echo "no CurseForge key in the keyring (secret-tool lookup service curseforge)" >&2
  exit 1
fi
export CURSEFORGE_API_KEY=$key GOTMPDIR=/var/tmp TMPDIR=/var/tmp

snap=/var/tmp/mortar-pre-$sha-$(date +%s)
cp -a --reflink=auto "$HOME/.local/share/mortar" "$snap"
echo "data snapshot: $snap"

capped() { systemd-run --user --scope -q -p MemoryMax=24G -p MemorySwapMax=0 -E CURSEFORGE_API_KEY -E GOTMPDIR -E TMPDIR -- "$@"; }
echo "building (log: $log)"
bun install --frozen-lockfile >"$log" 2>&1
capped wails3 task package >>"$log" 2>&1
if [ "$windows" = 1 ]; then
  capped wails3 task windows:package >>"$log" 2>&1
  zip -j -q "$HOME/Downloads/mortar-$(git describe --tags --abbrev=0 --match 'v[0-9]*' | sed 's/^v//')-$sha-windows-amd64-installer.zip" bin/mortar-amd64-installer.exe
fi

# A running copy keeps its old code until it restarts, so it is closed before the swap and the launch at the end
# opens this build. Exit 3 is "not running"; anything else means a game or downloads kept it open.
quit=0
[ -x "$HOME/Apps/mortar.appimage" ] && { "$HOME/Apps/mortar.appimage" quit >/dev/null 2>&1 || quit=$?; }
[ "$quit" = 0 ] || [ "$quit" = 3 ] || echo "Mortar stayed open (a game or downloads are running): restart it to load $sha" >&2

cp bin/mortar-linux-x86_64.AppImage "$HOME/Downloads/mortar.appimage"
flatpak run it.mijorus.gearlever --remove "$HOME/Apps/mortar.appimage" <<<y >/dev/null 2>&1 || true
flatpak run it.mijorus.gearlever --integrate "$HOME/Downloads/mortar.appimage" <<<y >/dev/null 2>&1
installed=$("$HOME/Apps/mortar.appimage" version)
echo "installed: $installed"
case "$installed" in
  *"$sha"*) ;;
  *)
    echo "the installed build does not report $sha" >&2
    exit 1
    ;;
esac

for old in /var/tmp/mortar-pre-*; do
  [ "$old" != "$snap" ] && [ -d "$old" ] || continue
  rm -rf -- "$old"
  echo "deleted superseded snapshot: $old"
done
if [ "$windows" = 1 ]; then
  echo "superseded installer zips:"
  for old in "$HOME"/Downloads/mortar-*-windows-amd64-installer.zip; do
    case "$old" in *"-$sha-"*) ;; *) [ -e "$old" ] && echo "  $old" ;; esac
  done
fi
gtk-launch mortar >/dev/null 2>&1 || true
