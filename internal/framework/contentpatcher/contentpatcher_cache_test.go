package contentpatcher

import (
	"bytes"
	"compress/gzip"
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

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/testenv/packs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/go-tilemask"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
)

// dataFolder is the data folder the test's environment resolves to: mortar on Linux, Mortar on Windows.
func dataFolder(t *testing.T) string {
	t.Helper()
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestContentPackDiskCache(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	im, extra, pngPath := diskCachePack(t)
	first := readContentPack(im, false)
	if len(first.patches) != 2 || first.patches[0].kind != "load" || first.patches[1].kind != "edit" {
		t.Fatalf("initial pack = %#v", first.patches)
	}
	flushPackDiskCache([]framework.Mod{im})
	cachePath := filepath.Join(dataFolder(t), "cache", "problems-content-packs.json")
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
	hit := readContentPack(im, false)
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
	invalidatedJSON := readContentPack(im, false)
	if len(invalidatedJSON.patches) != 1 || invalidatedJSON.patches[0].kind != "edit" {
		t.Fatalf("included JSON invalidation = %#v", invalidatedJSON.patches)
	}
	flushPackDiskCache([]framework.Mod{im})

	img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	img.SetNRGBA(16, 0, color.NRGBA{A: 255})
	writePNG(t, pngPath, img)
	touchFile(t, pngPath)
	resetContentPackCaches()
	invalidatedPNG := readContentPack(im, false)
	if got := invalidatedPNG.patches[0].shapes[0].cells; got != ";1,0" {
		t.Fatalf("PNG invalidation cells = %q, want %q", got, ";1,0")
	}
	flushPackDiskCache([]framework.Mod{im})

	cache = readDiskPackCache(t, cachePath)
	entry = cache.Packs[filepath.Clean(im.Folder)]
	entry.Pack = nil
	cache.Packs[filepath.Clean(im.Folder)] = entry
	cache.Version = contentPackParserVersion - 1
	var raw bytes.Buffer
	if err := writePackCache(&raw, cache); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePath, raw.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	resetContentPackCaches()
	versionReset := readContentPack(im, false)
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
	preloadContentPacks(mods, false)
	var wg sync.WaitGroup
	got := make([]cachedPack, len(mods))
	for i, im := range mods {
		wg.Go(func() {
			got[i] = readContentPack(im, false)
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
	raw, err := safefs.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cache, ok := readPackCache(bytes.NewReader(raw))
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
	got, ok := readPackCache(bytes.NewReader(packed))
	if !ok || !reflect.DeepEqual(got, cache) {
		t.Fatal("cache does not round-trip")
	}
}

// BenchmarkPackDiskCacheLoad decodes a cache of 40 packs with 30 image patches each.
func BenchmarkPackDiskCacheLoad(b *testing.B) {
	_, packed := generatedPackCache(b)
	b.SetBytes(int64(len(packed)))
	for b.Loop() {
		cache, ok := readPackCache(bytes.NewReader(packed))
		if !ok {
			b.Fatal("decode")
		}
		for _, entry := range cache.Packs {
			if _, ok := decodeDiskPack(entry.Pack); !ok {
				b.Fatal("decode pack")
			}
		}
	}
}

func TestPNGAlphaMemoIsBitMaskAndDroppedAfterPack(t *testing.T) {
	testfs.DataHome(t)
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
	if pix.Width != 32 || pix.Height != 16 {
		t.Fatalf("mask = %d x %d, want 32 x 16", pix.Width, pix.Height)
	}

	readContentPack(im, false)
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
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	im, _, _ := diskCachePack(t)
	_ = readContentPack(im, false)
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
	// The data folder too: a later test would otherwise write its caches into this test's removed temp folder.
	packDiskState.base = ""
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
		pack, err := encodeDiskPack(diskCachedPack{Patches: diskPatches, Mentions: map[string]bool{"other.mod": true}, Skips: 1})
		if err != nil {
			t.Fatal(err)
		}
		cache.Packs["/tmp/pack-"+strconv.Itoa(i)] = diskPackEntry{
			Fingerprint: "f" + strconv.Itoa(i),
			Files:       []packFileStamp{{Path: "content.json", Size: 100, ModTime: 1}},
			Pack:        pack,
		}
	}
	var packed bytes.Buffer
	if err := writePackCache(&packed, cache); err != nil {
		t.Fatal(err)
	}
	return cache, packed.Bytes()
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

func contentPackManifest(folder string) (smapimanifest.Manifest, bool) {
	im := packs.FromDisk(framework.Mod{Folder: folder})
	if !isContentPatcherPack(im) {
		return smapimanifest.Manifest{}, false
	}
	raw, err := safefs.ReadFile(filepath.Join(folder, smapimanifest.FileName))
	if err != nil {
		return smapimanifest.Manifest{}, false
	}
	man, err := smapimanifest.Parse(raw)
	return man, err == nil
}

// A check reads switched-off packs for tilesheet cleanup, and a check of another profile's mods must not empty the
// cached packs of this one: either would leave a pack to be parsed again, or read back as empty, on the next start.
func TestPackDiskCacheKeepsEveryPackItRead(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	cachePath := filepath.Join(dataFolder(t), "cache", "problems-content-packs.json")
	on, _, _ := diskCachePack(t)
	off, _, _ := diskCachePack(t)
	off.Enabled = false
	// Only a pack that loads new tilesheets makes the cleanup pass read the others.
	sheets := t.TempDir()
	testfs.WriteFile(t, sheets, "manifest.json", `{"UniqueID":"Test.Sheets","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`)
	testfs.WriteFile(t, sheets, "content.json", `{"Changes":[{"Action":"Load","Target":"Tilesheets/Test","FromFile":"sheet.png"}]}`)
	candidate := packs.FromDisk(framework.Mod{Enabled: true, Folder: sheets, UniqueID: "Test.Sheets", Name: "Sheets", Key: "Test.Sheets"})
	Driver{}.Analyze(framework.Input{Enabled: []framework.Mod{on, candidate}, All: []framework.Mod{on, candidate, off}})
	if pack, _ := decodeDiskPack(readDiskPackCache(t, cachePath).Packs[filepath.Clean(off.Folder)].Pack); len(pack.Patches) != 2 {
		t.Fatalf("switched-off pack on disk = %#v", pack)
	}

	resetContentPackCaches()
	_ = readContentPack(on, false)
	other, _, _ := diskCachePack(t)
	_ = readContentPack(other, false)
	flushPackDiskCache([]framework.Mod{other})
	_ = readContentPack(other, false)
	touchFile(t, filepath.Join(other.Folder, "content.json"))
	_ = readContentPack(other, false)
	flushPackDiskCache([]framework.Mod{other})
	resetContentPackCaches()
	if got := readContentPack(on, false); len(got.patches) != 2 {
		t.Fatalf("pack of the earlier check after two writes for other mods: %d patches, want 2", len(got.patches))
	}
}

// A pack file written while the pack is read keeps its stamp from before the read, so the fingerprint no longer
// matches and the stale read is not cached.
func TestPackFileChangedDuringReadInvalidatesFingerprint(t *testing.T) {
	root := t.TempDir()
	abs := testfs.WriteFile(t, root, "content.json", `{"Changes":[]}`)
	var pack cachedPack
	pack.recordPackFile(root, abs)
	testfs.WriteFile(t, root, "content.json", `{"Changes":[{"Action":"Load"}]}`)
	pack.recordPackFile(root, abs)
	if packFingerprintValid(root, pack.files, packFilesFingerprint(pack.files)) {
		t.Fatal("a file changed after its first stamp still validates")
	}
}

func TestPackCacheNoticesAConfigThatAppearsLater(t *testing.T) {
	root := t.TempDir()
	var pack cachedPack
	pack.recordPackFile(root, filepath.Join(root, "config.json"))
	pack.fingerprint = packFilesFingerprint(pack.files)
	if !packFingerprintValid(root, pack.files, pack.fingerprint) {
		t.Fatal("a pack without a config.json must stay valid while it has none")
	}
	if err := os.WriteFile(filepath.Join(root, "config.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if packFingerprintValid(root, pack.files, pack.fingerprint) {
		t.Fatal("the config.json that appeared was not noticed")
	}
}

// A data folder that was moved or copied keeps its pack cache: packs are keyed by their path under the data folder.
func TestPackDiskCacheSurvivesAMovedDataFolder(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	im, _, _ := diskCachePack(t)
	inside := filepath.Join(dataFolder(t), "profiles", "p", "mods", "pack")
	if err := os.MkdirAll(filepath.Dir(inside), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(im.Folder, inside); err != nil {
		t.Fatal(err)
	}
	im.Folder = inside
	readContentPack(im, false)
	flushPackDiskCache([]framework.Mod{im})

	oldData := dataFolder(t)
	newHome := t.TempDir()
	datadirtest.Use(t, newHome)
	newData, err := datadir.DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldData, newData); err != nil {
		t.Fatal(err)
	}
	im.Folder = filepath.Join(newData, "profiles", "p", "mods", "pack")

	// Same size and mtime, different text: only a cache hit still reports the old patches.
	content := filepath.Join(im.Folder, "content.json")
	info, err := os.Stat(content)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(content, []byte(strings.Repeat(" ", int(info.Size()))), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(content, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	resetContentPackCaches()
	if got := readContentPack(im, false); len(got.patches) != 2 {
		t.Fatalf("a moved data folder missed the pack cache: %d patches", len(got.patches))
	}
}

// An unchanged image's alpha mask comes from the disk cache, not from decoding the PNG again.
func TestOverlayAlphaMaskIsKeptOnDisk(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	loadPackDiskCache()

	im, _, pngPath := diskCachePack(t)
	key, ok := pngFileKey(im.Folder, "patch.png")
	if !ok {
		t.Fatal("pngFileKey")
	}
	first, ok := loadPNGAlpha(im.Folder, "patch.png", key)
	if !ok {
		t.Fatal("loadPNGAlpha")
	}
	want := maskBytes(t, first)

	// The same size and mtime with bytes no PNG decoder accepts: only the disk cache can still answer.
	info, err := os.Stat(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pngPath, make([]byte, info.Size()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(pngPath, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	pngAlphaCache = sync.Map{}
	got, ok := loadPNGAlpha(im.Folder, "patch.png", key)
	if !ok || got.Width != first.Width || got.Height != first.Height || !bytes.Equal(maskBytes(t, got), want) {
		t.Fatalf("mask after a restart differs from the decoded one (ok=%v)", ok)
	}
}

func TestOverlayAlphaMaskWithBadChecksumIsNotServed(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	loadPackDiskCache()

	im, _, pngPath := diskCachePack(t)
	key, ok := pngFileKey(im.Folder, "patch.png")
	if !ok {
		t.Fatal("pngFileKey")
	}
	if _, ok := loadPNGAlpha(im.Folder, "patch.png", key); !ok {
		t.Fatal("loadPNGAlpha")
	}
	disk := alphaDiskPath(key)
	raw, err := safefs.ReadFile(disk)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-8] ^= 0xff // the trailer's CRC
	if err := safefs.WriteFile(disk, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pngPath, make([]byte, info.Size()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(pngPath, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	pngAlphaCache = sync.Map{}
	if _, ok := loadPNGAlpha(im.Folder, "patch.png", key); ok {
		t.Fatal("a mask whose checksum fails was served")
	}
}

// A cache header could claim any size; it is held to the pixel bound the mask was decoded under, so a crafted one
// is refused before its claim is allocated.
func TestAlphaDiskRefusesAnOversizedHeaderBeforeAllocating(t *testing.T) {
	t.Parallel()
	for _, header := range []string{"65536 65536\n", "4294967296 4294967296\n"} {
		refusesAlphaHeader(t, header)
	}
}

func refusesAlphaHeader(t *testing.T, header string) {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(header)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "mask.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	if readAlphaDisk(path) != nil {
		t.Fatalf("header %q read as a mask", header)
	}
	runtime.ReadMemStats(&after)
	if grew := after.TotalAlloc - before.TotalAlloc; grew > 64<<20 {
		t.Fatalf("refusing the header allocated %d MiB", grew>>20)
	}
}

// A mask written by an earlier release still reads, and is written back the same.
func TestAlphaDiskReadsAnEarlierReleasesFile(t *testing.T) {
	t.Parallel()
	old, err := safefs.ReadFile("testdata/mask-gzip-19x5.gz")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "old.gz")
	if err := safefs.WriteFile(path, old, 0o600); err != nil {
		t.Fatal(err)
	}
	got := readAlphaDisk(path)
	if got == nil || got.Width != 19 || got.Height != 5 {
		t.Fatalf("earlier release's mask = %v", got)
	}
	if !bytes.Equal(maskBytes(t, got), old) {
		t.Fatal("mask no longer writes the bytes it was read from")
	}
}

func maskBytes(t *testing.T, m *tilemask.Mask) []byte {
	t.Helper()
	var buf bytes.Buffer
	if _, err := m.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
