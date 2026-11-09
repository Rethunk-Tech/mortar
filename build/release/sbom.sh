#!/usr/bin/env bash
# sbom.sh OUT writes a CycloneDX SBOM of the Go module and the bun workspaces (frontend and web) from go.mod and
# bun.lock, so it needs no installed dependencies. syft is pinned by checksum; a different version means new sums.
set -euo pipefail
out="$(realpath -m "${1:?usage: sbom.sh OUT}")"
root="$(cd "$(dirname "$0")/../.." && pwd)"
version="1.57.0"
case "$(uname -m)" in
  x86_64) arch=amd64 sum=9c077a7d43029b3b13e98a8db720661e700c4d58db6352153a06395ae4a2a704 ;;
  aarch64) arch=arm64 sum=410cbfa0ae0f63246bc0057843760b1de2833cd23919e025e368fd7681308cdb ;;
  *) echo "unsupported architecture $(uname -m)" >&2; exit 1 ;;
esac
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
tgz="$tmp/syft.tar.gz"
curl -fsSL --retry 18 --retry-delay 10 -o "$tgz" "https://github.com/anchore/syft/releases/download/v$version/syft_${version}_linux_${arch}.tar.gz"
echo "$sum  $tgz" | sha256sum -c - >/dev/null
tar -xzf "$tgz" -C "$tmp" syft

# Only the manifests and lockfiles are scanned: a built tree would add node_modules and test fixtures that never ship.
src="$tmp/src"
mkdir "$src"
(cd "$root" && git ls-files -z go.mod go.sum bun.lock package.json frontend/package.json web/package.json | xargs -0 cp --parents -t "$src")
app_version="$(cd "$root" && go run ./cmd/version -print)"
(cd "$src" && SYFT_CHECK_FOR_APP_UPDATE=false "$tmp/syft" scan dir:. --source-name mortar --source-version "$app_version" -o "cyclonedx-json=$out" -q)
jq -e '[.components[].purl // empty | select(startswith("pkg:golang/"))] | length > 0' "$out" >/dev/null
jq -e '[.components[].purl // empty | select(startswith("pkg:npm/"))] | length > 0' "$out" >/dev/null
