package dlwatch

import (
	"fmt"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/profile"
)

// Event names for the binding generator.
const ArrivedEvent = "dlwatch:arrived"

const (
	StateAsked     = "asked"
	StateInstalled = "installed"
	StateIgnored   = "ignored"
)

// Item is one archive noticed this process.
type Item struct {
	N           int    `json:"n"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	ModID       int    `json:"modId"`
	State       string `json:"state"`
	ProfileID   string `json:"profileId"`
	ProfileName string `json:"profileName"`
	Game        string `json:"game"`
}

// Arrival is the window toast payload.
type Arrival struct {
	N           int    `json:"n"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	ModID       int    `json:"modId"`
	ProfileID   string `json:"profileId"`
	ProfileName string `json:"profileName"`
	Game        string `json:"game"`
}

// Deps wires the watcher to settings, the open profile, and archive install.
type Deps struct {
	Dir     func() string
	Enabled func() bool
	Current func() (game, profileID, profileName string, err error)
	Install func(game, profileID, path string, src profile.Source) (profile.InstallResult, error)
	Emit    func(name string, data any)
}

// Service watches Downloads and keeps this session's list.
type Service struct {
	deps  Deps
	mu    sync.Mutex
	items []Item
	watch *Folder
}

func New(d Deps) *Service {
	s := &Service{deps: d}
	go s.loop()
	return s
}

func (s *Service) loop() {
	s.pollOnce()
	t := time.NewTicker(pollEvery)
	defer t.Stop()
	for range t.C {
		s.pollOnce()
	}
}

func (s *Service) dir() string {
	if s.deps.Dir != nil {
		if p := s.deps.Dir(); p != "" {
			return p
		}
	}
	return UserDir()
}

func (s *Service) enabled() bool {
	return s.deps.Enabled == nil || s.deps.Enabled()
}

func nexusSource(name string, modID int) profile.Source {
	if modID <= 0 {
		return profile.Source{}
	}
	return profile.Source{Kind: profile.KindNexus, Name: name, ModID: modID}
}

// List is the session table for the CLI and tests.
func (s *Service) List() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Item, len(s.items))
	copy(out, s.items)
	return out
}

// Ignore remembers path for this session so it is not asked again.
func (s *Service) Ignore(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, it := range s.items {
		if it.Path == path {
			s.items[i].State = StateIgnored
			return nil
		}
	}
	return fmt.Errorf("unknown download %q", path)
}

// Install puts the 1-based list entry into the open profile.
func (s *Service) Install(n int) (profile.InstallResult, error) {
	s.mu.Lock()
	if n < 1 || n > len(s.items) {
		s.mu.Unlock()
		return profile.InstallResult{}, fmt.Errorf("no download %d", n)
	}
	it := s.items[n-1]
	s.mu.Unlock()
	if s.deps.Install == nil {
		return profile.InstallResult{}, fmt.Errorf("install is not available")
	}
	game, profileID, _, err := s.current()
	if err != nil {
		return profile.InstallResult{}, err
	}
	if it.Game != "" {
		game = it.Game
	}
	if it.ProfileID != "" {
		profileID = it.ProfileID
	}
	res, err := s.deps.Install(game, profileID, it.Path, nexusSource(it.Name, it.ModID))
	if err != nil {
		return res, err
	}
	s.mu.Lock()
	if n >= 1 && n <= len(s.items) && s.items[n-1].Path == it.Path {
		s.items[n-1].State = StateInstalled
	}
	s.mu.Unlock()
	return res, nil
}

func (s *Service) current() (string, string, string, error) {
	if s.deps.Current == nil {
		return "", "", "", fmt.Errorf("no open profile")
	}
	return s.deps.Current()
}

func (s *Service) watcher() *Folder {
	if s.watch != nil {
		return s.watch
	}
	s.watch = &Folder{Dir: s.dir()}
	return s.watch
}

// Notice injects a finished archive (tests and the poll loop).
func (s *Service) notice(path string) (Item, error) {
	inf := Identify(path)
	game, profileID, profileName, err := s.current()
	if err != nil {
		game, profileID, profileName = "", "", ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.items {
		if it.Path == path {
			return it, nil
		}
	}
	it := Item{
		N: len(s.items) + 1, Path: path, Name: inf.Name, ModID: inf.ModID,
		State: StateAsked, ProfileID: profileID, ProfileName: profileName, Game: game,
	}
	s.items = append(s.items, it)
	if s.deps.Emit != nil {
		s.deps.Emit(ArrivedEvent, Arrival{
			N: it.N, Path: it.Path, Name: it.Name, ModID: it.ModID,
			ProfileID: it.ProfileID, ProfileName: it.ProfileName, Game: it.Game,
		})
	}
	return it, nil
}

// PollOnce snapshots on first use, then notices newly stable archives.
func (s *Service) pollOnce() {
	if !s.enabled() {
		return
	}
	f := s.watcher()
	if !f.started {
		_ = f.Snapshot()
		f.started = true
		return
	}
	for _, path := range f.Poll() {
		_, _ = s.notice(path)
	}
}
