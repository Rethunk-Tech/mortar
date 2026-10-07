package savessvc

import (
	"cmp"
	"context"
	"errors"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// SaveCheck is how a save's last-played mod list compares to a profile.
type SaveCheck struct {
	Folder            string      `json:"folder"`
	Farm              string      `json:"farm"`
	LastProfileID     string      `json:"lastProfileId"`
	LastProfileExists bool        `json:"lastProfileExists"`
	LastMods          []PlayedMod `json:"lastMods"`
	Missing           []Lack      `json:"missing"`
	ContentMissing    int         `json:"contentMissing"`
}

func contentMissingCount(recorded []PlayedMod, missing []Lack) int {
	ids := map[string]bool{}
	for _, m := range recorded {
		if m.ContentPackFor != "" {
			ids[m.ID.Fold()] = true
		}
	}
	n := 0
	for _, l := range missing {
		if ids[l.ID.Fold()] {
			n++
		}
	}
	return n
}

func (s *Service) farmOf(gameID, folder string) string {
	scanner := s.scanners[gameID]
	if scanner == nil || folder == "" {
		return folder
	}
	infos, err := scanner.Scan(nil)
	if err != nil {
		return folder
	}
	for _, in := range infos {
		if in.Folder == folder {
			if in.Farm != "" {
				return in.Farm
			}
			return folder
		}
	}
	return folder
}

func (s *Service) describeLacks(ctx context.Context, gameID string, lacks []Lack) []Lack {
	wanted := map[mod.ID]bool{}
	for _, l := range lacks {
		wanted[l.ID] = true
	}
	names := s.describe(ctx, gameID, wanted)
	for i := range lacks {
		if d, ok := names[lacks[i].ID]; ok {
			lacks[i].Name, lacks[i].Where = d.name, d.where
		}
	}
	return lacks
}

// Check compares the save's last-played mods with the profile. An empty profileID uses the recorded one.
func (s *Service) Check(ctx context.Context, game, saveFolder, profileID string) (SaveCheck, error) {
	if saveFolder == "" {
		return SaveCheck{}, errors.New("a save is required")
	}
	if s.last == nil {
		return SaveCheck{Folder: saveFolder, Farm: s.farmOf(game, saveFolder)}, nil
	}
	rec, ok, err := s.last.Get(game, saveFolder)
	if err != nil {
		return SaveCheck{}, err
	}
	out := SaveCheck{Folder: saveFolder, Farm: s.farmOf(game, saveFolder)}
	if !ok {
		return out, nil
	}
	out.LastProfileID = rec.ProfileID
	out.LastMods = rec.Mods
	out.LastProfileExists = profileExists(s.profiles, game, rec.ProfileID)
	id := cmp.Or(profileID, rec.ProfileID)
	if s.profiles == nil || id == "" {
		out.Missing = MissingFrom(rec.Mods, nil, nil)
		out.ContentMissing = contentMissingCount(rec.Mods, out.Missing)
		return out, nil
	}
	mods, err := s.profiles.Mods(game, id)
	if err != nil {
		return SaveCheck{}, err
	}
	present, enabled := haveMaps(mods)
	out.Missing = s.describeLacks(ctx, game, MissingFrom(rec.Mods, present, enabled))
	out.ContentMissing = contentMissingCount(rec.Mods, out.Missing)
	return out, nil
}
