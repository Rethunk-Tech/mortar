#!/usr/bin/env bash
# Rebuilds ../mod.dll, the compiled fixture the reader's tests read. The game stand-in is built only to compile against.
set -euo pipefail
cd "$(dirname "$0")"
out=$(mktemp -d "${TMPDIR:-/var/tmp}/dotnet-fixture.XXXXXX")
trap 'rm -rf "$out" Game/obj Mod/obj' EXIT
dotnet build Mod/Mod.csproj -c Release -o "$out" -p:Deterministic=true -p:DebugType=none -nologo -v quiet
cp "$out/Mod.dll" ../mod.dll
