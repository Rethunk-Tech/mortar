package problems

import (
	"cmp"
	"context"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// maxPath is Windows' MAX_PATH. It counts the terminating NUL, so 259 characters is the longest full path a program
// opens unless it passes \\?\ paths or its manifest declares longPathAware; LongPathsEnabled in the registry lifts the
// limit only for the latter, and games seldom declare it. Mortar's own copies are not limited, so such a file is laid
// out without an error and only the game fails to open it.
const maxPath = 260

// checkLongPaths is set on Windows, where the game reads the paths. Wine opens a path past MAX_PATH (the BepInEx
// matrix's edge.longpath launch), and Linux and macOS allow far longer ones.
var checkLongPaths = runtime.GOOS == "windows"

// LongPath is an entry with files whose full path, where the game reads it, is at least MAX_PATH characters long, so
// the game may not open them. Placed is set when that path is in a folder of the game (a folder loader's mods), which
// only a shorter game folder fixes; otherwise the file is read in the profile, inside Mortar's data folder. Longest is
// the longest of them and Length its length as Windows counts it.
type LongPath struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Files   int    `json:"files"`
	Longest string `json:"longest"`
	Length  int    `json:"length"`
	Placed  bool   `json:"placed"`
}

// readPath is a file the game reads for a launch of the profile, with the entry that holds it. A file no entry holds
// has no key and names the top folder it is in instead.
type readPath struct {
	key, folder, path string
	placed            bool
}

// pathLength is the length of p in UTF-16 code units, which is how MAX_PATH is counted, for a path of at least maxPath
// bytes; a shorter one is under the limit and gives its byte length.
func pathLength(p string) int {
	// UTF-8 never takes fewer bytes than UTF-16 takes units, so fewer bytes than the limit is under it.
	if len(p) < maxPath {
		return len(p)
	}
	n := 0
	for _, r := range p {
		n += utf16.RuneLen(r)
	}
	return n
}

// longPathRows is one row per entry, or per folder no entry holds, with a path past the limit, in the order their files
// first come up.
func longPathRows(mods []framework.Mod, files []readPath) []LongPath {
	var out []LongPath
	for _, f := range files {
		n := pathLength(f.path)
		if n < maxPath {
			continue
		}
		i := slices.IndexFunc(out, func(x LongPath) bool { return x.Key == f.key && (f.key != "" || x.Name == f.folder) })
		if i < 0 {
			name := cmp.Or(f.key, f.folder)
			if m := slices.IndexFunc(mods, func(x framework.Mod) bool { return x.Key == f.key }); m >= 0 && mods[m].Name != "" {
				name = mods[m].Name
			}
			out = append(out, LongPath{Key: f.key, Name: name})
			i = len(out) - 1
		}
		row := &out[i]
		row.Files++
		row.Placed = row.Placed || f.placed
		if n > row.Length {
			row.Longest, row.Length = f.path, n
		}
	}
	return out
}

// placedPaths turns the files the profile's packages lay out, by slash path below dir with their entry's key, into the
// paths the game reads them at: a file below the loader's placed folder at the same path below root, every other file
// where it is. An empty root places nothing.
func placedPaths(dir string, owners map[string]string, placedFolder, root string) []readPath {
	out := make([]readPath, 0, len(owners))
	for _, rel := range slices.Sorted(maps.Keys(owners)) {
		f := readPath{key: owners[rel], path: filepath.Join(dir, filepath.FromSlash(rel))}
		if below, ok := underFolder(rel, placedFolder); ok && root != "" {
			f.path, f.placed = filepath.Join(root, filepath.FromSlash(below)), true
		}
		out = append(out, f)
	}
	return out
}

// underFolder is rel below folder, both slash paths; every path is below the empty folder.
func underFolder(rel, folder string) (string, bool) {
	if folder == "" {
		return rel, true
	}
	return strings.CutPrefix(rel, folder+"/")
}

// longModsFolderPaths lists the files of a redirected profile's mods folder, which the game reads in place, whose path
// is past the limit, each held by the mod whose folder holds it, else by no mod and named by its top folder. Only those
// are matched to a mod, since an 800-mod profile holds tens of thousands of files.
func longModsFolderPaths(modsDir string, mods []framework.Mod) []readPath {
	folders := map[string]string{}
	for _, m := range mods {
		if rel, err := filepath.Rel(modsDir, m.Folder); err == nil && rel != "." && filepath.IsLocal(rel) {
			folders[filepath.ToSlash(rel)] = m.Key
		}
	}
	// A folder that cannot be read is skipped: its files are no worse off than before the check. So is one whose name
	// starts with a dot, which is how a switched-off mod is kept and which the loader does not read.
	top, err := os.ReadDir(modsDir)
	if err != nil {
		return nil
	}
	// Each top folder is walked on its own goroutine: on Windows a walk mostly waits on each folder's listing, and an
	// 800-mod folder one after another took about a second.
	found := make([][]readPath, len(top))
	sem := make(chan struct{}, max(1, runtime.GOMAXPROCS(0)))
	var wg sync.WaitGroup
	for i, e := range top {
		if e.IsDir() && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			found[i] = longPathsBelow(modsDir, e.Name(), folders)
		})
	}
	wg.Wait()
	return slices.Concat(found...)
}

