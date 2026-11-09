package datasvc

import (
	"cmp"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// Progress is a size walk in progress.
type Progress struct {
	Measuring  bool   `json:"measuring"`
	Copying    bool   `json:"copying"`
	Bytes      int64  `json:"bytes"`
	TotalBytes int64  `json:"totalBytes"`
	Files      int    `json:"files"`
	TotalFiles int    `json:"totalFiles"`
	Path       string `json:"path"`
}

// ProfileSize is one profile's mods folder, not the shared store.
type ProfileSize struct {
	Game string `json:"game"`
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// GameUsage is one game's profiles, store items, save backups, and cache entries.
type GameUsage struct {
	Game string `json:"game"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// Usage is disk use of Mortar's data folder.
type Usage struct {
	Path             string        `json:"path"`
	Profiles         []ProfileSize `json:"profiles"`
	Games            []GameUsage   `json:"games"`
	Store            int64         `json:"store"`
	Cache            int64         `json:"cache"`
	Backups          int64         `json:"backups"`
	Trash            int64         `json:"trash"`
	Total            int64         `json:"total"`
	SharedSaved      int64         `json:"sharedSaved"`
	SharedSavedKnown bool          `json:"sharedSavedKnown"`
}

// CacheInfo is Mortar's cache folder.
type CacheInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// EntrySize is one store key's size on disk.
type EntrySize struct {
	Game string `json:"game"`
	Key  string `json:"key"`
	Size int64  `json:"size"`
}

type profileMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

const (
	probeBatch   = 1024
	probeWorkers = 16
	// jobsBuffered is how many batches a walk may run ahead of the accounting, which takes the folders in order.
	jobsBuffered = 1024
)

// batchFile is a regular file to size, or (followed set) a symlink target already stat-ed that only marks its inode
// as seen, kept in the batch so it lands in walk order among the files.
type batchFile struct {
	path     string
	d        fs.DirEntry
	followed os.FileInfo
}

// probeJob is one batch of files whose extent maps are being read; wg is done when probes is filled.
type probeJob struct {
	files  []batchFile
	probes []batchResult
	wg     sync.WaitGroup
}

// startProbes reads files' extent maps in the background, at most cap(slots) reads at a time across all jobs.
func startProbes(files []batchFile, slots chan struct{}) *probeJob {
	j := &probeJob{files: files, probes: make([]batchResult, len(files))}
	next := atomic.Int64{}
	for range min(probeWorkers, len(files)) {
		j.wg.Go(func() {
			for {
				i := int(next.Add(1)) - 1
				if i >= len(files) {
					return
				}
				if files[i].followed != nil {
					continue
				}
				slots <- struct{}{}
				info, err := files[i].d.Info()
				if err == nil {
					j.probes[i] = batchResult{info, probeFile(files[i].path), true}
				}
				<-slots
			}
		})
	}
	return j
}

type batchResult struct {
	info  os.FileInfo
	probe fileProbe
	read  bool
}

// usageAcc accumulates the buckets of one Measure walk.
type usageAcc struct {
	root       string
	u          *Usage
	mods       map[string]int64
	direct     map[string]int64
	backupGame map[string]int64
	cacheSeg   map[[2]string]int64
	share      *shareAcc
	report     func(Progress)
}

func (a *usageAcc) account(path string, info os.FileInfo, probe fileProbe) {
	a.share.addFollowed(path, info)
	n, known := fileExclusive(path, info, probe, a.share)
	if known {
		a.share.allocated += n
	}
	a.u.Total += n
	rel := path
	if r, relErr := filepath.Rel(a.root, path); relErr == nil {
		rel = r
	}
	slash := filepath.ToSlash(rel)
	a.bucket(slash, n)
	if a.report != nil {
		a.report(Progress{Measuring: true, Bytes: a.u.Total, Path: slash})
	}
}

// bucket charges n bytes of the file at slash (relative to the data root) to its top-level folder and game.
func (a *usageAcc) bucket(slash string, n int64) {
	switch {
	case hasPrefix(slash, "store/"):
		a.u.Store += n
		if game := nestedGame(slash, "store/"); game != "" {
			a.direct[game] += n
		}
	case hasPrefix(slash, "cache/"):
		a.u.Cache += n
		a.cacheSeg[cacheSegs(slash)] += n
	case hasPrefix(slash, "backups/"):
		a.u.Backups += n
		if game := nestedGame(slash, "backups/"); game != "" {
			a.backupGame[game] += n
		}
	case hasPrefix(slash, "trash/"):
		a.u.Trash += n
	default:
		if game := nestedGame(slash, "profiles/"); game != "" {
			a.direct[game] += n
		}
		if game, id, ok := profileMods(slash); ok {
			a.mods[game+"/"+id] += n
		}
	}
}

// Measure walks root without following symlinks and reports each bucket's exclusive size.
func Measure(root string, report func(Progress)) (Usage, error) {
	u := Usage{Path: root, Profiles: []ProfileSize{}, Games: []GameUsage{}}
	root = filepath.Clean(root)
	acc := &usageAcc{
		root: root, u: &u, mods: map[string]int64{}, direct: map[string]int64{}, backupGame: map[string]int64{},
		cacheSeg: map[[2]string]int64{}, share: newShareAcc(), report: report,
	}
	if err := acc.walkAll(); err != nil {
		return Usage{}, err
	}
	u.SharedSaved, u.SharedSavedKnown = acc.share.saved()
	dirs, err := datadir.ProfileDirs(filepath.Join(root, "profiles"))
	if err != nil {
		return Usage{}, err
	}
	for _, d := range dirs {
		name := d.ID
		if meta := readProfile(filepath.Join(d.Dir, "profile.json")); meta.Name != "" {
			name = meta.Name
		}
		u.Profiles = append(u.Profiles, ProfileSize{Game: d.Game, ID: d.ID, Name: name, Size: acc.mods[d.Game+"/"+d.ID]})
	}
	u.Games = acc.gameRows(dirs)
	return u, nil
}

// walkAll walks every top-level folder on its own goroutine and accounts the results. The files' extent maps are read
// by probeWorkers readers shared by all the walks. Results are accounted in folder order and then walk order, since
// which file a shared block is charged to depends on the order.
func (a *usageAcc) walkAll() error {
	slots := make(chan struct{}, probeWorkers)
	type folder struct {
		path      string
		skipNamed bool
	}
	var folders []folder
	for _, name := range []string{"store", "profiles", "backups", "cache", "trash"} {
		folders = append(folders, folder{filepath.Join(a.root, name), false})
	}
	folders = append(folders, folder{a.root, true})
	jobs := make([]chan *probeJob, len(folders))
	errs := make([]error, len(folders))
	for i, f := range folders {
		jobs[i] = make(chan *probeJob, jobsBuffered)
		go func() {
			defer close(jobs[i])
			errs[i] = walkFolder(a.root, f.path, f.skipNamed, jobs[i], slots)
		}()
	}
	for _, queue := range jobs {
		for job := range queue {
			job.wg.Wait()
			for i, f := range job.files {
				if f.followed != nil {
					a.share.addFollowed(f.path, f.followed)
				} else if r := job.probes[i]; r.read {
					a.account(f.path, r.info, r.probe)
				}
			}
		}
	}
	return firstErr(errs)
}

func firstErr(errs []error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// walkFolder walks path and sends its regular files, in batches, to out with their extent maps being read.
func walkFolder(root, path string, skipNamed bool, out chan<- *probeJob, slots chan struct{}) error {
	var batch []batchFile
	flush := func() {
		if len(batch) > 0 {
			out <- startProbes(batch, slots)
			batch = nil
		}
	}
	st, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if st.Mode()&os.ModeSymlink != 0 || !st.IsDir() {
		return nil
	}
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == path {
				return err
			}
			return fs.SkipDir
		}
		if skipNamed && isTopLevelBucket(root, p, d) {
			return fs.SkipDir
		}
		if d.Type()&fs.ModeSymlink != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			if info, ok := followedFile(root, p); ok {
				batch = append(batch, batchFile{followed: info})
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		batch = append(batch, batchFile{path: p, d: d})
		if len(batch) == probeBatch {
			flush()
		}
		return nil
	})
	flush()
	return err
}

// isTopLevelBucket is whether p is one of the folders directly under root that has its own walk.
func isTopLevelBucket(root, p string, d fs.DirEntry) bool {
	if p == root {
		return false
	}
	rel, err := filepath.Rel(root, p)
	if err != nil || filepath.Dir(rel) != "." {
		return false
	}
	switch d.Name() {
	case "store", "profiles", "backups", "cache", "trash":
		return true
	}
	return false
}

// followedFile is the target of the symlink p when it is a regular file under root.
func followedFile(root, p string) (os.FileInfo, bool) {
	info, err := os.Stat(filepath.Clean(p))
	if err != nil || !info.Mode().IsRegular() {
		return nil, false
	}
	resolved, err := safefs.EvalSymlinks(p)
	if err != nil || !safefs.UnderRoot(root, resolved) {
		return nil, false
	}
	return info, true
}

// gameRows turns the per-game sums into rows. Store folders that are not a game (the shared blob store, loader
// installers) are one row with an empty Game, so they are counted without being listed as games.
func (a *usageAcc) gameRows(dirs []datadir.ProfileDir) []GameUsage {
	known := map[string]struct{}{}
	sizes := map[string]int64{}
	for id, n := range a.direct {
		known[id] = struct{}{}
		sizes[id] = n
	}
	for id, n := range a.backupGame {
		known[id] = struct{}{}
		sizes[id] += n
	}
	for segs, n := range a.cacheSeg {
		switch {
		case segs[0] != "":
			if _, ok := known[segs[0]]; ok {
				sizes[segs[0]] += n
				continue
			}
			fallthrough
		default:
			if _, ok := known[segs[1]]; ok {
				sizes[segs[1]] += n
			}
		}
	}
	hasProfiles := map[string]bool{}
	for _, d := range dirs {
		hasProfiles[d.Game] = true
	}
	rows := []GameUsage{}
	var shared int64
	for id, n := range sizes {
		name := id
		if g := game.Find(id); g != nil {
			name = g.Name()
		} else if !hasProfiles[id] {
			shared += n
			continue
		}
		rows = append(rows, GameUsage{Game: id, Name: name, Size: n})
	}
	if shared > 0 {
		rows = append(rows, GameUsage{Size: shared})
	}
	return rows
}

func nestedGame(slash, prefix string) string {
	rest, ok := strings.CutPrefix(slash, prefix)
	if !ok {
		return ""
	}
	game, more, found := strings.Cut(rest, "/")
	// Dot-prefixed folders are Mortar's own indexes (store/.manifests), not games.
	if !found || game == "" || more == "" || strings.HasPrefix(game, ".") {
		return ""
	}
	return game
}

func cacheSegs(slash string) [2]string {
	rest, ok := strings.CutPrefix(slash, "cache/")
	if !ok {
		return [2]string{}
	}
	first, rest, found := strings.Cut(rest, "/")
	if !found {
		return [2]string{first, ""}
	}
	second, _, _ := strings.Cut(rest, "/")
	return [2]string{first, second}
}

func hasPrefix(slash, prefix string) bool {
	return slash == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(slash, prefix)
}

func profileMods(slash string) (game, id string, ok bool) {
	rest, found := strings.CutPrefix(slash, "profiles/")
	if !found {
		return "", "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) < 4 || parts[2] != "mods" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func readProfile(path string) profileMeta {
	b, err := safefs.ReadFile(path)
	if err != nil {
		return profileMeta{}
	}
	var m profileMeta
	_ = json.Unmarshal(b, &m)
	return m
}

// ModUse is one store item's disk use.
type ModUse struct {
	Game        string `json:"game"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	Profiles    int    `json:"profiles"`
	ProfileSize int64  `json:"profileSize"`
	LastUsed    string `json:"lastUsed"`
}

// ModUsage is store items with sizes, who uses them, and a store-size total.
type ModUsage struct {
	Total int64    `json:"total"`
	Items []ModUse `json:"items"`
}

type profileEntries struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Entries []struct {
		Key            string   `json:"key"`
		Item           string   `json:"item"`
		ExtraStoreKeys []string `json:"extraStoreKeys"`
		Source         struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"source"`
	} `json:"entries"`
}

// MeasureMods reports each store folder's size, live-profile use, and copy sizes under profiles/.
func MeasureMods(root string) (ModUsage, error) {
	out := ModUsage{Items: []ModUse{}}
	names, uses, copies := profileUse(root)
	entries, err := store.OpenAt(filepath.Join(root, "store")).Entries()
	if err != nil {
		return ModUsage{}, err
	}
	for _, e := range entries {
		id := e.Game + "/" + e.Key
		n := e.Size
		if n == 0 {
			n = dirSize(e.Dir)
		}
		name := cmp.Or(names[id], e.Key)
		lastUsed := ""
		if !e.LastUsed.IsZero() {
			lastUsed = e.LastUsed.UTC().Format(time.RFC3339)
		}
		out.Items = append(out.Items, ModUse{
			Game: e.Game, Key: e.Key, Name: name, Size: n,
			Profiles: uses[id], ProfileSize: copies[id], LastUsed: lastUsed,
		})
		out.Total += n
	}
	return out, nil
}

// storeSizes is the Size column of MeasureMods alone, from the sizes the store records; an item without one is
// measured and its size recorded. items must be the store the rest of Mortar writes through, whose lock keeps this
// write from undoing another.
func storeSizes(items *store.Store) ([]EntrySize, error) {
	entries, err := items.Entries()
	if err != nil {
		return nil, err
	}
	out := make([]EntrySize, 0, len(entries))
	measured := map[string]map[string]int64{}
	for _, e := range entries {
		n := e.Size
		if n == 0 {
			n = dirSize(e.Dir)
			if measured[e.Game] == nil {
				measured[e.Game] = map[string]int64{}
			}
			measured[e.Game][e.Key] = n
		}
		out = append(out, EntrySize{Game: e.Game, Key: e.Key, Size: n})
	}
	if len(measured) > 0 {
		if err := items.RecordSizes(measured); err != nil {
			slog.Warn("record store sizes", "err", err)
		}
	}
	return out, nil
}

func profileUse(root string) (names map[string]string, uses map[string]int, copies map[string]int64) {
	names = map[string]string{}
	uses = map[string]int{}
	copies = map[string]int64{}
	dirs, _ := datadir.ProfileDirs(filepath.Join(root, "profiles"))
	for _, d := range dirs {
		p := readProfileEntries(filepath.Join(d.Dir, "profile.json"))
		mods := datadir.ModsDirOf(d.Dir)
		seen := map[string]bool{}
		for _, e := range p.Entries {
			keys := append([]string{cmp.Or(e.Item, e.Key)}, e.ExtraStoreKeys...)
			for _, key := range keys {
				if key == "" {
					continue
				}
				id := d.Game + "/" + key
				if e.Source.Name != "" && names[id] == "" {
					names[id] = sourceLabel(e.Source.Name, e.Source.Version)
				}
				if !seen[id] {
					uses[id]++
					seen[id] = true
				}
				copies[id] += copySize(mods, e.Key, e.Item, key)
			}
		}
	}
	return names, uses, copies
}

func sourceLabel(name, version string) string {
	if version == "" {
		return name
	}
	return name + " " + version
}

// copySize is the size of the profile's copy of store key under mods for the entry entryKey.
func copySize(mods, entryKey, item, key string) int64 {
	if key == entryKey && item == "" {
		return dirSize(filepath.Join(mods, key)) + dirSize(filepath.Join(mods, "."+key))
	}
	return dirSize(filepath.Join(mods, entryKey, key))
}

func readProfileEntries(path string) profileEntries {
	b, err := safefs.ReadFile(path)
	if err != nil {
		return profileEntries{}
	}
	var p profileEntries
	_ = json.Unmarshal(b, &p)
	return p
}

func clearCache(dir string) error {
	ents, found, err := safefs.ReadDirIfExists(filepath.Clean(dir))
	if err != nil {
		return err
	}
	if !found {
		return os.MkdirAll(filepath.Clean(dir), 0o700)
	}
	var errs []error
	for _, e := range ents {
		errs = append(errs, safefs.RemoveAll(filepath.Clean(filepath.Join(dir, e.Name()))))
	}
	return errors.Join(errs...)
}

func usageFingerprint(root string) string {
	var n int64
	for _, rel := range []string{filepath.Join("store", "index.json"), "profiles", "store"} {
		info, err := os.Stat(filepath.Clean(filepath.Join(root, rel)))
		if err != nil {
			continue
		}
		n += info.ModTime().UnixNano()
	}
	return strconv.FormatInt(n, 10)
}
