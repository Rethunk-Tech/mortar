package problems

import (
	"context"
	"runtime"
	"strings"
	"sync"

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

	mu    sync.Mutex
	cache map[string]cached
}

// cached is a result with the fingerprint of the mods and environment it was computed for.
type cached struct {
	fingerprint string
	result      Result
}

func NewService(home string, s *settings.Store, profiles *profile.Store, m *meta.Client) *Service {
	return &Service{home: home, settings: s, profiles: profiles, meta: m, cache: map[string]cached{}}
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

// Problems checks the profile's mods. The answer is kept until the mods or versions change, unless a lookup
// failed, in which case the next call tries again.
func (s *Service) Problems(ctx context.Context, gameID, id string) (Result, error) {
	installed, err := s.profiles.Installed(gameID, id)
	if err != nil {
		return Result{}, err
	}
	mods := make([]Installed, len(installed))
	for i, m := range installed {
		mods[i] = Installed{Key: m.Key, SourceKind: m.Source.Kind, Enabled: m.Enabled, Manifest: m.Manifest}
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
