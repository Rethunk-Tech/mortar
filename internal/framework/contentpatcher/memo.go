package contentpatcher

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

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
	Files          []packFileStamp           `json:"files"`
	AssetConflicts []framework.AssetConflict `json:"assetConflicts"`
	Settings       []framework.SettingHint   `json:"settings"`
	Cleanup        []framework.Cleanup       `json:"cleanup"`
	Redundant      []framework.Redundant     `json:"redundant"`
}

func analysisMemoDir() (string, error) {
	base, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cache", "problems-content-patcher"), nil
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

// analysisKey covers what a check reads besides files: the build, the mods and the scan depth.
func analysisKey(in framework.Input) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%d|%t|%#v", buildStamp(), contentPackParserVersion, SkipImageOverlap, in)
	return hex.EncodeToString(h.Sum(nil))
}

func lookupAnalysis(key string) (framework.Findings, bool) {
	dir, err := analysisMemoDir()
	if err != nil {
		return framework.Findings{}, false
	}
	path := filepath.Join(dir, key+".json.gz")
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return framework.Findings{}, false
	}
	payload, ok := unzipPackCache(raw)
	if !ok {
		return framework.Findings{}, false
	}
	var memo analysisMemo
	if json.Unmarshal(payload, &memo) != nil || !stampsValid(memo.Files) {
		return framework.Findings{}, false
	}
	now := time.Now()
	_ = os.Chtimes(path, now, now)
	return framework.Findings{AssetConflicts: memo.AssetConflicts, Settings: memo.Settings, Cleanup: memo.Cleanup, Redundant: memo.Redundant}, true
}

// stampsValid stats in chunks on every core: a large profile's packs stamp tens of thousands of files.
func stampsValid(files []packFileStamp) bool {
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

func stampOf(abs string) (packFileStamp, bool) {
	info, err := os.Stat(abs)
	switch {
	case err == nil:
		return packFileStamp{Path: abs, Size: info.Size(), ModTime: info.ModTime().UnixNano()}, true
	case errors.Is(err, fs.ErrNotExist):
		return packFileStamp{Path: abs, Size: absentSize}, true
	}
	return packFileStamp{}, false
}

func storeAnalysis(key string, files []packFileStamp, f framework.Findings) {
	cutoff := time.Now().Add(-racyStampWindow).UnixNano()
	if slices.ContainsFunc(files, func(s packFileStamp) bool { return s.ModTime > cutoff }) {
		return
	}
	dir, err := analysisMemoDir()
	if err != nil {
		return
	}
	raw, err := json.Marshal(analysisMemo{
		Files: files, AssetConflicts: f.AssetConflicts, Settings: f.Settings, Cleanup: f.Cleanup, Redundant: f.Redundant,
	})
	if err != nil {
		return
	}
	packed, err := zipPackCache(raw)
	if err != nil || os.MkdirAll(dir, 0o700) != nil {
		return
	}
	if datadir.WriteFile(filepath.Join(dir, key+".json.gz"), packed, 0o600) != nil {
		return
	}
	pruneAnalysisMemos(dir)
}

func pruneAnalysisMemos(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type memoFile struct {
		name string
		at   int64
	}
	var memos []memoFile
	for _, e := range entries {
		if info, err := e.Info(); err == nil && !e.IsDir() {
			memos = append(memos, memoFile{e.Name(), info.ModTime().UnixNano()})
		}
	}
	slices.SortFunc(memos, func(a, b memoFile) int { return cmp.Compare(b.at, a.at) })
	for _, m := range memos[min(len(memos), analysisMemoKeep):] {
		_ = os.Remove(filepath.Join(dir, m.name))
	}
}

// passReads collects the files a check reads outside the packs' own stamps (load files compared byte for byte, mod
// folders and the maps in them). Checks of two profiles at once share it, which only adds stamps to each. log keeps
// every read in order, repeats included, so a part of a check can take the reads it made (see markReads).
var passReads struct {
	sync.Mutex
	active     int
	files      map[string]packFileStamp
	log        []packFileStamp
	missed     int
	incomplete bool
}

// recordReads starts collecting; the func it returns ends it and gives the stamps, or false when a file changed while
// it was read.
func recordReads() func() ([]packFileStamp, bool) {
	passReads.Lock()
	if passReads.active == 0 {
		passReads.files, passReads.log, passReads.missed, passReads.incomplete = map[string]packFileStamp{}, nil, 0, false
	}
	passReads.active++
	passReads.Unlock()
	return func() ([]packFileStamp, bool) {
		passReads.Lock()
		defer passReads.Unlock()
		passReads.active--
		if passReads.active == 0 {
			passReads.log = nil
		}
		files := make([]packFileStamp, 0, len(passReads.files))
		for _, s := range passReads.files {
			files = append(files, s)
		}
		return files, !passReads.incomplete
	}
}

// noteRead stamps abs before it is read; info nil means it was not there.
func noteRead(abs string, info fs.FileInfo) {
	stamp := packFileStamp{Path: abs, Size: absentSize}
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
func readsSince(m readMark) ([]packFileStamp, bool) {
	passReads.Lock()
	defer passReads.Unlock()
	if passReads.active == 0 || passReads.missed != m.missed {
		return nil, false
	}
	return slices.Clone(passReads.log[m.log:]), true
}

// packStamps adds the stamps of every pack the check read, and of each pack that has no content.json to read.
func packStamps(mods []framework.Mod, files []packFileStamp) ([]packFileStamp, bool) {
	for _, im := range mods {
		if im.Folder == "" || !isContentPatcherPack(im) {
			continue
		}
		root := filepath.Clean(im.Folder)
		if _, read := packValidated.Load(root); read {
			if c, ok := packCache.Load(root); ok {
				if pack, ok := c.(cachedPack); ok {
					for _, f := range pack.files {
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
