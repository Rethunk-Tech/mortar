package contentpatcher

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/testenv/packs"
)

// age backdates the files, and each folder's files, past the window in which a memo does not trust a stamp.
func age(t *testing.T, at time.Time, paths ...string) {
	t.Helper()
	for _, path := range paths {
		entries, _ := os.ReadDir(path)
		for _, e := range entries {
			if err := os.Chtimes(filepath.Join(path, e.Name()), at, at); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAnalyzeMemoServesUnchangedModsAndNoticesAnEditedLoadFile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	content := `{"Format":"2.0.0","Changes":[{"Action":"Load","Target":"Data/Thing","FromFile":"thing.json"}]}`
	a := packs.Disk(t, "A.Pack", map[string]string{"manifest.json": packs.Manifest("A.Pack"), "content.json": content, "thing.json": `{"a":1}`})
	b := packs.Disk(t, "B.Pack", map[string]string{"manifest.json": packs.Manifest("B.Pack"), "content.json": content, "thing.json": `{"b":2}`})
	age(t, time.Now().Add(-time.Hour), a.Folder, b.Folder)
	mods := []framework.Mod{a, b}
	in := framework.Input{Enabled: mods, All: mods}

	fresh := Driver{}.Analyze(in)
	if len(fresh.AssetConflicts) != 1 {
		t.Fatalf("conflicts = %#v, want the two loads of Data/Thing", fresh.AssetConflicts)
	}
	key := analysisKey(in)
	if _, ok := lookupAnalysis(key); !ok {
		t.Fatal("the check was not memoised")
	}
	resetContentPackCaches()
	if served := (Driver{}).Analyze(in); !reflect.DeepEqual(served, fresh) {
		t.Fatalf("memoised findings = %#v, want %#v", served, fresh)
	}

	// No pack stamp covers a loaded JSON file; the check's own read of it does. Same size, so only the time differs.
	thing := filepath.Join(b.Folder, "thing.json")
	if err := os.WriteFile(thing, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	age(t, time.Now().Add(-30*time.Minute), thing)
	resetContentPackCaches()
	if got := (Driver{}).Analyze(in); len(got.AssetConflicts) != 0 {
		t.Fatalf("identical loads still conflict after the edit: %#v", got.AssetConflicts)
	}
}

func TestAnalyzeMemoKeepsImageOverlapDepthsApart(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	sheet := func(x int) string {
		img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
		img.SetNRGBA(x, 0, color.NRGBA{A: 255})
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	content := `{"Format":"2.0.0","Changes":[{"Action":"EditImage","Target":"Maps/springobjects","FromFile":"p.png","ToArea":{"X":0,"Y":0,"Width":32,"Height":16},"PatchMode":"Overlay"}]}`
	a := packs.Disk(t, "A.Pack", map[string]string{"manifest.json": packs.Manifest("A.Pack"), "content.json": content, "p.png": sheet(0)})
	b := packs.Disk(t, "B.Pack", map[string]string{"manifest.json": packs.Manifest("B.Pack"), "content.json": content, "p.png": sheet(8)})
	age(t, time.Now().Add(-time.Hour), a.Folder, b.Folder)
	mods := []framework.Mod{a, b}

	exact := framework.Input{Enabled: mods, All: mods}
	skipped := framework.Input{Enabled: mods, All: mods, SkipImageOverlap: true}
	if analysisKey(exact) == analysisKey(skipped) {
		t.Fatal("one memo key serves both overlap depths")
	}
	cells := func(skip bool) string {
		return readContentPack(a, skip).patches[0].shapes[0].cells
	}
	for range 2 {
		if cells(false) == "" {
			t.Fatal("the full scan was served the skipped pack")
		}
		if cells(true) != "" {
			t.Fatal("the skipped scan was served the full pack")
		}
	}
}
