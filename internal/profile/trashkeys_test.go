package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestHistoryKeysCacheReadsTheOldShapeAndRewritesItCompact(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
	dir, err := e.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := readHistory(dir)
	if err != nil || len(data.Events) == 0 {
		t.Fatalf("history = %+v, %v", data, err)
	}
	old := map[string][]string{}
	for _, ev := range data.Events {
		old[ev.SnapshotID] = []string{"cached-only-key", "cached-only-key"}
	}
	b, _ := json.MarshalIndent(old, "", "  ")
	path := filepath.Join(dir, snapshotKeysFile)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	keys, err := historyStoreKeys(dir)
	if err != nil || !slices.Contains(keys, "cached-only-key") {
		t.Fatalf("old-shape cache not used: %v, %v", keys, err)
	}
	var cur snapshotKeys
	raw, _ := safefs.ReadFile(path)
	if err := json.Unmarshal(raw, &cur); err != nil || cur.Format != snapshotKeysFormat || len(cur.Snapshots) != len(old) {
		t.Fatalf("cache not rewritten in the new shape: %s", raw)
	}
	if n := slices.Index(cur.Keys, "cached-only-key"); n < 0 || slices.Index(cur.Keys[n+1:], "cached-only-key") >= 0 {
		t.Fatalf("union not deduplicated: %v", cur.Keys)
	}
}
