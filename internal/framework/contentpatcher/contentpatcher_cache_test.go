package contentpatcher

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/testenv/packs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

func TestContentPackDiskCache(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	im, extra, pngPath := diskCachePack(t)
	first := readContentPack(im)
	if len(first.patches) != 2 || first.patches[0].kind != "load" || first.patches[1].kind != "edit" {
		t.Fatalf("initial pack = %#v", first.patches)
	}
	flushPackDiskCache([]framework.Mod{im})
	cachePath := filepath.Join(dataHome, "mortar", "cache", "problems-content-packs.json")
	cache := readDiskPackCache(t, cachePath)
	entry, ok := cache.Packs[filepath.Clean(im.Folder)]
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
	hit := readContentPack(im)
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
	invalidatedJSON := readContentPack(im)
	if len(invalidatedJSON.patches) != 1 || invalidatedJSON.patches[0].kind != "edit" {
		t.Fatalf("included JSON invalidation = %#v", invalidatedJSON.patches)
	}
	flushPackDiskCache([]framework.Mod{im})

	img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	img.SetNRGBA(16, 0, color.NRGBA{A: 255})
	writePNG(t, pngPath, img)
	touchFile(t, pngPath)
	resetContentPackCaches()
	invalidatedPNG := readContentPack(im)
	if got := invalidatedPNG.patches[0].shapes[0].cells; got != ";1,0" {
		t.Fatalf("PNG invalidation cells = %q, want %q", got, ";1,0")
	}
	flushPackDiskCache([]framework.Mod{im})

	cache = readDiskPackCache(t, cachePath)
	entry = cache.Packs[filepath.Clean(im.Folder)]
	entry.Pack = diskCachedPack{}
	cache.Packs[filepath.Clean(im.Folder)] = entry
	cache.Version = contentPackParserVersion - 1
	raw, err := encodePackCache(cache)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	resetContentPackCaches()
	versionReset := readContentPack(im)
	if len(versionReset.patches) != 1 || versionReset.patches[0].kind != "edit" {
		t.Fatalf("parser version did not invalidate cache: %#v", versionReset.patches)
	}

	flushPackDiskCache(nil)
	cache = readDiskPackCache(t, cachePath)
	if _, ok := cache.Packs[filepath.Clean(im.Folder)]; !ok {
		t.Fatal("disk cache dropped a pack folder that still exists")
	}
	if err := os.RemoveAll(im.Folder); err != nil {
		t.Fatal(err)
	}
	flushPackDiskCache(nil)
	cache = readDiskPackCache(t, cachePath)
	if len(cache.Packs) != 0 {
		t.Fatalf("missing pack folder still cached = %#v", cache.Packs)
	}
}

func diskCachePack(t *testing.T) (framework.Mod, string, string) {
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
	writeFile("manifest.json", `{"UniqueID":"Test.Pack","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`)
	writeFile("content.json", `{"Changes":[{"Action":"Include","FromFile":"extra.json"},{"Action":"EditImage","Target":"Maps/Test","FromFile":"patch.png","PatchMode":"Overlay","ToArea":{"X":0,"Y":0}}]}`)
	extra := writeFile("extra.json", `{"Changes":[{"Action":"Load","Target":"Data/Test","FromFile":"load.json"}]}`)
	writeFile("load.json", `{"Value":1}`)
	img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	img.SetNRGBA(0, 0, color.NRGBA{A: 255})
	pngPath := filepath.Join(root, "patch.png")
	writePNG(t, pngPath, img)
	return packs.FromDisk(framework.Mod{Enabled: true, Folder: root, UniqueID: "Disk.Cache", Name: "Disk Cache", Key: "Disk.Cache"}), extra, pngPath
}

