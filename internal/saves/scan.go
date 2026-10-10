// Package saves finds a game's saves and reads Stardew Valley's for the mods they have used. SMAPI writes no mod list
// into a save, but mods leave keys prefixed with their mod id, which are matched against the mod dataset's index.
package saves

import (
	"bytes"
	"cmp"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

const (
	cacheFile = "saves-scan.json"
	infoFile  = "SaveGameInfo"
)

// Info is what a scan learned about one save. Season is 0 (spring) to 3 (winter); Day is 0 when SaveGameInfo
// did not say. Played is when the save was last written, in Unix milliseconds. WhichFarm is Game1.whichFarm
// (−1 when the tag is missing; 7 for a custom farm, whose Data/AdditionalFarms id is WhichModFarm).
// MillisecondsPlayed and Money come from SaveGameInfo. Used holds mod ids with lowercased locals.
type Info struct {
	Folder             string   `json:"folder"`
	Farm               string   `json:"farm"`
	Farmer             string   `json:"farmer"`
	Season             int      `json:"season"`
	Day                int      `json:"day"`
	Year               int      `json:"year"`
	Played             int64    `json:"played"`
	WhichFarm          int      `json:"whichFarm"`
	WhichModFarm       string   `json:"whichModFarm,omitempty"`
	MillisecondsPlayed int64    `json:"millisecondsPlayed"`
	Money              int      `json:"money"`
	Used               []mod.ID `json:"used"`
	// Unrecorded means the save's format names no mods, so an empty Used says nothing about what it needs.
	Unrecorded bool `json:"unrecorded"`
}

const scanRev = 4

// stamp is what a cached result was computed from; any change recomputes it. Index is the dataset index's size,
// because a newer index can recognise IDs an older one missed. Rev is this parser's shape, so a new field
// is not served from an older cache.
type stamp struct {
	Main  int64 `json:"main"`
	Size  int64 `json:"size"`
	Info  int64 `json:"info"`
	Index int   `json:"index"`
	Rev   int   `json:"rev"`
}

type cached struct {
	Stamp stamp `json:"stamp"`
	Info  Info  `json:"info"`
}

// Scanner scans the saves in Dir, laid out as Files and Companions say (see Layout), and caches results in CacheDir.
type Scanner struct {
	Dir        string
	Files      []string
	Companions []string
	CacheDir   string
	mu         sync.Mutex
}

// Layout is the scanned folder and its save shape.
func (s *Scanner) Layout() Layout {
	return Layout{Dir: s.Dir, Files: s.Files, Companions: s.Companions}
}

// Scan returns every save in Dir, newest first. A save that cannot be read is left out and reported in the
// error, which accompanies the readable ones.
func (s *Scanner) Scan(index map[string][]meta.Ref) ([]Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.Layout().Names()
	if err != nil {
		return nil, err
	}
	old := s.load()
	type result struct {
		c   cached
		err error
		ok  bool
	}
	results := make([]result, len(entries))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for i, e := range entries {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			st, ok, err := s.stampOf(e, len(index))
			if err != nil || !ok {
				results[i].err = err
				return
			}
			if c, hit := old[e]; hit && c.Stamp == st {
				results[i].c, results[i].ok = c, true
				return
			}
			info, err := s.read(e, index)
			results[i].c, results[i].ok, results[i].err = cached{Stamp: st, Info: info}, err == nil, err
		})
	}
	wg.Wait()
	var errs []error
	fresh := map[string]cached{}
	var out []Info
	for _, r := range results {
		errs = append(errs, r.err)
		if r.ok {
			fresh[r.c.Info.Folder] = r.c
			out = append(out, r.c.Info)
		}
	}
	s.store(fresh)
	slices.SortFunc(out, func(a, b Info) int {
		if a.Played != b.Played {
			return int(b.Played - a.Played)
		}
		return strings.Compare(a.Folder, b.Folder)
	})
	return out, errors.Join(errs...)
}

// Newest reads only the save folder most recently written. It is used by the launch warning, where the other saves
// and their mod names are not needed until the user opens the Saves tab.
func (s *Scanner) Newest(index map[string][]meta.Ref) (Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.Layout().Names()
	if err != nil {
		return Info{}, err
	}
	var newest string
	var played int64
	for _, entry := range entries {
		st, ok, err := s.stampOf(entry, len(index))
		if err != nil {
			return Info{}, err
		}
		if !ok {
			continue
		}
		at := st.Info
		at = cmp.Or(at, st.Main)
		if newest == "" || at > played || (at == played && entry > newest) {
			newest, played = entry, at
		}
	}
	if newest == "" {
		return Info{}, nil
	}
	return s.read(newest, index)
}