// longPathsBelow is longModsFolderPaths for top, one entry of the mods folder.
func longPathsBelow(modsDir, top string, folders map[string]string) []readPath {
	var out []readPath
	_ = fs.WalkDir(os.DirFS(modsDir), top, func(rel string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}
		if p := filepath.Join(modsDir, filepath.FromSlash(rel)); err == nil && d.Type().IsRegular() && pathLength(p) >= maxPath {
			f := readPath{key: ownerKey(rel, folders), path: p}
			if f.key == "" {
				f.folder = top
			}
			out = append(out, f)
		}
		return nil
	})
	return out
}

// ownerKey is the key of the mod whose folder, in folders by slash path below the mods folder, is nearest above rel, a
// file's slash path there; empty when no mod's folder holds it.
func ownerKey(rel string, folders map[string]string) string {
	for dir := path.Dir(rel); dir != "."; dir = path.Dir(dir) {
		if key, ok := folders[dir]; ok {
			return key
		}
	}
	return ""
}

// longPaths is the profile's entries with a file the game would read at a path past MAX_PATH. A game whose profile
// holds its packages (deploy "profile") is checked against the files they lay out, the list a launch syncs, so before
// the first launch too; a redirected one against its mods folder.
func (s *Service) longPaths(gameID, id string, mods []framework.Mod) []LongPath {
	if !checkLongPaths || s.profiles == nil {
		return nil
	}
	if info, ok := components.Game(gameID); !ok || info.Deploy != components.DeployProfile {
		modsDir, err := s.profiles.ModsDir(gameID, id)
		if err != nil {
			return nil
		}
		return longPathRows(mods, longModsFolderPaths(modsDir, mods))
	}
	dir, err := s.profiles.ProfileDir(gameID, id)
	if err != nil {
		return nil
	}
	owners, err := s.profiles.PackageFileOwners(gameID, id)
	if err != nil {
		return nil
	}
	folder, root := "", ""
	if l, ok := game.LoaderOf(gameID, s.profiles.LoaderID(gameID, id)); ok && s.settings != nil {
		if p, ok := l.(loader.PlacesFolder); ok {
			if f, role, ok := p.PlacedFolder(gameID); ok {
				if r, err := game.PathFor(s.home, s.settings.Get(), gameID, "", role); err == nil {
					folder, root = strings.TrimPrefix(path.Clean("/"+f), "/"), r
				}
			}
		}
	}
	return longPathRows(mods, placedPaths(dir, owners, folder, root))
}

// ChangedEvent is emitted with a Changed when rows a ProblemsNoWait answer went without are in.
const ChangedEvent = "problems:changed"

// Changed names the profile whose problems changed.
type Changed struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
}

// longPathWalk is the long-path walk of one profile for one fingerprint of its mods; rows is set when done closes.
type longPathWalk struct {
	fingerprint string
	done        chan struct{}
	rows        []LongPath
	// walked is set with rows, before then runs and done closes.
	walked bool
	// then is what runs once the rows are in and before done closes, such as the health point of the check.
	then []func()
	// was is what the walk before found, which an answer that does not wait holds until this walk ends, so the rows of
	// a profile change once instead of leaving and coming back.
	was []LongPath
	// owed is set when an answer held was, so the end of the walk is announced.
	owed bool
}

// longPathWalkOf is the profile's walk for fingerprint fp, running walk on a goroutine of its own unless the walk for
// fp has already started. A build that checks nothing has its empty rows at once.
func (s *Service) longPathWalkOf(gameID, id, fp string, walk func() []LongPath) *longPathWalk {
	key := gameID + "/" + id
	s.mu.Lock()
	defer s.mu.Unlock()
	last := s.long[key]
	if last != nil && last.fingerprint == fp {
		return last
	}
	w := &longPathWalk{fingerprint: fp, done: make(chan struct{})}
	if last != nil {
		w.was = last.found()
	}
	s.long[key] = w
	if !checkLongPaths {
		w.walked = true
		close(w.done)
		return w
	}
	go func() {
		rows := walk()
		s.mu.Lock()
		w.rows, w.walked = rows, true
		then := w.then
		s.mu.Unlock()
		for _, f := range then {
			f()
		}
		s.mu.Lock()
		close(w.done)
		owed := w.owed
		s.mu.Unlock()
		if owed && s.Emit != nil {
			s.Emit(ChangedEvent, Changed{Game: gameID, Profile: id})
		}
	}()
	return w
}

// afterLongPaths runs f once w has its rows, at once when it has them already.
func (s *Service) afterLongPaths(w *longPathWalk, f func()) {
	s.mu.Lock()
	if !w.walked {
		w.then = append(w.then, f)
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	f()
}

// found is the walk's rows once it has ended, else the rows of the walk before. The caller holds Service.mu.
func (w *longPathWalk) found() []LongPath {
	select {
	case <-w.done:
		return w.rows
	default:
		return w.was
	}
}

// withLongPaths gives r the walk's rows, waiting for them when wait is set. An answer that does not wait, or whose ctx
// ends first, gets the rows the walk before found, and the end of this one is announced.
func (s *Service) withLongPaths(ctx context.Context, w *longPathWalk, wait bool, r Result) Result {
	if wait {
		select {
		case <-w.done:
		case <-ctx.Done():
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r.LongPaths = w.found()
	select {
	case <-w.done:
	default:
		w.owed = true
	}
	return r
}
