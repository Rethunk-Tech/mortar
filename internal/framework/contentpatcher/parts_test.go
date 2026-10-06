package contentpatcher

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/testenv/packs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// partsProfile is A and B loading Data/Thing, and A and C loading Data/Other: the same data, unless D is enabled.
func partsProfile(t *testing.T) []framework.Mod {
	t.Helper()
	load := func(changes ...string) string {
		return `{"Format":"2.0.0","Changes":[` + strings.Join(changes, ",") + `]}`
	}
	thing := `{"Action":"Load","Target":"Data/Thing","FromFile":"own.json"}`
	other := `{"Action":"Load","Target":"Data/Other","FromFile":"own.json"}`
	mods := []framework.Mod{
		packs.Disk(t, "A.Pack", map[string]string{"manifest.json": packs.Manifest("A.Pack"), "content.json": load(thing, other), "own.json": `{"a":1}`}),
		packs.Disk(t, "B.Pack", map[string]string{"manifest.json": packs.Manifest("B.Pack"), "content.json": load(thing), "own.json": `{"b":1}`}),
		packs.Disk(t, "C.Pack", map[string]string{"manifest.json": packs.Manifest("C.Pack"), "own.json": `{"c":1}`, "same.json": `{"a":1}`, "content.json": load(
			`{"Action":"Load","Target":"Data/Other","FromFile":"own.json","When":{"HasMod |contains=D.Pack":"true"}}`,
			`{"Action":"Load","Target":"Data/Other","FromFile":"same.json","When":{"HasMod |contains=D.Pack":"false"}}`,
		)}),
		packs.Disk(t, "D.Pack", map[string]string{"manifest.json": packs.Manifest("D.Pack"), "content.json": load(`{"Action":"Load","Target":"Data/Lone","FromFile":"own.json"}`), "own.json": `{}`}),
	}
	for _, m := range mods {
		age(t, time.Now().Add(-time.Hour), m.Folder)
	}
	return mods
}

func analyzeMods(mods []framework.Mod) framework.Findings {
	enabled := slices.DeleteFunc(slices.Clone(mods), func(m framework.Mod) bool { return !m.Enabled })
	return Driver{}.Analyze(framework.Input{Enabled: enabled, All: mods})
}

// incrementalMatchesFull checks mods after a change, then again with nothing kept, and requires the same findings
// and that the first check reused at least one part kept before the change.
func incrementalMatchesFull(t *testing.T, mods []framework.Mod, wantConflicts int) {
	t.Helper()
	parts.Lock()
	before := map[string]bool{}
	for key := range parts.entries {
		before[key] = true
	}
	parts.Unlock()
	got := analyzeMods(mods)
	parts.Lock()
	reused := 0
	for key, e := range parts.entries {
		if before[key] && e.Check == parts.check {
			reused++
		}
	}
	parts.Unlock()
	if reused == 0 {
		t.Fatal("the check after the change reused no part")
	}

	Driver{}.Forget()
	resetContentPackCaches()
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, "cache")); err != nil {
		t.Fatal(err)
	}
	want := analyzeMods(mods)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("incremental findings = %#v\nfull = %#v", got, want)
	}
	if len(want.AssetConflicts) != wantConflicts {
		t.Fatalf("conflicts = %#v, want %d", want.AssetConflicts, wantConflicts)
	}
}

func partsTest(t *testing.T) {
	t.Helper()
	testfs.DataHome(t)
	resetContentPackCaches()
	Driver{}.Forget()
	t.Cleanup(func() {
		resetContentPackCaches()
		Driver{}.Forget()
	})
}

func TestPartsAfterPackEdit(t *testing.T) {
	partsTest(t)
	mods := partsProfile(t)
	analyzeMods(mods)
	own := filepath.Join(mods[2].Folder, "own.json")
	if err := os.WriteFile(own, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	age(t, time.Now().Add(-30*time.Minute), own)
	// C now loads what A loads, so only Data/Thing still conflicts.
	incrementalMatchesFull(t, mods, 1)
}

func TestPartsAfterPackAdded(t *testing.T) {
	partsTest(t)
	mods := partsProfile(t)
	without := slices.Clone(mods)
	without[3].Enabled = false
	analyzeMods(without)
	// D's arrival changes what C loads, while the same two packs still load Data/Other.
	incrementalMatchesFull(t, mods, 2)
}

func TestPartsAfterPackRemoved(t *testing.T) {
	partsTest(t)
	mods := partsProfile(t)
	analyzeMods(mods)
	incrementalMatchesFull(t, slices.Delete(slices.Clone(mods), 1, 2), 1)
}
