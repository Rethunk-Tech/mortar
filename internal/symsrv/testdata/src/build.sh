#!/usr/bin/env bash
# Rebuilds ../UnityPlayer.dll and its Windows PDB, the native module the symbol tests map addresses in, with clang-cl
# and lld's link flavour (rust-lld, which rustup ships, or lld-link).
set -euo pipefail
cd "$(dirname "$0")"
out=$(mktemp -d "${TMPDIR:-/var/tmp}/symsrv-fixture.XXXXXX")
trap 'rm -rf "$out"' EXIT
link=$(command -v lld-link || find "$HOME/.rustup/toolchains" -name rust-lld -path '*stable*' | head -n 1)
flavour=()
[[ $link == *rust-lld ]] && flavour=(-flavor link)
clang-cl --target=x86_64-pc-windows-msvc /c /Z7 /Od /GS- /Fo"$out/crash.obj" crash.cpp
"$link" "${flavour[@]}" /dll /debug /noentry /nodefaultlib /pdbaltpath:%_PDB% /out:"$out/UnityPlayer.dll" \
  /pdb:"$out/UnityPlayer_Win64_player_mono_x64.pdb" "$out/crash.obj"
cp "$out/UnityPlayer.dll" "$out/UnityPlayer_Win64_player_mono_x64.pdb" ..
