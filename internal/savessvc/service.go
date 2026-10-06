// Package savessvc tells the frontend how well each save fits a profile.
package savessvc

import (
	"cmp"
	"context"
	"errors"
	"log"
	"maps"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/saves"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

const (
	nameTimeout = 15 * time.Second
	nameWorkers = 8
)

// Lack is a mod the save has used that the profile does not run. Name falls back to the mod id. Where is the page
// to get it from, found the way a missing dependency's is, and nil when the mod is unknown or the dataset could not
// be reached; a Nexus page or a GitHub repository in it can be queued into the profile.
type Lack struct {
	ID   mod.ID `json:"id"`
	Name string `json:"name"`
	// Disabled means the profile has the mod switched off; otherwise it is absent.
	Disabled bool          `json:"disabled"`
	Where    *problems.Ref `json:"where"`
}

// Fit is one save and what it has used that the profile lacks. Season is 0 (spring) to 3 (winter); Day is 0
// when unknown. Played is when the save was last written, in Unix milliseconds. WhichFarm is Game1.whichFarm
// (−1 when missing). MillisecondsPlayed and Money come from SaveGameInfo.
type Fit struct {
	Folder             string      `json:"folder"`
	Farm               string      `json:"farm"`
	Farmer             string      `json:"farmer"`
	Season             int         `json:"season"`
	Day                int         `json:"day"`
	Year               int         `json:"year"`
	Played             int64       `json:"played"`
	WhichFarm          int         `json:"whichFarm"`
	MillisecondsPlayed int64       `json:"millisecondsPlayed"`
	Money              int         `json:"money"`
	Missing            []Lack      `json:"missing"`
	LastProfileID      string      `json:"lastProfileId"`
	LastProfileAt      int64       `json:"lastProfileAt"`
	LastProfileExists  bool        `json:"lastProfileExists"`
	LastMods           []PlayedMod `json:"lastMods"`
	LastMissing        []Lack      `json:"lastMissing"`
}

// Service exposes the save scan to the frontend.
type Service struct {
	home     string
	profiles *profile.Store
	settings *settings.Store
	meta     *meta.Client
	// scanners holds one saves scanner per implemented game that has a save folder.
	scanners map[string]*saves.Scanner
	// pinned holds scanners for profiles pinned to an install, by saves folder.
	pinnedMu sync.Mutex
	pinned   map[string]*saves.Scanner
	// Launches, when set, refuses restore while the game is launching or running.
	Launches *launchsvc.Service
	last     *Store
	busy     func() bool
	// schedMu guards lastScheduled, when the last scheduled backup pass ran, which the status read shares with the
	// RunScheduledBackups goroutine.
	schedMu       sync.Mutex
	lastScheduled map[string]time.Time
	// Emit is nil in tests that do not watch events.
	Emit func(name string, data any)
	// Enqueue queues downloads when FromSave cannot reuse a store item.
	Enqueue func(context.Context, []queue.Request) ([]queue.Item, error)
}

// NewService reads saves from the save folder of every implemented game that has one and caches scans in <datadir>/cache.
func NewService(home string, profiles *profile.Store, store *settings.Store, client *meta.Client) (*Service, error) {
	base, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	scanners := map[string]*saves.Scanner{}
	for _, id := range game.Implemented() {
		if !game.HasSaves(id) {
			continue
		}
		savesDir, err := game.SavesDir(home, store.Get(), id, "")
		if err != nil {
			// An enabled game that is not installed has no folder to resolve; it gets no scanner until a restart
			// finds it, like any other change of the selected install.
			log.Printf("saves: %s: %v", id, err)
			continue
		}
		scanners[id] = &saves.Scanner{Dir: savesDir, CacheDir: filepath.Join(base, "cache", id)}
	}
	return &Service{home: home, profiles: profiles, settings: store, meta: client, scanners: scanners, last: NewStore(base)}, nil
}

// scannerFor is the scanner of the saves folder profileID reads: its pinned install's, else the game's selected one.
func (s *Service) scannerFor(gameID, profileID string) (*saves.Scanner, error) {
	selected := s.scanners[gameID]
	pin := s.profiles.InstallOf(gameID, profileID)
	if selected == nil || pin == "" {
		return selected, nil
	}
	dir, err := game.SavesDir(s.home, s.settings.Get(), gameID, pin)
	if err != nil {
		return nil, err
	}
	s.pinnedMu.Lock()
	defer s.pinnedMu.Unlock()
	if sc, ok := s.pinned[dir]; ok {
		return sc, nil
	}
	if s.pinned == nil {
		s.pinned = map[string]*saves.Scanner{}
	}
	sc := &saves.Scanner{Dir: dir, CacheDir: filepath.Join(selected.CacheDir, "install-"+pin)}
	s.pinned[dir] = sc
	return sc, nil
}

// Saves scans the game's saves and compares each with the profile. Wails runs it off the UI thread; a first scan
// of large saves takes well under a second, and later calls read the cache.
func (s *Service) Saves(ctx context.Context, game, profileID string) ([]Fit, error) {
	scanner, err := s.scannerFor(game, profileID)
	if err != nil {
		return nil, err
	}
	if scanner == nil {
		return []Fit{}, nil
	}
	index, err := s.meta.Index(ctx)
	if err != nil {
		return nil, err
	}
	mods, err := s.profiles.Mods(game, profileID)
	if err != nil {
		return nil, err
	}
	present, enabled := haveMaps(mods)
	infos, err := scanner.Scan(index)
	if err != nil {
		log.Printf("save scan: %v", err)
	}
	dismissed := s.settings.Get().Dismissed
	fits := make([]Fit, len(infos))
	wanted := map[mod.ID]bool{}
	for i, in := range infos {
		lacks := saves.Lacking(in.Used, enabled, asIDs(dismissed[in.Folder]))
		fits[i] = Fit{
			Folder: in.Folder, Farm: in.Farm, Farmer: in.Farmer, Season: in.Season, Day: in.Day, Year: in.Year,
			Played: in.Played, WhichFarm: in.WhichFarm, MillisecondsPlayed: in.MillisecondsPlayed, Money: in.Money,
			Missing: make([]Lack, len(lacks)),
		}
		for j, l := range lacks {
			fits[i].Missing[j] = Lack{ID: l.ID, Name: l.ID.Local(), Disabled: l.Disabled}
			wanted[l.ID] = true
		}
		s.fillLast(game, &fits[i], present, enabled)
		for _, l := range fits[i].LastMissing {
			wanted[l.ID] = true
		}
	}
	names := s.describe(ctx, game, wanted)
	for i := range fits {
		for j := range fits[i].Missing {
			if d, ok := names[fits[i].Missing[j].ID]; ok {
				fits[i].Missing[j].Name, fits[i].Missing[j].Where = d.name, d.where
			}
		}
		for j := range fits[i].LastMissing {
			if d, ok := names[fits[i].LastMissing[j].ID]; ok {
				fits[i].LastMissing[j].Name, fits[i].LastMissing[j].Where = d.name, d.where
			}
		}
	}
	return fits, nil
}

func fitFor(in saves.Info, have map[string]bool, dismissed []mod.ID) (Fit, bool) {
	lacks := saves.Lacking(in.Used, have, dismissed)
	fit := Fit{
		Folder: in.Folder, Farm: in.Farm, Farmer: in.Farmer, Season: in.Season, Day: in.Day, Year: in.Year,
		Played: in.Played, WhichFarm: in.WhichFarm, MillisecondsPlayed: in.MillisecondsPlayed, Money: in.Money,
		Missing: make([]Lack, len(lacks)),
	}
	for i, l := range lacks {
		fit.Missing[i] = Lack{ID: l.ID, Name: l.ID.Local(), Disabled: l.Disabled}
	}
	return fit, len(lacks) > 0
}

type described struct {
	name  string
	where *problems.Ref
}

// describe looks up the display name and the page of each mod id. A page that cannot be fetched leaves the
// id as the name, so an offline machine still gets a usable list.
func (s *Service) describe(ctx context.Context, gameID string, ids map[mod.ID]bool) map[mod.ID]described {
	ctx, cancel := context.WithTimeout(ctx, nameTimeout)
	defer cancel()
	t, _ := game.NexusTitle(gameID)
	domain := t.Domain
	var mu sync.Mutex
	out := map[mod.ID]described{}
	var wg sync.WaitGroup
	sem := make(chan struct{}, nameWorkers)
	for id := range ids {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			where, _ := problems.Locate(ctx, s.meta, domain, id, "", nil)
			d := described{name: id.Local(), where: where}
			if where != nil && where.Site == "Nexus" {
				if page, err := s.meta.Page(ctx, where.PageID); err == nil {
					d.name = cmp.Or(page.Name, id.Local())
					for _, f := range page.Downloads {
						for _, m := range f.Mods {
							if mod.Equal(m.ModID(), id) && m.Name != "" {
								d.name = m.Name
							}
						}
					}
				}
			}
			mu.Lock()
			out[id] = d
			mu.Unlock()
		})
	}
	wg.Wait()
	return out
}

