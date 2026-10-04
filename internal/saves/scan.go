// Package saves reads Stardew Valley saves for the mods they have used. SMAPI writes no mod list into a save, but
// mods leave keys prefixed with their UniqueID, which are matched against the mod dataset's index.
package saves

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/meta"
)

const (
	cacheFile = "saves-scan.json"
	infoFile  = "SaveGameInfo"
)

// Info is what a scan learned about one save. Season is 0 (spring) to 3 (winter); Day is 0 when SaveGameInfo
// did not say. Played is when the save was last written, in Unix milliseconds. WhichFarm is Game1.whichFarm
// (−1 when the tag is missing). MillisecondsPlayed and Money come from SaveGameInfo. Used holds lowercased UniqueIDs.
type Info struct {
	Folder             string   `json:"folder"`
	Farm               string   `json:"farm"`
	Farmer             string   `json:"farmer"`
	Season             int      `json:"season"`
	Day                int      `json:"day"`
	Year               int      `json:"year"`
	Played             int64    `json:"played"`
	WhichFarm          int      `json:"whichFarm"`
	MillisecondsPlayed int64    `json:"millisecondsPlayed"`
	Money              int      `json:"money"`
	Used               []string `json:"used"`
}

const scanRev = 1

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

// Scanner scans the saves in Dir and caches results in CacheDir.
type Scanner struct {
	Dir      string
	CacheDir string
	mu       sync.Mutex
}

// Scan returns every save in Dir, newest first. A save that cannot be read is left out and reported in the
// error, which accompanies the readable ones.
func (s *Scanner) Scan(index map[string][]meta.Ref) ([]Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
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
		if !e.IsDir() {
			continue
		}
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			st, ok, err := s.stampOf(e.Name(), len(index))
			if err != nil || !ok {
				results[i].err = err
				return
			}
			if c, hit := old[e.Name()]; hit && c.Stamp == st {
				results[i].c, results[i].ok = c, true
				return
			}
			info, err := s.read(e.Name(), index)
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
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return Info{}, nil
	}
	if err != nil {
		return Info{}, err
	}
	var newest string
	var played int64
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		st, ok, err := s.stampOf(entry.Name(), len(index))
		if err != nil {
			return Info{}, err
		}
		if !ok {
			continue
		}
		at := st.Info
		if at == 0 {
			at = st.Main
		}
		if newest == "" || at > played || (at == played && entry.Name() > newest) {
			newest, played = entry.Name(), at
		}
	}
	if newest == "" {
		return Info{}, nil
	}
	return s.read(newest, index)
}

// stampOf reports ok=false for a folder that is not a save: Stardew's main file is named like its folder.
func (s *Scanner) stampOf(folder string, index int) (st stamp, ok bool, err error) {
	main, err := os.Stat(filepath.Join(s.Dir, folder, folder))
	if errors.Is(err, fs.ErrNotExist) {
		return st, false, nil
	}
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
	info := Info{Folder: folder, WhichFarm: -1, Used: []string{}}
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
	keys, which, hasFarm, err := distinctKeys(f)
	if err != nil {
		return info, err
	}
	if hasFarm {
		info.WhichFarm = which
	}
	seen := map[string]struct{}{}
	for k := range keys {
		if id, ok := uniqueID(k, index); ok {
			seen[id] = struct{}{}
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
	UniqueID string
	Disabled bool
}

// Lacking returns the used UniqueIDs that are not enabled in the profile, minus the dismissed ones. have maps a
// lowercased UniqueID in the profile to whether it is enabled.
func Lacking(used []string, have map[string]bool, dismissed []string) []Lack {
	out := []Lack{}
	for _, id := range used {
		enabled, present := have[id]
		if enabled || slices.ContainsFunc(dismissed, func(d string) bool { return strings.EqualFold(d, id) }) {
			continue
		}
		out = append(out, Lack{UniqueID: id, Disabled: present})
	}
	return out
}
