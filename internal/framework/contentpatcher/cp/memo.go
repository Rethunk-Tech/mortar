package cp

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
)

// A check's findings are kept on disk with a stamp of every file the check read. The next check of the same mods,
// typically the first one after a restart, then stats those files and reads one small file instead of loading and
// scanning every pack.

// analysisMemoKeep bounds the memos kept, one per recently checked set of mods.
const analysisMemoKeep = 8

// racyStampWindow is how recent a file's modification time may be before its stamp is not trusted: a second write
// within the filesystem's timestamp granularity, at the same size, would leave the stamp unchanged.
const racyStampWindow = 2 * time.Second

type analysisMemo struct {
	// Files holds absolute paths.
	Files                []FileStamp       `json:"files"`
	Conflicts            []Conflict        `json:"conflicts,omitempty"`
	Hints                []Hint            `json:"hints,omitempty"`
	Shadowed             []Shadowed        `json:"shadowed,omitempty"`
	Bundled              map[string]ModRef `json:"bundled,omitempty"`
	Supporting           []string          `json:"supporting,omitempty"`
	Mentioned            []string          `json:"mentioned,omitempty"`
	Tilesheets           []TilesheetUse    `json:"tilesheets,omitempty"`
	TilesheetsUnreadable bool              `json:"tilesheetsUnreadable,omitempty"`
}

func analysisMemoDir() (string, error) {
	if cacheRoot == "" {
		return "", errNoCacheRoot
	}
	return filepath.Join(cacheRoot, "cache", "problems-content-patcher"), nil
}

// buildStamp ties a memo to the build that wrote it, so a changed analyzer never serves an older one's findings.
var buildStamp = sync.OnceValue(func() string {
	stamp := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		stamp = info.String()
	}
	if exe, err := os.Executable(); err == nil {
		if info, err := os.Stat(exe); err == nil {
			stamp += "|" + strconv.FormatInt(info.Size(), 10) + "|" + strconv.FormatInt(info.ModTime().UnixNano(), 10)
		}
	}
	return stamp
})

// analysisKey covers what a check reads besides files: the build, the mods and the scan depth. The request's own
// callbacks are left out: they are the caller's code, not part of what a check of these mods finds.
func analysisKey(in Request) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%d|%t|%#v|%#v", buildStamp(), contentPackParserVersion, in.SkipImageOverlap, in.Enabled, in.All)
	return hex.EncodeToString(h.Sum(nil))
}

func lookupAnalysis(key string) (Findings, bool) {
	dir, err := analysisMemoDir()
	if err != nil {
		return Findings{}, false
	}
	path := filepath.Join(dir, key+".json.gz")
	raw, err := safefs.ReadFile(path)
	if err != nil {
		return Findings{}, false
	}
	payload, ok := unzipPackCache(raw)
	if !ok {
		return Findings{}, false
	}
	var memo analysisMemo
	if json.Unmarshal(payload, &memo) != nil || !stampsValid(memo.Files) {
		return Findings{}, false
	}
	now := time.Now()
	_ = os.Chtimes(path, now, now)
	return Findings{
		Conflicts: memo.Conflicts, Hints: memo.Hints, Shadowed: memo.Shadowed, Bundled: memo.Bundled,
		Supporting: memo.Supporting, Mentioned: memo.Mentioned,
		Tilesheets: memo.Tilesheets, TilesheetsUnreadable: memo.TilesheetsUnreadable,
	}, true
}

// stampsValid stats in chunks on every core: a large profile's packs stamp tens of thousands of files.
func stampsValid(files []FileStamp) bool {
	var stale atomic.Bool
	var wg sync.WaitGroup
	for chunk := range slices.Chunk(files, 1024) {
		wg.Go(func() {
			for _, want := range chunk {
				if stale.Load() {
					return
				}
				if got, ok := stampOf(want.Path); !ok || got != want {
					stale.Store(true)
					return
				}
			}
		})
	}
	wg.Wait()
	return !stale.Load()
}

func stampOf(abs string) (FileStamp, bool) {
	info, err := os.Stat(abs)
	switch {
	case err == nil:
		return FileStamp{Path: abs, Size: info.Size(), ModTime: info.ModTime().UnixNano()}, true
	case errors.Is(err, fs.ErrNotExist):
		return FileStamp{Path: abs, Size: absentSize}, true
	}
	return FileStamp{}, false
}

