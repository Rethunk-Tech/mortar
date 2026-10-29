#!/usr/bin/env bash
# Rebuilds ../mod.dll and ../embedded.dll, the compiled mod other packages' tests read, and ../e2e/*.dll, the ones the e2e specs install. The sources are go-dotnet's game stand-in and mod, from the module version go.mod pins.
set -euo pipefail
cd "$(dirname "$0")"
repo=$(git rev-parse --show-toplevel)
out=$(mktemp -d "${TMPDIR:-/var/tmp}/dotnet-fixture.XXXXXX")
trap 'chmod -R u+w "$out" 2>/dev/null; rm -rf "$out" E2E/obj' EXIT
# Every dotnet invocation leaves an empty dir in TMPDIR, so it gets one that goes with $out.
mkdir "$out/tmp"
export TMPDIR="$out/tmp"
# The module cache is read-only and dotnet writes obj/ beside the project, so the stand-in and the mod are built from a copy.
module=$(go -C "$repo" mod download -json github.com/Rethunk-Tech/go-dotnet | jq -er .Dir)
cp -r "$module/testdata/src/Game" "$module/testdata/src/Mod" "$out/"
chmod -R u+w "$out/Game" "$out/Mod"
dotnet build "$out/Mod/Mod.csproj" -c Release -o "$out/mod" -p:Deterministic=true -p:DebugType=none -nologo -v quiet
cp "$out/mod/Mod.dll" ../mod.dll
dotnet build "$out/Mod/Mod.csproj" -c Release -o "$out/embedded" -p:Deterministic=true -p:DebugType=embedded -nologo -v quiet
cp "$out/embedded/Mod.dll" ../embedded.dll
mkdir -p ../e2e
for name in Small Large Filler Alpha Beta; do
  dotnet build E2E/E2E.csproj -c Release -o "$out/$name" -p:GameProject="$out/Game/Game.csproj" -p:Fixture="$name" -p:Deterministic=true -p:DebugType=none -nologo -v quiet
  cp "$out/$name/$name.dll" "../e2e/$name.dll"
done
