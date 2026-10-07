#!/usr/bin/env bash
# Fails when CURSEFORGE_API_KEY is set but the built binary does not contain it. The key is read from the
# environment, never argv, and compared as bytes: a key starting with `$` is not a safe grep pattern.
set -euo pipefail

[ -n "${CURSEFORGE_API_KEY:-}" ] || exit 0
python3 -I -c '
import os, sys
if os.environ["CURSEFORGE_API_KEY"].strip().encode() not in open(sys.argv[1], "rb").read():
    sys.exit(f"{sys.argv[1]}: CURSEFORGE_API_KEY is set but is not embedded in the binary")
' "$1"
