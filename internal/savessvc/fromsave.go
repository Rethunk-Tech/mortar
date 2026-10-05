package savessvc

import (
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// FromSaveResult is a profile built from a save's last-played mod list.
type FromSaveResult struct {
	Profile profile.Profile `json:"profile"`
	Added   []string        `json:"added"`
	Queued  []string        `json:"queued"`
	Missing []string        `json:"missing"`
}

func sourceFromPlayed(m PlayedMod) profile.Source {
	return profile.Source{
		Kind: m.SourceKind, Name: m.Name, ModID: m.ModID, FileID: m.FileID,
		Version: m.Version, Repo: m.Repo, Tag: m.Tag, Asset: m.Asset,
	}
}

func queueReq(game, profileID string, m PlayedMod) (queue.Request, bool) {
	req := queue.Request{Kind: queue.KindInstall, Game: game, Profile: profileID, Name: m.Name, Version: m.Version}
	switch {
	case m.SourceKind == profile.KindNexus && m.ModID > 0:
		req.ModID, req.FileID, req.Latest = m.ModID, m.FileID, true
		return req, true
	case m.SourceKind == profile.KindGitHub && m.Repo != "":
		req.Repo, req.Tag, req.Asset = m.Repo, m.Tag, m.Asset
		return req, true
	default:
		return queue.Request{}, false
	}
}

func takenNames(all []profile.Profile) []string {
	out := make([]string, 0, len(all))
	for _, p := range all {
		out = append(out, p.Name)
	}
	return out
}

func (s *Service) enqueue(reqs []queue.Request) error {
	if len(reqs) == 0 {
		return nil
	}
	if s.Enqueue != nil {
		_, err := s.Enqueue(reqs)
		return err
	}
	return nil
}

func (s *Service) addRecorded(game, id string, mods []PlayedMod) (added, queued, missing []string, err error) {
	seen := map[string]bool{}
	var reqs []queue.Request
	for _, m := range mods {
		if m.Key == "" || seen[m.Key] {
			continue
		}
		seen[m.Key] = true
		_, addErr := s.profiles.AddEntry(game, id, m.Key, sourceFromPlayed(m))
		if addErr == nil {
			added = append(added, m.Key)
			continue
		}
		if usererr.KindOf(addErr) == usererr.NotFound {
			if req, ok := queueReq(game, id, m); ok {
				reqs = append(reqs, req)
				queued = append(queued, m.UniqueID)
				continue
			}
		}
		label := m.Name
		if label == "" {
			label = m.UniqueID
		}
		missing = append(missing, label)
	}
	return added, queued, missing, s.enqueue(reqs)
}

// FromSave creates a profile named after the farm with the save's last-played mods.
func (s *Service) FromSave(game, saveFolder string) (FromSaveResult, error) {
	if s.profiles == nil || s.last == nil {
		return FromSaveResult{}, errors.New("saves are unavailable")
	}
	if strings.TrimSpace(saveFolder) == "" {
		return FromSaveResult{}, errors.New("a save is required")
	}
	rec, ok, err := s.last.Get(game, saveFolder)
	if err != nil {
		return FromSaveResult{}, err
	}
	if !ok || len(rec.Mods) == 0 {
		return FromSaveResult{}, errors.New("this save has no recorded mod list")
	}
	all, err := s.profiles.List(game)
	if err != nil {
		return FromSaveResult{}, err
	}
	created, err := s.profiles.Create(game, profile.UniqueName(takenNames(all), s.farmOf(game, saveFolder)))
	if err != nil {
		return FromSaveResult{}, err
	}
	added, queued, missing, err := s.addRecorded(game, created.ID, rec.Mods)
	if err != nil {
		return FromSaveResult{Profile: created, Added: added, Queued: queued, Missing: missing}, err
	}
	if p, readErr := s.profiles.List(game); readErr == nil {
		for _, pr := range p {
			if pr.ID == created.ID {
				created = pr
				break
			}
		}
	}
	return FromSaveResult{Profile: created, Added: added, Queued: queued, Missing: missing}, nil
}
