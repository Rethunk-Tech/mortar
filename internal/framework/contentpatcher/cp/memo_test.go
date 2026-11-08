package cp

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
	dataHome(t)
	resetCaches()
	t.Cleanup(resetCaches)

	content := `{"Format":"2.0.0","Changes":[{"Action":"Load","Target":"Data/Thing","FromFile":"thing.json"}]}`
	a := diskPack(t, "A.Pack", map[string]string{"manifest.json": packManifest("A.Pack"), "content.json": content, "thing.json": `{"a":1}`})
	b := diskPack(t, "B.Pack", map[string]string{"manifest.json": packManifest("B.Pack"), "content.json": content, "thing.json": `{"b":2}`})
	age(t, time.Now().Add(-time.Hour), a.Folder, b.Folder)
	mods := []Mod{a, b}
	in := Request{Enabled: mods, All: mods, Order: testOrder}

	fresh := analyze(in)
	if len(fresh.Conflicts) != 1 {
		t.Fatalf("conflicts = %#v, want the two loads of Data/Thing", fresh.Conflicts)
	}
	key := analysisKey(in)
	if _, ok := lookupAnalysis(key); !ok {
		t.Fatal("the check was not memoised")
	}
	resetCaches()
	if served := analyze(in); !reflect.DeepEqual(served, fresh) {
		t.Fatalf("memoised findings = %#v, want %#v", served, fresh)
	}

	// No pack stamp covers a loaded JSON file; the check's own read of it does. Same size, so only the time differs.
	thing := filepath.Join(b.Folder, "thing.json")
	if err := os.WriteFile(thing, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	age(t, time.Now().Add(-30*time.Minute), thing)
	resetCaches()
	if got := analyze(in); len(got.Conflicts) != 0 {
		t.Fatalf("identical loads still conflict after the edit: %#v", got.Conflicts)
	}
}

func TestAnalyzeMemoKeepsImageOverlapDepthsApart(t *testing.T) {
	dataHome(t)
	resetCaches()
	t.Cleanup(resetCaches)

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
	a := diskPack(t, "A.Pack", map[string]string{"manifest.json": packManifest("A.Pack"), "content.json": content, "p.png": sheet(0)})
	b := diskPack(t, "B.Pack", map[string]string{"manifest.json": packManifest("B.Pack"), "content.json": content, "p.png": sheet(8)})
	age(t, time.Now().Add(-time.Hour), a.Folder, b.Folder)
	mods := []Mod{a, b}

	exact := Request{Enabled: mods, All: mods}
	skipped := Request{Enabled: mods, All: mods, SkipImageOverlap: true}
	if analysisKey(exact) == analysisKey(skipped) {
		t.Fatal("one memo key serves both overlap depths")
	}
	cells := func(skip bool) string {
		return ReadPack(a, skip).Patches[0].Shapes[0].Cells
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
