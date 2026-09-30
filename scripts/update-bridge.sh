#!/usr/bin/env bash
# Vendors a release of Rethunk-AI/mortar-smapi-bridge and pins its version and sha256 in bridge.go.
set -euo pipefail
version=${1:?usage: scripts/update-bridge.sh <version>}
root=$(cd "$(dirname "$0")/.." && pwd)
dir=$root/internal/game/stardew
asset=MortarSmapiBridge-$version.zip

gh release download "v$version" -R Rethunk-AI/mortar-smapi-bridge -p "$asset" -D "$dir/vendor" --clobber
sha=$(sha256sum "$dir/vendor/$asset" | cut -d' ' -f1)
sed -i \
  -e "s|bridgeVersion = \".*\"|bridgeVersion = \"$version\"|" \
  -e "s|bridgeSHA256  = \".*\"|bridgeSHA256  = \"$sha\"|" \
  -e "s|//go:embed vendor/.*|//go:embed vendor/$asset|" \
  "$dir/bridge.go"
find "$dir/vendor" -name 'MortarSmapiBridge-*.zip' ! -name "$asset" -delete
echo "bridge $version sha256 $sha"
