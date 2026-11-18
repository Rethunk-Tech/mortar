#!/usr/bin/env bash
# cursor-agent runs this in each fresh `--worktree` checkout, with ROOT_WORKTREE_PATH naming the checkout it was
# made from. Only what the gate cannot generate itself is missing there: node_modules (a worktree has none, and
# `bun run --cwd frontend catalogs` needs @lingui/cli before the gate's own bindings step can run) and the Wails
# bindings. frontend/dist is covered by the committed placeholder the embed needs.
set -euo pipefail
cd "$(dirname "$0")/.."

root=${ROOT_WORKTREE_PATH:-}
if [[ -z $root || ! -d $root ]]; then
  echo "setup-worktree: ROOT_WORKTREE_PATH must name the checkout this worktree was made from" >&2
  exit 1
fi

# bun installs the same lockfile in both trees, so the installed trees are linked rather than downloaded again.
for dir in node_modules frontend/node_modules web/node_modules; do
  if [[ -d $root/$dir && ! -e $dir ]]; then
    ln -s "$root/$dir" "$dir"
  fi
done

bun run bindings
