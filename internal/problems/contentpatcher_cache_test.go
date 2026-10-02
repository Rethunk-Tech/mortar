package problems

import (
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func TestContentPackDiskCache(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	mod, extra, pngPath := diskCachePack(t)
	first := readContentPack(mod)
	if len(first.patches) != 2 || first.patches[0].kind != "load" || first.patches[1].kind != "edit" {
		t.Fatalf("initial pack = %#v", first.patches)
	}
	flushPackDiskCache([]Installed{mod})
	cachePath := filepath.Join(dataHome, "mortar", "cache", "problems-content-packs.json")
	cache := readDiskPackCache(t, cachePath)
	entry, ok := cache.Packs[filepath.Clean(mod.Folder)]
	if !ok || cache.Version != contentPackParserVersion {
		t.Fatalf("disk cache = %#v", cache)
	}
	for _, want := range []string{"content.json", "extra.json", "patch.png"} {
		found := false
		for _, file := range entry.Files {
			found = found || file.Path == want
		}
		if !found {
			t.Fatalf("disk cache files = %#v, missing %q", entry.Files, want)
		}
	}

	resetContentPackCaches()
	hit := readContentPack(mod)
	if len(hit.patches) != len(first.patches) || hit.patches[1].shapes[0].cells != first.patches[1].shapes[0].cells {
		t.Fatalf("disk cache hit = %#v, want %#v", hit.patches, first.patches)
	}

	if err := os.WriteFile(extra, []byte(`{"Changes":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	touchFile(t, extra)
	resetContentPackCaches()
	invalidatedJSON := readContentPack(mod)
	if len(invalidatedJSON.patches) != 1 || invalidatedJSON.patches[0].kind != "edit" {
		t.Fatalf("included JSON invalidation = %#v", invalidatedJSON.patches)
	}
	flushPackDiskCache([]Installed{mod})

	img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	img.SetNRGBA(16, 0, color.NRGBA{A: 255})
	writePNG(t, pngPath, img)
	touchFile(t, pngPath)
	resetContentPackCaches()
	invalidatedPNG := readContentPack(mod)
	if got := invalidatedPNG.patches[0].shapes[0].cells; got != ";1,0" {
		t.Fatalf("PNG invalidation cells = %q, want %q", got, ";1,0")
	}
	flushPackDiskCache([]Installed{mod})

	cache = readDiskPackCache(t, cachePath)
	entry = cache.Packs[filepath.Clean(mod.Folder)]
	entry.Pack = diskCachedPack{}
	cache.Packs[filepath.Clean(mod.Folder)] = entry
	cache.Version = contentPackParserVersion - 1
	raw, err := json.Marshal(cache)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	resetContentPackCaches()
	versionReset := readContentPack(mod)
	if len(versionReset.patches) != 1 || versionReset.patches[0].kind != "edit" {
		t.Fatalf("parser version did not invalidate cache: %#v", versionReset.patches)
	}

	flushPackDiskCache(nil)
	cache = readDiskPackCache(t, cachePath)
	if len(cache.Packs) != 0 {
		t.Fatalf("stale pack cache entries = %#v", cache.Packs)
	}
}

func diskCachePack(t *testing.T) (Installed, string, string) {
	t.Helper()
	root := t.TempDir()
	writeFile := func(name, contents string) string {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	writeFile("manifest.json", `{"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`)
	writeFile("content.json", `{"Changes":[{"Action":"Include","FromFile":"extra.json"},{"Action":"EditImage","Target":"Maps/Test","FromFile":"patch.png","PatchMode":"Overlay","ToArea":{"X":0,"Y":0}}]}`)
	extra := writeFile("extra.json", `{"Changes":[{"Action":"Load","Target":"Data/Test","FromFile":"load.json"}]}`)
	writeFile("load.json", `{"Value":1}`)
	img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	img.SetNRGBA(0, 0, color.NRGBA{A: 255})
	pngPath := filepath.Join(root, "patch.png")
	writePNG(t, pngPath, img)
	return Installed{Enabled: true, Folder: root, UniqueID: "Disk.Cache", Name: "Disk Cache", Key: "Disk.Cache"}, extra, pngPath
}

func readDiskPackCache(t *testing.T, path string) diskPackCache {
	t.Helper()
	raw, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cache diskPackCache
	if err := json.Unmarshal(raw, &cache); err != nil {
		t.Fatal(err)
	}
	return cache
}

func resetContentPackCaches() {
	packCache = sync.Map{}
	pngShapeCache = sync.Map{}
	mapCache = sync.Map{}
	packDiskState.Lock()
	packDiskState.loaded = false
	packDiskState.path = ""
	packDiskState.entries = nil
	packDiskState.dirty = false
	packDiskState.Unlock()
}

func touchFile(t *testing.T, path string) {
	t.Helper()
	stamp := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
}
