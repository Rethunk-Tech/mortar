#!/usr/bin/env bash
# catalog-compat.sh [MANIFEST]: decodes MANIFEST (default: the bundled catalog) with the internal/components package
# of every supported Mortar release (v0.3.8 on, AGENTS.md § Brownfield), so a catalog change that would leave one on
# its bundled copy (as v0.3.0 was by The Sims 4's entry) is caught before it is signed. It also fails when a release
# would skip a game it read from the newest published catalog, as v0.3.8 to v0.7.0 skipped Darkest Dungeon once its
# linuxMarker held a folder. Needs network (gh) for the tags, their modules and the published catalog.
set -euo pipefail
cd "$(dirname "$0")/.."
manifest=$(realpath "${1:-internal/components/components.json}")
mapfile -t tags < <(git ls-remote --tags origin 'v*' | awk '{print $2}' | sed -n 's|^refs/tags/\(v[0-9][^^]*\)$|\1|p' | sort -V | awk '!seen && $0 == "v0.3.8" {seen=1} seen')
work=$(mktemp -d "${TMPDIR:-/var/tmp}/mortar-compat-XXXXXX")
trap 'rm -rf "$work"' EXIT
latest=$(gh release list --limit 50 --json tagName --jq '[.[] | select(.tagName | startswith("components-v2-"))][0].tagName')
gh release download "$latest" --pattern components.json --output "$work/published.json"
failed=0
for tag in "${tags[@]}"; do
  git rev-parse -q --verify "refs/tags/$tag" >/dev/null || git fetch -q --depth=1 origin "refs/tags/$tag:refs/tags/$tag"
  dir=$work/$tag
  mkdir -p "$dir"
  git archive "$tag" | tar -x -m -C "$dir"
  cat >"$dir/internal/components/compat_check_test.go" <<'GO'
package components

import (
	"encoding/json"
	"os"
	"testing"
)

func compatRead(t *testing.T, env string) ([]byte, map[string]bool) {
	b, err := os.ReadFile(os.Getenv(env))
	if err != nil {
		t.Fatal(err)
	}
	m, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, g := range m.Games {
		ids[g.ID] = true
	}
	return b, ids
}

func TestCompatDecode(t *testing.T) {
	b, read := compatRead(t, "COMPAT_MANIFEST")
	_, before := compatRead(t, "COMPAT_PUBLISHED")
	var raw struct {
		Games []struct {
			ID string `json:"id"`
		} `json:"games"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	for _, g := range raw.Games {
		if before[g.ID] && !read[g.ID] {
			t.Errorf("skips game %q, which it reads from the published catalog", g.ID)
		}
	}
	t.Logf("reads %d games", len(read))
}
GO
  if (cd "$dir" && COMPAT_MANIFEST=$manifest COMPAT_PUBLISHED=$work/published.json go test -count=1 -run '^TestCompatDecode$' -v ./internal/components) >"$work/$tag.log" 2>&1; then
    echo "$tag: $(grep -o 'reads [0-9]* games' "$work/$tag.log")"
  else
    echo "$tag: refuses the catalog:"
    grep -E 'compat_check_test.go|FAIL|error' "$work/$tag.log" | head -10
    failed=1
  fi
done
exit "$failed"
