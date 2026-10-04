#!/usr/bin/env bash
# render.sh OUT_DIR [--local]: writes the Flathub submission for build/config.yml's version to OUT_DIR:
# tech.rethunk.Mortar.yml (the sideload manifest's header and finish-args above this directory's modules, building
# the v<version> tag from source), go-sources.json and node-sources.json (every Go module and npm package the build
# reads, as proxy.golang.org and registry.npmjs.org files), yarn.lock and SUBMISSION.md.
# --local builds this checkout's HEAD commit instead of the tag on GitHub, for a test build.
# Needs network, Go, bun, jq and uvx.
set -euo pipefail
usage="usage: render.sh OUT_DIR [--local]"
out="${1:?$usage}"
mode="${2:-}"
if [ $# -gt 2 ] || { [ -n "$mode" ] && [ "$mode" != --local ]; }; then
  echo "$usage" >&2
  exit 2
fi
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../../.." && pwd)"
version="$(sed -n 's/^  version: "\([^"]*\)".*/\1/p' "$root/build/config.yml")"
node_generator="git+https://github.com/flatpak/flatpak-builder-tools@74697c75b630d7330e77250fc13cb5ea688d9479#subdirectory=node"
mkdir -p "$out"
work="$(mktemp -d "${TMPDIR:-/var/tmp}/mortar-flathub.XXXXXX")"
cleanup() {
  case "$work" in
    */mortar-flathub.*) chmod -R u+w "$work" && rm -rf "$work" ;;
  esac
}
trap cleanup EXIT

if [ "$mode" = --local ]; then
  commit="$(git -C "$root" rev-parse HEAD)"
  source_edit="s|url: https://github.com/Rethunk-AI/mortar.git|url: file://$root|; /^        tag: v/d"
else
  # A dry run renders before the tag exists; its manifest names the commit the tag will point at.
  commit="$(git -C "$root" rev-parse "v$version^{commit}" 2>/dev/null || git -C "$root" rev-parse HEAD)"
  source_edit=""
fi
{
  sed '/^modules:/,$d' "$root/build/linux/flatpak/tech.rethunk.Mortar.yml"
  sed -n '/^sdk-extensions:/,$p' "$here/tech.rethunk.Mortar.yml"
} | sed -e "s/@VERSION@/$version/g; s/@COMMIT@/$commit/g" -e "$source_edit" >"$out/tech.rethunk.Mortar.yml"
sed "s/@VERSION@/$version/g" "$here/SUBMISSION.md" >"$out/SUBMISSION.md"

# The module cache a clean build of Mortar and the Wails CLI fills, for both Flathub architectures, is exactly the
# set of files the offline build's GOPROXY=file:// has to serve, less the toolchain switch's own download and Mortar's
# own module (the git source).
gomod="$work/gomod"
(
  cd "$root"
  export GOMODCACHE="$gomod" GOPROXY=https://proxy.golang.org GOPRIVATE='' GONOPROXY='' GONOSUMDB='' GOFLAGS='' CGO_ENABLED=1
  wails_dir="$(go mod download -json github.com/wailsapp/wails/v3 | jq -r .Dir)"
  for arch in amd64 arm64; do
    GOARCH=$arch go list -e -deps -tags production . >/dev/null
    GOARCH=$arch go list -e -deps . >/dev/null
    (cd "$wails_dir" && GOARCH=$arch go list -e -deps ./cmd/wails3 >/dev/null)
  done
)
(
  cd "$gomod/cache/download"
  find . -path ./golang.org/toolchain -prune -o -path ./sumdb -prune -o -path './github.com/!rethunk-!a!i/mortar' -prune -o -type f \( -name '*.mod' -o -name '*.zip' -o -name '*.info' \) -print | sort |
    while IFS= read -r f; do
      f="${f#./}"
      # Go rewrites .info files as it caches them; the proxy's own copy is what the build downloads.
      case "$f" in
        *.info) sum="$(curl -fsSL "https://proxy.golang.org/$f" | sha256sum | cut -d' ' -f1)" ;;
        *) sum="$(sha256sum "$f" | cut -d' ' -f1)" ;;
      esac
      jq -n --arg f "$f" --arg sum "$sum" \
        '{type: "file", url: ("https://proxy.golang.org/" + $f), sha256: $sum, dest: ("goproxy/" + ($f | sub("/[^/]*$"; ""))), "dest-filename": ($f | sub(".*/"; ""))}'
    done
) | jq -s . >"$out/go-sources.json"

# flatpak-node-generator reads npm, yarn and pnpm lockfiles, not bun's, so bun writes the same resolution as a yarn v1
# lockfile. Yarn has no lock entry for a workspace, and the build runs no install scripts, so only the package
# tarballs are kept (the generator also adds Playwright's browsers and node headers for scripts).
mkdir -p "$work/node/frontend"
cp "$root/package.json" "$root/bun.lock" "$work/node/"
cp "$root/frontend/package.json" "$work/node/frontend/"
(cd "$work/node" && bun install --frozen-lockfile --ignore-scripts --yarn >/dev/null)
awk 'BEGIN { RS = ""; ORS = "\n\n" } !/resolved "workspace:/' "$work/node/yarn.lock" >"$out/yarn.lock"
uvx --from "$node_generator" flatpak-node-generator yarn "$out/yarn.lock" -o "$work/node-sources.json" >/dev/null
jq '[.[] | select(.type == "file")]' "$work/node-sources.json" >"$out/node-sources.json"
