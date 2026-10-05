package control

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/bisect"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
)

// queueAdd queues one mod from a source for the profile: Nexus by mod id (File is a file id, else the queue
// resolves one), GitHub by owner/repo (Version is the tag, File the asset), Thunderstore by Namespace-Name
// (Version, else the newest, with its dependencies).
func (s *Services) queueAdd(p Params) (any, error) {
	if s.Queue == nil {
		return nil, errors.New("the download queue is unavailable")
	}
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	req := queue.Request{Kind: queue.KindInstall, Game: p.Game, Profile: prof.ID, Name: p.ID, Version: p.Version}
	switch p.Source {
	case "nexus":
		if req.ModID, err = strconv.Atoi(p.ID); err != nil || req.ModID < 1 {
			return nil, fmt.Errorf("nexus mod id %q is not a number", p.ID)
		}
		if p.File != "" {
			if req.FileID, err = strconv.Atoi(p.File); err != nil || req.FileID < 1 {
				return nil, fmt.Errorf("nexus file id %q is not a number", p.File)
			}
		}
	case "github":
		req.Repo, req.Tag, req.Asset, req.FileName = p.ID, p.Version, p.File, p.File
	case "thunderstore":
		req.Package = p.ID
	default:
		return nil, fmt.Errorf("source %q cannot be queued; use nexus, github or thunderstore", p.Source)
	}
	if _, err := s.Queue.Add([]queue.Request{req}); err != nil {
		return nil, err
	}
	return s.Queue.State(), nil
}

// profileSet writes one scalar field of the profile; a key that is no field is a per-profile override of a game
// setting, as before.
func (s *Services) profileSet(p Params, prof profile.Profile, id string) (any, error) {
	g, v := p.Game, p.Value
	flag := func() (bool, error) {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, fmt.Errorf("%s needs true or false, got %q", p.Key, v)
		}
		return b, nil
	}
	switch p.Key {
	case "notes":
		return s.Profiles.SetNotes(g, id, v)
	case "color":
		return s.Profiles.SetAppearance(g, id, v, prof.Icon, prof.Description)
	case "icon":
		return s.Profiles.SetAppearance(g, id, prof.Color, v, prof.Description)
	case "description":
		return s.Profiles.SetAppearance(g, id, prof.Color, prof.Icon, v)
	case "install":
		return s.Profiles.SetInstall(g, id, v)
	case "launchOptions":
		return s.Profiles.SetLaunchOptions(g, id, v)
	case "launchPrefix":
		return s.Profiles.SetLaunchSettings(g, id, v, prof.LaunchEnv)
	case "launchEnv":
		return s.Profiles.SetLaunchSettings(g, id, prof.LaunchPrefix, strings.ReplaceAll(v, `\n`, "\n"))
	case "defaultLaunchPreset":
		return s.Profiles.SetDefaultLaunchPreset(g, id, v)
	case "skipPlayCheck":
		on, err := flag()
		if err != nil {
			return nil, err
		}
		return s.Profiles.SetSkipPlayCheck(g, id, on)
	case "hidden":
		on, err := flag()
		if err != nil {
			return nil, err
		}
		return s.Profiles.SetHidden(g, id, on)
	case "cover":
		if v == "" {
			return s.Profiles.ClearCover(g, id)
		}
		return s.Profiles.SetCover(g, id, v)
	}
	return s.Profiles.SetOverride(g, id, p.Key, v)
}

// storeCheck verifies the store's files and lists the damaged items of game (every game when empty).
func (s *Services) storeCheck(ctx context.Context, game string) (any, error) {
	if s.StoreCheck == nil {
		return nil, errors.New("the store check is unavailable")
	}
	sum, err := s.StoreCheck.Check(ctx)
	if err != nil {
		return nil, err
	}
	if game != "" {
		kept := sum.Damaged[:0]
		for _, d := range sum.Damaged {
			if d.Game == game {
				kept = append(kept, d)
			}
		}
		sum.Damaged = kept
	}
	return sum, nil
}

func (s *Services) storeRepair(p Params, profileID string) (any, error) {
	if s.StoreCheck == nil {
		return nil, errors.New("the store check is unavailable")
	}
	if p.Key == "" {
		return nil, errors.New("store repair needs a store key")
	}
	return s.StoreCheck.Repair(p.Game, profileID, p.Key)
}

func (s *Services) bisectStart(p Params) (any, error) {
	if s.Bisect == nil {
		return nil, errors.New("crash check is unavailable")
	}
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	id, err := s.Bisect.Start(p.Game, prof.ID)
	if err != nil {
		return nil, err
	}
	return s.Bisect.Status(id)
}

func (s *Services) bisectStatus(id string) (bisect.Status, error) {
	if s.Bisect == nil {
		return bisect.Status{}, errors.New("crash check is unavailable")
	}
	return s.Bisect.Status(id)
}

func (s *Services) bisectStop(id string) (bisect.Status, error) {
	if s.Bisect == nil {
		return bisect.Status{}, errors.New("crash check is unavailable")
	}
	if err := s.Bisect.Stop(id); err != nil {
		return bisect.Status{}, err
	}
	return s.Bisect.Status(id)
}
