#!/usr/bin/env bash
# Deletes a sandbox folder without reaching the user's own files through it.
#
# The hidden display's session bus starts desktop daemons (the document portal, gvfs) with the sandbox's run folder as
# their runtime directory, and they mount the user's real shared documents and remote shares there. A recursive delete
# through those mounts deletes the originals. So this stops every process whose runtime directory is inside the folder,
# unmounts whatever is mounted under it, deepest first, and deletes without crossing into a mount that is still there.
#
#   rm-sandbox.sh DIR            release DIR's daemons and mounts, then delete DIR
#   rm-sandbox.sh --release DIR  only release them (a sandbox kept for inspection stays safe to delete by hand)
set -euo pipefail

release_only=""
if [ "${1:-}" = --release ]; then
  release_only=1
  shift
fi
dir=$(realpath -e -- "${1:?usage: rm-sandbox.sh [--release] DIR}" 2>/dev/null) || exit 0

stopped=0
for env in /proc/[0-9]*/environ; do
  run=""
  { while IFS= read -r -d '' kv; do
    [[ $kv == XDG_RUNTIME_DIR=* ]] && run=${kv#XDG_RUNTIME_DIR=} && break
  done; } 2>/dev/null <"$env" || true
  if [[ -n $run && ($run == "$dir" || $run == "$dir"/*) ]]; then
    pid=${env#/proc/}
    kill "${pid%/environ}" 2>/dev/null && stopped=$((stopped + 1))
  fi
done

# Field 5 of mountinfo is the mount point, with a space written as \040.
mapfile -t mounts < <(awk -v d="$dir" '{ m = $5; gsub(/\\040/, " ", m); if (m == d || index(m, d "/") == 1) print m }' \
  /proc/self/mountinfo | sort -r)
for m in "${mounts[@]}"; do
  fusermount3 -u -z "$m" 2>/dev/null || umount -l "$m" 2>/dev/null || echo "rm-sandbox: could not unmount $m" >&2
done
echo "rm-sandbox: $dir: stopped $stopped processes, unmounted ${#mounts[@]} mounts" >&2

[ -n "$release_only" ] && exit 0
# A mount that would not go keeps its mount point and whatever is under it; the delete then fails rather than crossing it.
rm -rf --one-file-system -- "$dir"
