package problems

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	if hit.patches[1].imageDigest == "" || hit.patches[1].imageDigest != first.patches[1].imageDigest {
		t.Fatalf("image digest was not restored from the disk cache")
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

func TestReadContentPackConcurrent(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	mods := make([]Installed, 8)
	for i := range mods {
		mod, _, _ := diskCachePack(t)
		mod.UniqueID = "Disk.Cache." + strconv.Itoa(i)
		mod.Key = mod.UniqueID
		mods[i] = mod
	}
	preloadContentPacks(mods)
	var wg sync.WaitGroup
	got := make([]cachedPack, len(mods))
	for i, mod := range mods {
		wg.Go(func() {
			got[i] = readContentPack(mod)
		})
	}
	wg.Wait()
	for i, pack := range got {
		if len(pack.patches) != 2 || pack.patches[1].imageDigest == "" || pack.patches[1].shapes[0].cells == "" {
			t.Fatalf("pack %d = %#v", i, pack.patches)
		}
	}
}

func readDiskPackCache(t *testing.T, path string) diskPackCache {
	t.Helper()
	raw, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 2 || raw[0] != 0x1f || raw[1] != 0x8b {
		t.Fatalf("cache file is not gzip")
	}
	payload, ok := unzipPackCache(raw)
	if !ok {
		t.Fatal("unzip pack cache")
	}
	if bytes.Contains(payload, []byte("imageSource")) {
		t.Fatalf("cache still stores imageSource")
	}
	var cache diskPackCache
	if err := json.Unmarshal(payload, &cache); err != nil {
		t.Fatal(err)
	}
	return cache
}

func TestEncodeCellSet(t *testing.T) {
	cases := []string{"", ";0,0", ";1,0", ";0,0;1,0;2,0;0,1"}
	for _, cells := range cases {
		got := decodeCellSet(encodeCellSet(cells))
		if got != cells {
			t.Fatalf("cells %q round-trip as %q", cells, got)
		}
	}
	sparse := ";0,0;500,400"
	if decodeCellSet(encodeCellSet(sparse)) != sparse {
		t.Fatalf("sparse cells")
	}
}

func TestPackDiskCacheSizeAndLoad(t *testing.T) {
	oldJSON, compactJSON, packed := generatedPackCacheBytes(t)
	if len(packed) >= 20<<20 {
		t.Fatalf("gzip cache %d bytes, want under 20MB", len(packed))
	}
	if len(packed) >= len(oldJSON)/4 {
		t.Fatalf("gzip cache %d bytes vs old JSON %d", len(packed), len(oldJSON))
	}
	var cache diskPackCache
	payload, ok := unzipPackCache(packed)
	if !ok || json.Unmarshal(payload, &cache) != nil || cache.Version != contentPackParserVersion {
		t.Fatalf("compact load failed")
	}
	if len(payload) != len(compactJSON) {
		t.Fatalf("unzip size %d, compact JSON %d", len(payload), len(compactJSON))
	}
	t.Logf("fixture packs=%d oldJSON=%d compactJSON=%d gzip=%d", len(cache.Packs), len(oldJSON), len(compactJSON), len(packed))
}

func BenchmarkPackDiskCacheLoad(b *testing.B) {
	oldJSON, _, packed := generatedPackCacheBytes(b)
	b.Run("oldJSON", func(b *testing.B) {
		b.SetBytes(int64(len(oldJSON)))
		for b.Loop() {
			var cache oldDiskPackCache
			if json.Unmarshal(oldJSON, &cache) != nil {
				b.Fatal("unmarshal old")
			}
		}
	})
	b.Run("compactGzip", func(b *testing.B) {
		b.SetBytes(int64(len(packed)))
		for b.Loop() {
			payload, ok := unzipPackCache(packed)
			if !ok {
				b.Fatal("unzip")
			}
			var cache diskPackCache
			if json.Unmarshal(payload, &cache) != nil {
				b.Fatal("unmarshal compact")
			}
		}
	})
}

func resetContentPackCaches() {
	packCache = sync.Map{}
	pngShapeCache = sync.Map{}
	pngAlphaCache = sync.Map{}
	cellSetCache = sync.Map{}
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

type oldDiskPackCache struct {
	Version int                         `json:"version"`
	Packs   map[string]oldDiskPackEntry `json:"packs"`
}

type oldDiskPackEntry struct {
	Fingerprint string            `json:"fingerprint"`
	Files       []packFileStamp   `json:"files"`
	Pack        oldDiskCachedPack `json:"pack"`
}

type oldDiskCachedPack struct {
	Patches  []oldDiskPatch        `json:"patches"`
	Tokens   []diskTokenDefinition `json:"tokens"`
	Mentions map[string]bool       `json:"mentions"`
	Schema   map[string]diskSchema `json:"schema"`
	Skips    int                   `json:"skips"`
}

type oldDiskPatch struct {
	Kind          string              `json:"kind"`
	Target        string              `json:"target"`
	FromFile      string              `json:"fromFile"`
	Priority      string              `json:"priority"`
	PatchMode     string              `json:"patchMode"`
	When          diskWhen            `json:"when"`
	Shapes        []oldDiskShape      `json:"shapes"`
	Spouse        string              `json:"spouse"`
	Places        map[string][]string `json:"places"`
	Image         bool                `json:"image"`
	ImageSource   []byte              `json:"imageSource"`
	ImageFromArea string              `json:"imageFromArea"`
	Source        string              `json:"source"`
	Index         int                 `json:"index"`
	Action        string              `json:"action"`
	ToArea        string              `json:"toArea"`
	TokenName     string              `json:"tokenName"`
	TokenValue    string              `json:"tokenValue"`
}

type oldDiskShape struct {
	Kind  byte   `json:"kind"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	W     int    `json:"w"`
	H     int    `json:"h"`
	Cells string `json:"cells"`
	Layer string `json:"layer"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Tiny  bool   `json:"tiny"`
}

func generatedPackCacheBytes(t testing.TB) (oldJSON, compactJSON, packed []byte) {
	t.Helper()
	const packs = 40
	const patches = 30
	cells := strings.Builder{}
	for y := range 20 {
		for x := range 20 {
			cells.WriteByte(';')
			cells.WriteString(strconv.Itoa(x))
			cells.WriteByte(',')
			cells.WriteString(strconv.Itoa(y))
		}
	}
	cellStr := cells.String()
	png := bytes.Repeat([]byte{0x89, 0x50, 0x4e, 0x47}, 2048)
	old := oldDiskPackCache{Version: 6, Packs: map[string]oldDiskPackEntry{}}
	compact := diskPackCache{Version: contentPackParserVersion, Packs: map[string]diskPackEntry{}}
	for i := range packs {
		root := "/tmp/pack-" + strconv.Itoa(i)
		oldPatches := make([]oldDiskPatch, patches)
		newPatches := make([]diskPatch, patches)
		for p := range patches {
			when := diskWhen{Spouse: "abigail", Places: map[string][]string{"season": {"spring"}}}
			oldPatches[p] = oldDiskPatch{
				Kind: "edit", Target: "maps/town", FromFile: "patch.png", PatchMode: "Overlay",
				When: when, Spouse: "abigail", Places: when.Places, Image: true, ImageSource: png,
				ImageFromArea: `{"X":0,"Y":0}`, Source: "content.json", Index: p, Action: kindEditImage,
				Shapes: []oldDiskShape{{Kind: 'r', Cells: cellStr}},
			}
			cp := cpPatch{
				kind: "edit", target: "maps/town", fromFile: "patch.png", patchMode: "Overlay",
				when: cpWhenOfDisk(when), image: true, imageFromArea: `{"X":0,"Y":0}`,
				source: "content.json", index: p, action: kindEditImage,
				shapes: []cpShape{{kind: 'r', cells: cellStr}},
			}
			newPatches[p] = diskPatchOf(cp)
		}
		old.Packs[root] = oldDiskPackEntry{
			Fingerprint: "f" + strconv.Itoa(i),
			Files:       []packFileStamp{{Path: "content.json", Size: 100, ModTime: 1}},
			Pack:        oldDiskCachedPack{Patches: oldPatches, Mentions: map[string]bool{"other.mod": true}, Skips: 1},
		}
		compact.Packs[root] = diskPackEntry{
			Fingerprint: "f" + strconv.Itoa(i),
			Files:       []packFileStamp{{Path: "content.json", Size: 100, ModTime: 1}},
			Pack:        diskCachedPack{Patches: newPatches, Mentions: map[string]bool{"other.mod": true}, Skips: 1},
		}
	}
	var err error
	oldJSON, err = json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	compactJSON, err = json.Marshal(compact)
	if err != nil {
		t.Fatal(err)
	}
	packed, err = zipPackCache(compactJSON)
	if err != nil {
		t.Fatal(err)
	}
	if tb, ok := t.(*testing.T); ok {
		var sample oldDiskCachedPack
		for _, entry := range old.Packs {
			sample = entry.Pack
			break
		}
		sizeOf := func(v any) int { raw, _ := json.Marshal(v); return len(raw) }
		tb.Logf("per-pack old JSON: patches=%d tokens=%d mentions=%d schema=%d (imageSource+cells dominate patches)",
			sizeOf(sample.Patches), sizeOf(sample.Tokens), sizeOf(sample.Mentions), sizeOf(sample.Schema))
	}
	return oldJSON, compactJSON, packed
}
