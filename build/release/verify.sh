#!/usr/bin/env bash
# Downloads every asset of release TAG and checks it against the staged files in DIR and against the signed
# manifest.json (size and sha512 digest, the algorithm `wails3 updater manifest` writes).
set -euo pipefail
tag="${1:?usage: verify.sh TAG DIR}"
dir="${2:?usage: verify.sh TAG DIR}"
dl="$(mktemp -d)"
trap 'rm -rf "$dl"' EXIT

gh release download "$tag" --repo "$GITHUB_REPOSITORY" --dir "$dl"

fail=0
for f in "$dir"/*; do
  name="$(basename "$f")"
  if [ ! -f "$dl/$name" ]; then
    echo "MISSING from release: $name" >&2
    fail=1
  elif ! cmp -s "$f" "$dl/$name"; then
    echo "DIFFERS from staged file: $name" >&2
    fail=1
  fi
done

while IFS=$'\t' read -r url size digest; do
  name="$(basename "$url")"
  if [ ! -f "$dl/$name" ]; then
    echo "manifest lists $name but the release has no such asset" >&2
    fail=1
    continue
  fi
  got_size="$(stat -c %s "$dl/$name")"
  got_digest="$(openssl dgst -sha512 -binary "$dl/$name" | base64 -w0)"
  if [ "$got_size" != "$size" ] || [ "$got_digest" != "$digest" ]; then
    echo "manifest mismatch for $name (size $got_size vs $size)" >&2
    fail=1
  fi
done < <(jq -r '.artifacts[] | [.url, .size, .digest] | @tsv' "$dl/manifest.json")

if [ "$fail" != 0 ]; then
  echo "release $tag failed verification" >&2
  exit 1
fi
echo "release $tag verified: $(find "$dir" -mindepth 1 -maxdepth 1 | wc -l) staged files present, manifest digests match"