// LastSaveGap is the most recently written save when it uses mods the profile lacks or has switched off, the save
// the game most likely loads next; ok is false when that save has everything or there are no saves.
func (s *Service) LastSaveGap(ctx context.Context, game, profileID string) (fit Fit, ok bool, err error) {
	scanner, err := s.scannerFor(game, profileID)
	if err != nil {
		return Fit{}, false, err
	}
	if scanner == nil {
		return Fit{}, false, nil
	}
	index, err := s.meta.Index(ctx)
	if err != nil {
		return Fit{}, false, err
	}
	mods, err := s.profiles.Mods(game, profileID)
	if err != nil {
		return Fit{}, false, err
	}
	present, enabled := haveMaps(mods)
	info, err := scanner.Newest(index)
	if err != nil {
		return Fit{}, false, err
	}
	if info.Folder == "" {
		return Fit{}, false, nil
	}
	fit, ok = fitFor(info, enabled, asIDs(s.settings.Get().Dismissed[info.Folder]))
	s.fillLast(game, &fit, present, enabled)
	if len(fit.LastMissing) > 0 {
		ok = true
	}
	return fit, ok, nil
}

// Dismiss stops warning about uniqueID for the save folder.
func (s *Service) Dismiss(saveFolder string, uniqueID mod.ID) error {
	if saveFolder == "" || uniqueID == "" {
		return errors.New("save folder and mod are required")
	}
	return s.settings.AppendDismissed(saveFolder, uniqueID.Fold())
}

// RestoreDismissed shows uniqueID's missing-mod warning for the save folder again.
func (s *Service) RestoreDismissed(saveFolder string, uniqueID mod.ID) error {
	if saveFolder == "" || uniqueID == "" {
		return errors.New("save folder and mod are required")
	}
	id := uniqueID.Fold()
	_, err := s.settings.Update(func(v *settings.Settings) {
		current := v.Dismissed[saveFolder]
		kept := slices.DeleteFunc(slices.Clone(current), func(x string) bool { return x == id })
		if len(kept) == len(current) {
			return
		}
		next := maps.Clone(v.Dismissed)
		if len(kept) == 0 {
			delete(next, saveFolder)
		} else {
			next[saveFolder] = kept
		}
		v.Dismissed = next
	})
	return err
}

// asIDs reads settings tokens that hold folded mod ids.
func asIDs(tokens []string) []mod.ID {
	out := make([]mod.ID, len(tokens))
	for i, t := range tokens {
		out[i] = mod.ID(t)
	}
	return out
}
