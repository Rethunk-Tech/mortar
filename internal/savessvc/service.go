// Package savessvc tells the frontend how well each save fits a profile.
package savessvc

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/saves"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

const (
	nexusMod    = "https://www.nexusmods.com/stardewvalley/mods/"
	nameTimeout = 15 * time.Second
	nameWorkers = 8
)

// Lack is a mod the save has used that the profile does not run. Name falls back to the UniqueID and URL is the
// mod's Nexus page, both from the mod dataset; URL is empty when the dataset lists no Nexus page.
type Lack struct {
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
	// Disabled means the profile has the mod switched off; otherwise it is absent.
	Disabled bool   `json:"disabled"`
	URL      string `json:"url"`
}

// Fit is one save and what it has used that the profile lacks. Season is 0 (spring) to 3 (winter); Day is 0
// when unknown. Played is when the save was last written, in Unix milliseconds.
type Fit struct {
	Folder  string `json:"folder"`
	Farm    string `json:"farm"`
	Farmer  string `json:"farmer"`
	Season  int    `json:"season"`
	Day     int    `json:"day"`
	Year    int    `json:"year"`
	Played  int64  `json:"played"`
	Missing []Lack `json:"missing"`
}

// Service exposes the save scan to the frontend.
type Service struct {
	profiles *profile.Store
	settings *settings.Store
	meta     *meta.Client
	scanner  *saves.Scanner
}

// NewService reads saves from the Stardew Valley Saves folder and caches scans in <datadir>/cache.
func NewService(profiles *profile.Store, store *settings.Store, client *meta.Client) (*Service, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	base, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	scanner := &saves.Scanner{Dir: filepath.Join(cfg, "StardewValley", "Saves"), CacheDir: filepath.Join(base, "cache")}
	return &Service{profiles: profiles, settings: store, meta: client, scanner: scanner}, nil
}

// Saves scans the game's saves and compares each with the profile. Wails runs it off the UI thread; a first scan
// of large saves takes well under a second, and later calls read the cache.
func (s *Service) Saves(game, profileID string) ([]Fit, error) {
	if game != "stardew" {
		return []Fit{}, nil
	}
	ctx := context.Background()
	index, err := s.meta.Index(ctx)
	if err != nil {
		return nil, err
	}
	mods, err := s.profiles.Mods(game, profileID)
	if err != nil {
		return nil, err
	}
	have := map[string]bool{}
	for _, m := range mods {
		id := strings.ToLower(m.UniqueID)
		have[id] = have[id] || m.Enabled
	}
	infos, err := s.scanner.Scan(index)
	if err != nil {
		log.Printf("save scan: %v", err)
	}
	dismissed := s.settings.Get().Dismissed
	fits := make([]Fit, len(infos))
	wanted := map[string]bool{}
	for i, in := range infos {
		lacks := saves.Lacking(in.Used, have, dismissed[in.Folder])
		fits[i] = Fit{
			Folder: in.Folder, Farm: in.Farm, Farmer: in.Farmer, Season: in.Season, Day: in.Day, Year: in.Year,
			Played: in.Played, Missing: make([]Lack, len(lacks)),
		}
		for j, l := range lacks {
			fits[i].Missing[j] = Lack{UniqueID: l.UniqueID, Name: l.UniqueID, Disabled: l.Disabled}
			wanted[l.UniqueID] = true
		}
	}
	names := s.describe(ctx, index, wanted)
	for i := range fits {
		for j := range fits[i].Missing {
			if d, ok := names[fits[i].Missing[j].UniqueID]; ok {
				fits[i].Missing[j].Name, fits[i].Missing[j].URL = d.name, d.url
			}
		}
	}
	return fits, nil
}

type described struct{ name, url string }

// describe looks up the display name and Nexus page of each UniqueID. A page that cannot be fetched leaves the
// UniqueID as the name, so an offline machine still gets a usable list.
func (s *Service) describe(ctx context.Context, index map[string][]meta.Ref, ids map[string]bool) map[string]described {
	ctx, cancel := context.WithTimeout(ctx, nameTimeout)
	defer cancel()
	var mu sync.Mutex
	out := map[string]described{}
	var wg sync.WaitGroup
	sem := make(chan struct{}, nameWorkers)
	for id := range ids {
		i := slices.IndexFunc(index[id], func(r meta.Ref) bool { return r.Site == "Nexus" })
		if i < 0 {
			continue
		}
		ref := index[id][i]
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			d := described{url: nexusMod + strconv.Itoa(ref.ID)}
			if page, err := s.meta.Page(ctx, ref.ID); err == nil {
				d.name = page.Name
				for _, f := range page.Downloads {
					for _, m := range f.Mods {
						if strings.EqualFold(m.UniqueID, id) && m.Name != "" {
							d.name = m.Name
						}
					}
				}
			}
			if d.name == "" {
				d.name = id
			}
			mu.Lock()
			out[id] = d
			mu.Unlock()
		})
	}
	wg.Wait()
	return out
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
		next := make(map[string][]string, len(v.Dismissed)+1)
		for k, ids := range v.Dismissed {
			next[k] = ids
		}
		next[saveFolder] = append(slices.Clone(v.Dismissed[saveFolder]), id)
		v.Dismissed = next
	})
	return err
}