// Layout is a saves folder and how a save sits in it: a file matching one of Files, or, without Files, a folder
// holding a file of its own name (Stardew Valley). A pattern may name one folder below Dir ("worlds_local/*.fwl"),
// and a pattern starting with "!" excludes the files it matches. A file save's name is its slash path under Dir.
// Companions are extensions of files beside a file save that share its stem and belong to it (a Valheim world's
// .db beside its .fwl).
type Layout struct {
	Dir        string
	Files      []string
	Companions []string
}

// IsSave reports whether name, a save's path under Dir, is a save.
func (l Layout) IsSave(name string) bool {
	if name == "" || name == "." || strings.Contains(name, "\\") || path.Clean(name) != name || path.IsAbs(name) || strings.HasPrefix(name, "..") {
		return false
	}
	if len(l.Files) == 0 {
		if name != path.Base(name) {
			return false
		}
		return fsx.IsFile(filepath.Join(l.Dir, name, name))
	}
	if !l.Matches(name) {
		return false
	}
	return fsx.IsFile(filepath.Join(l.Dir, filepath.FromSlash(name)))
}

// Matches reports whether name, a slash path under Dir, is named by Files, whether or not it exists.
func (l Layout) Matches(name string) bool {
	in := false
	for _, p := range l.Files {
		if not, ok := strings.CutPrefix(p, "!"); ok {
			if m, _ := path.Match(not, path.Base(name)); m {
				return false
			}
			continue
		}
		if m, _ := path.Match(p, name); m {
			in = true
		}
	}
	return in
}

// Paths are the files and folders, as slash paths under Dir, that make up save name: its folder or file and, for a
// file save, the companions that exist.
func (l Layout) Paths(name string) []string {
	out := []string{name}
	stem := strings.TrimSuffix(name, path.Ext(name))
	for _, ext := range l.Companions {
		if c := stem + ext; c != name {
			if fsx.IsFile(filepath.Join(l.Dir, filepath.FromSlash(c))) {
				out = append(out, c)
			}
		}
	}
	return out
}

// Names lists the saves in Dir by name; a missing Dir holds none.
func (l Layout) Names() ([]string, error) {
	dirs := []string{"."}
	for _, p := range l.Files {
		if d := path.Dir(p); !strings.HasPrefix(p, "!") && !slices.Contains(dirs, d) {
			dirs = append(dirs, d)
		}
	}
	var out []string
	for _, d := range dirs {
		ents, err := os.ReadDir(filepath.Join(l.Dir, filepath.FromSlash(d)))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range ents {
			if name := path.Join(d, e.Name()); l.IsSave(name) {
				out = append(out, name)
			}
		}
	}
	return out, nil
}

// WrittenSince lists the saves with a file written at or after t: the saves a run that started then played.
func (l Layout) WrittenSince(t time.Time) ([]string, error) {
	names, err := l.Names()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, name := range names {
		if mtime, _, err := l.newest(name); err == nil && !mtime.Before(t) {
			out = append(out, name)
		}
	}
	return out, nil
}

// newest is the latest write among save name's files, and their total size.
func (l Layout) newest(name string) (mtime time.Time, size int64, err error) {
	for _, p := range l.Paths(name) {
		fi, err := os.Stat(filepath.Join(l.Dir, filepath.FromSlash(p)))
		if err != nil {
			return mtime, size, err
		}
		if fi.ModTime().After(mtime) {
			mtime = fi.ModTime()
		}
		size += fi.Size()
	}
	return mtime, size, nil
}

// stampOf reports ok=false for an entry that is not a save (see Layout.IsSave).
func (s *Scanner) stampOf(folder string, index int) (st stamp, ok bool, err error) {
	if !s.Layout().IsSave(folder) {
		return st, false, nil
	}
	if len(s.Files) > 0 {
		mtime, size, err := s.Layout().newest(folder)
		if err != nil {
			return st, false, err
		}
		return stamp{Main: mtime.UnixNano(), Size: size, Rev: scanRev}, true, nil
	}
	main, err := os.Stat(filepath.Join(s.Dir, folder, folder))
	if err != nil {
		return st, false, err
	}
	st = stamp{Main: main.ModTime().UnixNano(), Size: main.Size(), Index: index, Rev: scanRev}
	if fi, err := os.Stat(filepath.Join(s.Dir, folder, infoFile)); err == nil {
		st.Info = fi.ModTime().UnixNano()
	}
	return st, true, nil
}