func storeAnalysis(key string, files []FileStamp, f Findings) {
	cutoff := time.Now().Add(-racyStampWindow).UnixNano()
	if slices.ContainsFunc(files, func(s FileStamp) bool { return s.ModTime > cutoff }) {
		return
	}
	dir, err := analysisMemoDir()
	if err != nil {
		return
	}
	memo := analysisMemo{
		Files: files, Conflicts: f.Conflicts, Hints: f.Hints, Shadowed: f.Shadowed, Bundled: f.Bundled,
		Supporting: f.Supporting, Mentioned: f.Mentioned,
		Tilesheets: f.Tilesheets, TilesheetsUnreadable: f.TilesheetsUnreadable,
	}
	if writePackedCache(filepath.Join(dir, key+".json.gz"), memo) {
		pruneAnalysisMemos(dir)
	}
}

func pruneAnalysisMemos(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type memoFile struct {
		Name string
		At   int64
	}
	var memos []memoFile
	for _, e := range entries {
		if info, err := e.Info(); err == nil && !e.IsDir() {
			memos = append(memos, memoFile{e.Name(), info.ModTime().UnixNano()})
		}
	}
	slices.SortFunc(memos, func(a, b memoFile) int { return cmp.Compare(b.At, a.At) })
	for _, m := range memos[min(len(memos), analysisMemoKeep):] {
		_ = safefs.Remove(filepath.Join(dir, m.Name))
	}
}

// passReads collects the files a check reads outside the packs' own stamps (load files compared byte for byte, mod
// folders and the maps in them). Checks of two profiles at once share it, which only adds stamps to each. log keeps
// every read in order, repeats included, so a part of a check can take the reads it made (see markReads).
var passReads struct {
	sync.Mutex
	active     int
	files      map[string]FileStamp
	log        []FileStamp
	missed     int
	incomplete bool
}

// recordReads starts collecting; the func it returns ends it and gives the stamps, or false when a file changed while
// it was read.
func recordReads() func() ([]FileStamp, bool) {
	passReads.Lock()
	if passReads.active == 0 {
		passReads.files, passReads.log, passReads.missed, passReads.incomplete = map[string]FileStamp{}, nil, 0, false
	}
	passReads.active++
	passReads.Unlock()
	return func() ([]FileStamp, bool) {
		passReads.Lock()
		defer passReads.Unlock()
		passReads.active--
		if passReads.active == 0 {
			passReads.log = nil
		}
		files := make([]FileStamp, 0, len(passReads.files))
		for _, s := range passReads.files {
			files = append(files, s)
		}
		return files, !passReads.incomplete
	}
}

// noteRead stamps abs before it is read; info nil means it was not there.
func noteRead(abs string, info fs.FileInfo) {
	stamp := FileStamp{Path: abs, Size: absentSize}
	if info != nil {
		stamp.Size, stamp.ModTime = info.Size(), info.ModTime().UnixNano()
	}
	passReads.Lock()
	defer passReads.Unlock()
	if passReads.active == 0 {
		return
	}
	passReads.log = append(passReads.log, stamp)
	if _, seen := passReads.files[abs]; !seen {
		passReads.files[abs] = stamp
	}
}

func noteUnstampable() {
	passReads.Lock()
	passReads.incomplete = true
	passReads.missed++
	passReads.Unlock()
}

type readMark struct{ log, missed int }

// markReads starts a part of a check, such as one conflict target; readsSince then gives the stamps of what the part
// read. A check running beside it can add its own reads, which only makes the part's stamps stricter.
func markReads() readMark {
	passReads.Lock()
	defer passReads.Unlock()
	return readMark{len(passReads.log), passReads.missed}
}

// readsSince is false when no check is recording or a file read since m could not be stamped.
func readsSince(m readMark) ([]FileStamp, bool) {
	passReads.Lock()
	defer passReads.Unlock()
	if passReads.active == 0 || passReads.missed != m.missed {
		return nil, false
	}
	return slices.Clone(passReads.log[m.log:]), true
}

// packStamps adds the stamps of every pack the check read, and of each pack that has no content.json to read.
func packStamps(mods []Mod, files []FileStamp) ([]FileStamp, bool) {
	for _, im := range mods {
		if im.Folder == "" || !IsContentPatcherPack(im) {
			continue
		}
		root := filepath.Clean(im.Folder)
		if _, read := packValidated.Load(root); read {
			if c, ok := packCache.Load(root); ok {
				if pack, ok := c.(Pack); ok {
					for _, f := range pack.Files {
						f.Path = filepath.Join(root, filepath.FromSlash(f.Path))
						files = append(files, f)
					}
					continue
				}
			}
		}
		stamp, ok := stampOf(filepath.Join(root, "content.json"))
		if !ok {
			return nil, false
		}
		if stamp.Size == absentSize {
			files = append(files, stamp)
		} else if im.Enabled {
			// Every enabled pack is read, so one left unvalidated changed while it was read.
			return nil, false
		}
	}
	return files, true
}
