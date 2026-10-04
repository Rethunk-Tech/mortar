#!/usr/bin/env bash
# render.sh BIN_DIR OUT_DIR: fills tech.rethunk.Mortar.yml's @VERSION@ and @sha:PATH@ tokens (PATH is repo-relative,
# or under bin/ for a release asset found in BIN_DIR) and writes the manifest plus SUBMISSION.md to OUT_DIR.
set -euo pipefail
bin="${1:?usage: render.sh BIN_DIR OUT_DIR}"
out="${2:?usage: render.sh BIN_DIR OUT_DIR}"
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../../.." && pwd)"
version="$(cd "$root" && go run ./cmd/version -print)"
mkdir -p "$out"

cp "$here/tech.rethunk.Mortar.yml" "$out/tech.rethunk.Mortar.yml"
while IFS= read -r path; do
  file="$root/$path"
  case "$path" in bin/*) file="$bin/${path#bin/}" ;; esac
  sum="$(sha256sum "$file" | cut -d' ' -f1)"
  sed -i "s|@sha:$path@|$sum|" "$out/tech.rethunk.Mortar.yml"
done < <(grep -o '@sha:[^@]*@' "$here/tech.rethunk.Mortar.yml" | sed 's/^@sha://; s/@$//' | sort -u)
sed -i "s/@VERSION@/$version/g" "$out/tech.rethunk.Mortar.yml"
sed "s/@VERSION@/$version/g" "$here/SUBMISSION.md" >"$out/SUBMISSION.md"