func TestReadContentPackConcurrent(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	mods := make([]framework.Mod, 8)
	for i := range mods {
		im, _, _ := diskCachePack(t)
		im.UniqueID = "Disk.Cache." + strconv.Itoa(i)
		im.Key = im.UniqueID
		mods[i] = im
	}
	preloadContentPacks(mods)
	var wg sync.WaitGroup
	got := make([]cachedPack, len(mods))
	for i, im := range mods {
		wg.Go(func() {
			got[i] = readContentPack(im)
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
	cache, ok := decodePackCache(raw)
	if !ok {
		t.Fatal("decode pack cache")
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
	cache, packed := generatedPackCache(t)
	if len(packed) >= 2<<20 {
		t.Fatalf("cache %d bytes for %d packs, want under 2MB", len(packed), len(cache.Packs))
	}
	got, ok := decodePackCache(packed)
	if !ok || !reflect.DeepEqual(got, cache) {
		t.Fatal("cache does not round-trip")
	}
}

// BenchmarkPackDiskCacheLoad decodes a cache of 40 packs with 30 image patches each.
func BenchmarkPackDiskCacheLoad(b *testing.B) {
	_, packed := generatedPackCache(b)
	b.SetBytes(int64(len(packed)))
	for b.Loop() {
		if _, ok := decodePackCache(packed); !ok {
			b.Fatal("decode")
		}
	}
}

func TestPNGAlphaMemoIsBitMaskAndDroppedAfterPack(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	im, _, pngPath := diskCachePack(t)
	info, err := os.Stat(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	fileKey := pngPath + "\x00" + strconv.FormatInt(info.Size(), 10) + "\x00" +
		strconv.FormatInt(info.ModTime().UnixNano(), 10)
	pix, ok := loadPNGAlpha(im.Folder, "patch.png", fileKey)
	if !ok {
		t.Fatal("loadPNGAlpha")
	}
	want := (32*16 + 7) / 8
	if len(pix.a) != want {
		t.Fatalf("mask bytes = %d, want %d (byte-per-pixel would be %d)", len(pix.a), want, 32*16)
	}

	readContentPack(im)
	n := 0
	pngAlphaCache.Range(func(_, _ any) bool {
		n++
		return true
	})
	if n != 0 {
		t.Fatalf("alpha cache entries after pack parse = %d", n)
	}
}

func TestPackMemoryCacheDropsFoldersNotInCurrentMods(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	im, _, _ := diskCachePack(t)
	_ = readContentPack(im)
	if _, ok := packCache.Load(filepath.Clean(im.Folder)); !ok {
		t.Fatal("pack was not memoized")
	}
	other := t.TempDir()
	flushPackDiskCache([]framework.Mod{{Folder: other}})
	if _, ok := packCache.Load(filepath.Clean(im.Folder)); ok {
		t.Fatal("packCache kept a folder that is not in the current mods list")
	}
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

func generatedPackCache(t testing.TB) (diskPackCache, []byte) {
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
	cache := diskPackCache{Version: contentPackParserVersion, Packs: map[string]diskPackEntry{}}
	for i := range packs {
		diskPatches := make([]diskPatch, patches)
		for p := range patches {
			when := diskWhen{Spouse: "abigail", Places: map[string][]string{"season": {"spring"}}}
			diskPatches[p] = diskPatchOf(cpPatch{
				kind: "edit", target: "maps/town", fromFile: "patch.png", patchMode: "Overlay",
				when: cpWhenOfDisk(when), image: true, imageFromArea: `{"X":0,"Y":0}`,
				source: "content.json", index: p, action: kindEditImage,
				shapes: []cpShape{{kind: 'r', cells: cellStr}},
			})
		}
		cache.Packs["/tmp/pack-"+strconv.Itoa(i)] = diskPackEntry{
			Fingerprint: "f" + strconv.Itoa(i),
			Files:       []packFileStamp{{Path: "content.json", Size: 100, ModTime: 1}},
			Pack:        diskCachedPack{Patches: diskPatches, Mentions: map[string]bool{"other.mod": true}, Skips: 1},
		}
	}
	packed, err := encodePackCache(cache)
	if err != nil {
		t.Fatal(err)
	}
	return cache, packed
}

func TestScanBenchConflictRSS(t *testing.T) {
	if os.Getenv("MORTAR_SCAN_BENCH") != "1" {
		t.Skip("set MORTAR_SCAN_BENCH=1 to measure a reflink copy at /var/tmp/scan-bench")
	}
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	var mods []framework.Mod
	var globErr, parseErr int
	packs, err := filepath.Glob("/var/tmp/scan-bench/*/mods/*")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, folder := range packs {
		info, err := os.Stat(folder)
		if err != nil || !info.IsDir() {
			globErr++
			continue
		}
		_ = filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return err
			}
			man, ok := contentPackManifest(path)
			if !ok {
				return nil
			}
			if man.UniqueID == "" || seen[man.UniqueID] {
				parseErr++
				return filepath.SkipDir
			}
			seen[man.UniqueID] = true
			mods = append(mods, framework.Mod{
				Key:      filepath.Base(folder) + ":" + man.UniqueID,
				Enabled:  true,
				Folder:   path,
				Manifest: man,
			})
			return filepath.SkipDir
		})
	}
	if len(mods) == 0 {
		t.Fatalf("no mods in /var/tmp/scan-bench (glob=%d stat=%d parse=%d)", len(packs), globErr, parseErr)
	}

	before := vmHWM()
	var peak atomic.Uint64
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		var ms runtime.MemStats
		for {
			select {
			case <-stop:
				return
			default:
				runtime.ReadMemStats(&ms)
				for {
					cur := peak.Load()
					if ms.HeapAlloc <= cur || peak.CompareAndSwap(cur, ms.HeapAlloc) {
						break
					}
				}
				time.Sleep(5 * time.Millisecond)
			}
		}
	})
	start := time.Now()
	conflicts, _ := assetConflictResults(mods)
	elapsed := time.Since(start)
	close(stop)
	wg.Wait()
	after := vmHWM()
	t.Logf("glob=%d mods=%d skippedStat=%d skippedParse=%d conflicts=%d elapsed=%s heapPeak=%d MiB vmHWM before=%d after=%d KiB",
		len(packs), len(mods), globErr, parseErr, len(conflicts), elapsed, peak.Load()>>20, before, after)
}

func vmHWM() int64 {
	raw, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for line := range strings.SplitSeq(string(raw), "\n") {
		if n, ok := strings.CutPrefix(line, "VmHWM:"); ok {
			n = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(n), "kB"))
			v, _ := strconv.ParseInt(n, 10, 64)
			return v
		}
	}
	return 0
}