func (s *Scanner) read(folder string, index map[string][]meta.Ref) (Info, error) {
	info := Info{Folder: folder, WhichFarm: -1, Used: []mod.ID{}}
	if len(s.Files) > 0 {
		// A save kept as one file (Lethal Company's encrypted ES3) names no mods, so its fit is unknown.
		info.Unrecorded = true
		mtime, _, err := s.Layout().newest(folder)
		if err != nil {
			return info, err
		}
		info.Played = mtime.UnixMilli()
		return info, nil
	}
	dir := filepath.Join(s.Dir, folder)
	// A missing SaveGameInfo only costs the details it holds.
	if b, err := fsx.ReadFile(filepath.Join(dir, infoFile)); err == nil {
		readInfo(b, &info)
	}
	if fi, err := os.Stat(filepath.Join(dir, infoFile)); err == nil {
		info.Played = fi.ModTime().UnixMilli()
	} else if fi, err := os.Stat(filepath.Join(dir, folder)); err == nil {
		info.Played = fi.ModTime().UnixMilli()
	}
	f, err := fsx.Open(filepath.Join(dir, folder))
	if err != nil {
		return info, err
	}
	defer func() { _ = f.Close() }()
	keys, farm, err := distinctKeys(f)
	if err != nil {
		return info, err
	}
	if farm.has {
		info.WhichFarm = farm.which
	}
	info.WhichModFarm = farm.mod
	seen := map[mod.ID]struct{}{}
	for k := range keys {
		if id, ok := uniqueID(k, index); ok {
			seen[mod.SMAPI(id)] = struct{}{}
		}
	}
	for id := range seen {
		info.Used = append(info.Used, id)
	}
	slices.Sort(info.Used)
	return info, nil
}

// readInfo fills farmer, farm, date, money and playtime from SaveGameInfo (root <Farmer> direct children).
// It stops once those tags have been seen so a large inventory is not fully walked.
func readInfo(b []byte, info *Info) {
	d := xml.NewDecoder(bytes.NewReader(bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})))
	depth := 0
	var field string
	var got uint8
	const all uint8 = 1<<7 - 1
	mark := func(bit uint8) {
		got |= bit
	}
	for got != all {
		tok, err := d.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			field = ""
			if depth == 2 {
				field = t.Name.Local
			}
		case xml.EndElement:
			depth--
			field = ""
		case xml.CharData:
			v := strings.TrimSpace(string(t))
			switch field {
			case "name":
				info.Farmer = v
				mark(1 << 0)
			case "farmName":
				info.Farm = v
				mark(1 << 1)
			case "seasonForSaveGame":
				info.Season, _ = strconv.Atoi(v)
				mark(1 << 2)
			case "dayOfMonthForSaveGame":
				info.Day, _ = strconv.Atoi(v)
				mark(1 << 3)
			case "yearForSaveGame":
				info.Year, _ = strconv.Atoi(v)
				mark(1 << 4)
			case "money":
				info.Money, _ = strconv.Atoi(v)
				mark(1 << 5)
			case "millisecondsPlayed":
				info.MillisecondsPlayed, _ = strconv.ParseInt(v, 10, 64)
				mark(1 << 6)
			}
			field = ""
		}
	}
}

func (s *Scanner) cachePath() string {
	if s.CacheDir == "" {
		return ""
	}
	return filepath.Join(s.CacheDir, cacheFile)
}

// load returns the cached results; a missing or unreadable cache only costs a rescan.
func (s *Scanner) load() map[string]cached {
	p := s.cachePath()
	if p == "" {
		return nil
	}
	b, err := fsx.ReadFile(p)
	if err != nil {
		return nil
	}
	var m map[string]cached
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	return m
}

func (s *Scanner) store(m map[string]cached) {
	if p := s.cachePath(); p != "" && os.MkdirAll(s.CacheDir, 0o700) == nil {
		_ = datadir.WriteJSON(p, m)
	}
}

// Lack is a mod a save has used that the profile does not run. Disabled means the profile has it but switched off.
type Lack struct {
	ID       mod.ID
	Disabled bool
}

// Lacking returns the used ids that are not enabled in the profile, minus the dismissed ones. have maps a folded
// id (mod.ID.Fold) in the profile to whether it is enabled.
func Lacking(used []mod.ID, have map[string]bool, dismissed []mod.ID) []Lack {
	out := []Lack{}
	for _, id := range used {
		enabled, present := have[id.Fold()]
		if enabled || slices.ContainsFunc(dismissed, func(d mod.ID) bool { return mod.Equal(d, id) }) {
			continue
		}
		out = append(out, Lack{ID: id, Disabled: present})
	}
	return out
}
