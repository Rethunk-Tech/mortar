package problems

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

// Service exposes the problem checks to the frontend.
type Service struct {
	home     string
	settings *settings.Store
	profiles *profile.Store
	meta     Meta

	mu      sync.Mutex
	cache   map[string]cached
	updates map[string]cachedUpdates
}

type cachedUpdates struct {
	fingerprint string
	at          time.Time
	result      UpdatesResult
}

// cached is a result with the fingerprint of the mods and environment it was computed for.
type cached struct {
	fingerprint string
	result      Result
}

func NewService(home string, s *settings.Store, profiles *profile.Store, m *meta.Client) *Service {
	return &Service{home: home, settings: s, profiles: profiles, meta: m, cache: map[string]cached{}, updates: map[string]cachedUpdates{}}
}

func platform() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "Mac"
	}
	return "Linux"
}

// environment reads the versions from the game's logs; without an install they stay empty.
func (s *Service) environment(id string) Environment {
	env := Environment{Platform: platform()}
	g := game.Find(id)
	if g == nil {
		return env
	}
	set := s.settings.Get()
	dir, err := game.InstallDir(s.home, set.GameFolders, id)
	if err != nil || dir == "" {
		return env
	}
	st := g.LoaderStatus(dir, set.Loaders[id])
	env.GameVersion, env.APIVersion = st.GameVersion, st.Version
	return env
}

func fingerprint(env Environment, mods []Installed) string {
	var b strings.Builder
	b.WriteString(env.GameVersion + "|" + env.APIVersion)
	for _, m := range mods {
		b.WriteString("\n" + m.Key + "|" + m.UniqueID + "|" + m.Version + "|" + m.Name)
		if m.Enabled {
			b.WriteString("|on")
		}
		for _, d := range m.Dependencies {
			b.WriteString("|" + d.UniqueID + ">=" + d.MinimumVersion)
		}
	}
	return b.String()
}

func (s *Service) installed(gameID, id string) ([]Installed, error) {
	installed, err := s.profiles.Installed(gameID, id)
	if err != nil {
		return nil, err
	}
	mods := make([]Installed, len(installed))
	for i, m := range installed {
		mods[i] = Installed{Key: m.Key, SourceKind: m.Source.Kind, Enabled: m.Enabled, Manifest: m.Manifest}
	}
	return mods, nil
}

// Problems checks the profile's mods. The answer is kept until the mods or versions change, unless a lookup
// failed, in which case the next call tries again.
func (s *Service) Problems(ctx context.Context, gameID, id string) (Result, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return Result{}, err
	}
	env := s.environment(gameID)
	fp := fingerprint(env, mods)
	key := gameID + "/" + id
	s.mu.Lock()
	c, ok := s.cache[key]
	s.mu.Unlock()
	if ok && c.fingerprint == fp {
		return c.result, nil
	}
	r := Check(ctx, s.meta, env, mods)
	if !r.Unknown {
		s.mu.Lock()
		s.cache[key] = cached{fp, r}
		s.mu.Unlock()
	}
	return r, nil
}

// Updates lists the newer versions SMAPI's API suggests for the profile's mods. The answer is kept for an hour
// or until the mods or versions change, unless SMAPI's API could not be reached, in which case the next call
// tries again.
func (s *Service) Updates(ctx context.Context, gameID, id string) (UpdatesResult, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return UpdatesResult{}, err
	}
	env := s.environment(gameID)
	fp := fingerprint(env, mods)
	key := gameID + "/" + id
	s.mu.Lock()
	c, ok := s.updates[key]
	s.mu.Unlock()
	if ok && c.fingerprint == fp && time.Since(c.at) < updatesTTL {
		return c.result, nil
	}
	r := CheckUpdates(ctx, s.meta, env, mods)
	if !r.Unknown {
		s.mu.Lock()
		s.updates[key] = cachedUpdates{fp, time.Now(), r}
		s.mu.Unlock()
	}
	return r, nil
}

// Relations says what the mod key/uniqueID needs, which mods need it and where its page is.
func (s *Service) Relations(gameID, id, key, uniqueID string) (Relations, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return Relations{}, err
	}
	r, ok := Relate(mods, key, uniqueID)
	if !ok {
		return Relations{}, errors.New("no such mod in this profile")
	}
	return r, nil
}
