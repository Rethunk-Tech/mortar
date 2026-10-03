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
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/queue"
	"github.com/Rethunk-AI/mortar/internal/saves"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

const (
	nameTimeout = 15 * time.Second
	nameWorkers = 8
)

// Lack is a mod the save has used that the profile does not run. Name falls back to the UniqueID. Where is the page
// to get it from, found the way a missing dependency's is, and nil when the mod is unknown or the dataset could not
// be reached; a Nexus page or a GitHub repository in it can be queued into the profile.
type Lack struct {
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
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
	scanner  *saves.Scanner
	// Launches, when set, refuses restore while the game is launching or running.
	Launches *launchsvc.Service
	last     *Store
	busy     func() bool
	// Enqueue queues downloads when FromSave cannot reuse a store item.
	Enqueue func([]queue.Request) ([]queue.Item, error)
}

// NewService reads saves from the Stardew Valley Saves folder and caches scans in <datadir>/cache.
func NewService(home string, profiles *profile.Store, store *settings.Store, client *meta.Client) (*Service, error) {
	_, selected, _, err := game.Resolve(home, store.Get(), "stardew")
	if err != nil {
		return nil, err
	}
	savesDir, err := game.SavesDir(selected, home)
	if err != nil {
		return nil, err
	}
	base, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	scanner := &saves.Scanner{Dir: savesDir, CacheDir: filepath.Join(base, "cache")}
	return &Service{home: home, profiles: profiles, settings: store, meta: client, scanner: scanner, last: NewStore(base)}, nil
}

// Saves scans the game's saves and compares each with the profile. Wails runs it off the UI thread; a first scan
// of large saves takes well under a second, and later calls read the cache.
func (s *Service) Saves(ctx context.Context, game, profileID string) ([]Fit, error) {
	if game != "stardew" {
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
	infos, err := s.scanner.Scan(index)
	if err != nil {
		log.Printf("save scan: %v", err)
	}
	dismissed := s.settings.Get().Dismissed
	fits := make([]Fit, len(infos))
	wanted := map[string]bool{}
	for i, in := range infos {
		lacks := saves.Lacking(in.Used, enabled, dismissed[in.Folder])
		fits[i] = Fit{
			Folder: in.Folder, Farm: in.Farm, Farmer: in.Farmer, Season: in.Season, Day: in.Day, Year: in.Year,
			Played: in.Played, WhichFarm: in.WhichFarm, MillisecondsPlayed: in.MillisecondsPlayed, Money: in.Money,
			Missing: make([]Lack, len(lacks)),
		}
		for j, l := range lacks {
			fits[i].Missing[j] = Lack{UniqueID: l.UniqueID, Name: l.UniqueID, Disabled: l.Disabled}
			wanted[l.UniqueID] = true
		}
		s.fillLast(game, &fits[i], present, enabled)
		for _, l := range fits[i].LastMissing {
			wanted[l.UniqueID] = true
		}
	}
	names := s.describe(ctx, wanted)
	for i := range fits {
		for j := range fits[i].Missing {
			if d, ok := names[fits[i].Missing[j].UniqueID]; ok {
				fits[i].Missing[j].Name, fits[i].Missing[j].Where = d.name, d.where
			}
		}
		for j := range fits[i].LastMissing {
			if d, ok := names[fits[i].LastMissing[j].UniqueID]; ok {
				fits[i].LastMissing[j].Name, fits[i].LastMissing[j].Where = d.name, d.where
			}
		}
	}
	return fits, nil
}

func fitFor(in saves.Info, have map[string]bool, dismissed []string) (Fit, bool) {
	lacks := saves.Lacking(in.Used, have, dismissed)
	fit := Fit{
		Folder: in.Folder, Farm: in.Farm, Farmer: in.Farmer, Season: in.Season, Day: in.Day, Year: in.Year,
		Played: in.Played, WhichFarm: in.WhichFarm, MillisecondsPlayed: in.MillisecondsPlayed, Money: in.Money,
		Missing: make([]Lack, len(lacks)),
	}
	for i, l := range lacks {
		fit.Missing[i] = Lack{UniqueID: l.UniqueID, Name: l.UniqueID, Disabled: l.Disabled}
	}
	return fit, len(lacks) > 0
}

type described struct {
	name  string
	where *problems.Ref
}

// describe looks up the display name and the page of each UniqueID. A page that cannot be fetched leaves the
// UniqueID as the name, so an offline machine still gets a usable list.
func (s *Service) describe(ctx context.Context, ids map[string]bool) map[string]described {
	ctx, cancel := context.WithTimeout(ctx, nameTimeout)
	defer cancel()
	var mu sync.Mutex
	out := map[string]described{}
	var wg sync.WaitGroup
	sem := make(chan struct{}, nameWorkers)
	for id := range ids {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			where, _ := problems.Locate(ctx, s.meta, id, "", nil)
			d := described{name: id, where: where}
			if where != nil && where.Site == "Nexus" {
				if page, err := s.meta.Page(ctx, where.PageID); err == nil {
					d.name = cmp.Or(page.Name, id)
					for _, f := range page.Downloads {
						for _, m := range f.Mods {
							if strings.EqualFold(m.UniqueID, id) && m.Name != "" {
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
	if game != "stardew" {
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
	info, err := s.scanner.Newest(index)
	if err != nil {
		return Fit{}, false, err
	}
	if info.Folder == "" {
		return Fit{}, false, nil
	}
	fit, ok = fitFor(info, enabled, s.settings.Get().Dismissed[info.Folder])
	s.fillLast(game, &fit, present, enabled)
	if len(fit.LastMissing) > 0 {
		ok = true
	}
	return fit, ok, nil
}

// lastGap is the most recently written save and whether it lacks mods.
func lastGap(fits []Fit) (Fit, bool) {
	if len(fits) == 0 {
		return Fit{}, false
	}
	last := fits[0]
	for _, f := range fits[1:] {
		if f.Played > last.Played {
			last = f
		}
	}
	return last, len(last.Missing) > 0
}

// Dismiss stops warning about uniqueID for the save folder.
func (s *Service) Dismiss(saveFolder, uniqueID string) error {
	if saveFolder == "" || uniqueID == "" {
		return errors.New("save folder and mod are required")
	}
	id := strings.ToLower(uniqueID)
	_, err := s.settings.Update(func(v *settings.Settings) {
		if slices.Contains(v.Dismissed[saveFolder], id) {
			return
		}
		next := maps.Clone(v.Dismissed)
		next[saveFolder] = append(slices.Clone(v.Dismissed[saveFolder]), id)
		v.Dismissed = next
	})
	return err
}

// RestoreDismissed shows uniqueID's missing-mod warning for the save folder again.
func (s *Service) RestoreDismissed(saveFolder, uniqueID string) error {
	if saveFolder == "" || uniqueID == "" {
		return errors.New("save folder and mod are required")
	}
	id := strings.ToLower(uniqueID)
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
