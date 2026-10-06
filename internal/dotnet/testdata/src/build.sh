#!/usr/bin/env bash
# Rebuilds ../mod.dll, the compiled fixture the reader's tests read, and ../e2e/*.dll, the ones the e2e specs install. The game stand-in is built only to compile against.
set -euo pipefail
cd "$(dirname "$0")"
out=$(mktemp -d "${TMPDIR:-/var/tmp}/dotnet-fixture.XXXXXX")
trap 'rm -rf "$out" Game/obj Mod/obj E2E/obj' EXIT
# Every dotnet invocation leaves an empty dir in TMPDIR, so it gets one that goes with $out.
mkdir "$out/tmp"
export TMPDIR="$out/tmp"
dotnet build Mod/Mod.csproj -c Release -o "$out" -p:Deterministic=true -p:DebugType=none -nologo -v quiet
cp "$out/Mod.dll" ../mod.dll
mkdir -p ../e2e
for name in Small Large Filler Alpha Beta; do
  dotnet build E2E/E2E.csproj -c Release -o "$out/$name" -p:Fixture="$name" -p:Deterministic=true -p:DebugType=none -nologo -v quiet
  cp "$out/$name/$name.dll" "../e2e/$name.dll"
done