func contentPackManifest(folder string) (manifest.Manifest, bool) {
	im := packs.FromDisk(framework.Mod{Folder: folder})
	if !isContentPatcherPack(im) {
		return manifest.Manifest{}, false
	}
	raw, err := fsx.ReadFile(filepath.Join(folder, manifest.FileName))
	if err != nil {
		return manifest.Manifest{}, false
	}
	man, err := manifest.Parse(raw)
	return man, err == nil
}

// A check reads switched-off packs for tilesheet cleanup, and a check of another profile's mods must not empty the
// cached packs of this one: either would leave a pack to be parsed again, or read back as empty, on the next start.
func TestPackDiskCacheKeepsEveryPackItRead(t *testing.T) {
	dataHome := testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	cachePath := filepath.Join(dataHome, "mortar", "cache", "problems-content-packs.json")
	on, _, _ := diskCachePack(t)
	off, _, _ := diskCachePack(t)
	off.Enabled = false
	// Only a pack that loads new tilesheets makes the cleanup pass read the others.
	sheets := t.TempDir()
	testfs.WriteFile(t, sheets, "manifest.json", `{"UniqueID":"Test.Sheets","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`)
	testfs.WriteFile(t, sheets, "content.json", `{"Changes":[{"Action":"Load","Target":"Tilesheets/Test","FromFile":"sheet.png"}]}`)
	candidate := packs.FromDisk(framework.Mod{Enabled: true, Folder: sheets, UniqueID: "Test.Sheets", Name: "Sheets", Key: "Test.Sheets"})
	Driver{}.Analyze(framework.Input{Enabled: []framework.Mod{on, candidate}, All: []framework.Mod{on, candidate, off}})
	if entry := readDiskPackCache(t, cachePath).Packs[filepath.Clean(off.Folder)]; len(entry.Pack.Patches) != 2 {
		t.Fatalf("switched-off pack on disk = %#v", entry.Pack)
	}

	resetContentPackCaches()
	_ = readContentPack(on)
	other, _, _ := diskCachePack(t)
	_ = readContentPack(other)
	flushPackDiskCache([]framework.Mod{other})
	_ = readContentPack(other)
	touchFile(t, filepath.Join(other.Folder, "content.json"))
	_ = readContentPack(other)
	flushPackDiskCache([]framework.Mod{other})
	resetContentPackCaches()
	if got := readContentPack(on); len(got.patches) != 2 {
		t.Fatalf("pack of the earlier check after two writes for other mods: %d patches, want 2", len(got.patches))
	}
}
