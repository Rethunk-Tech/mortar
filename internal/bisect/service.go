package bisect

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

const (
	StateStarting = "starting"
	StateRunning  = "running"
	StateStopped  = "stopped"
	StateDone     = "done"
	StateFailed   = "failed"
)

type ModResult struct {
	Key      string `json:"key"`
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
}

type Result struct {
	Mods []ModResult `json:"mods"`
}

type Status struct {
	ID       string  `json:"id"`
	Game     string  `json:"game"`
	Profile  string  `json:"profile"`
	State    string  `json:"state"`
	Step     int     `json:"step"`
	Total    int     `json:"total"`
	ModsLeft int     `json:"modsLeft"`
	Result   *Result `json:"result,omitempty"`
	Error    string  `json:"error,omitempty"`
}

type Service struct {
	profiles *profile.Store
	launches *launchsvc.Service
	seq      atomic.Uint64

	mu   sync.Mutex
	jobs map[string]*job
}

type job struct {
	status  Status
	id      string
	game    string
	profile string
	tempID  string
	mods    []profile.Mod
	refs    []profile.EnableRef
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewService(profiles *profile.Store, launches *launchsvc.Service) *Service {
	return &Service{
		profiles: profiles,
		launches: launches,
		jobs:     make(map[string]*job),
	}
}

// Start begins a crash check against a hidden duplicate of the requested profile.
func (s *Service) Start(gameID, profileID string) (string, error) {
	if gameID == "" || profileID == "" {
		return "", errors.New("game and profile are required")
	}
	temp, err := s.profiles.Duplicate(gameID, profileID)
	if err != nil {
		return "", err
	}
	if _, err := s.profiles.SetHidden(gameID, temp.ID, true); err != nil {
		_ = s.profiles.Delete(gameID, temp.ID)
		return "", err
	}
	mods, err := s.profiles.UserMods(gameID, temp.ID)
	if err != nil {
		_ = s.profiles.Delete(gameID, temp.ID)
		return "", err
	}
	var candidates []profile.Mod
	var refs []profile.EnableRef
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		candidates = append(candidates, mod)
		refs = append(refs, profile.EnableRef{Key: mod.Key, UniqueID: mod.UniqueID})
	}
	if len(candidates) == 0 {
		_ = s.profiles.Delete(gameID, temp.ID)
		return "", errors.New("the profile has no enabled user mods")
	}

	ctx, cancel := context.WithCancel(context.Background())
	id := fmt.Sprintf("bisect-%d", s.seq.Add(1))
	j := &job{
		id:      id,
		game:    gameID,
		profile: profileID,
		tempID:  temp.ID,
		mods:    candidates,
		refs:    refs,
		ctx:     ctx,
		cancel:  cancel,
	}
	s.mu.Lock()
	s.jobs[id] = j
	s.mu.Unlock()
	s.setStatus(Status{ID: id, Game: gameID, Profile: profileID, State: StateStarting, ModsLeft: len(candidates)})
	go s.run(j)
	return id, nil
}

func (s *Service) run(j *job) {
	defer j.cancel()
	defer func() {
		_ = s.profiles.Delete(j.game, j.tempID)
	}()

	mods := make([]Mod, 0, len(j.mods))
	byID := make(map[string]profile.Mod, len(j.mods))
	for _, mod := range j.mods {
		id := modID(mod)
		mods = append(mods, Mod{
			ID:           id,
			Dependencies: append([]string(nil), mod.Needs...),
			Group:        mod.Key,
		})
		byID[id] = mod
	}
	found, err := find(j.ctx, mods, func(ctx context.Context, disabled []string) (bool, error) {
		if _, err := s.profiles.SetModsEnabled(j.game, j.tempID, j.refs, true); err != nil {
			return false, err
		}
		closure := make(map[string]bool, len(disabled))
		for _, id := range disabled {
			closure[id] = true
		}
		refs := make([]profile.EnableRef, 0, len(disabled))
		for _, mod := range j.mods {
			if closure[modID(mod)] {
				refs = append(refs, profile.EnableRef{Key: mod.Key, UniqueID: mod.UniqueID})
			}
		}
		if _, err := s.profiles.SetModsEnabled(j.game, j.tempID, refs, false); err != nil {
			return false, err
		}
		healthy, _, err := s.launches.RunForBisect(ctx, j.game, j.tempID, false)
		return !healthy, err
	}, func(progress Progress) {
		s.setStatus(Status{
			ID:       j.id,
			Game:     j.game,
			Profile:  j.profile,
			State:    StateRunning,
			Step:     progress.Step,
			Total:    progress.Total,
			ModsLeft: progress.ModsLeft,
		})
	})
	if err != nil {
		state := StateFailed
		if errors.Is(err, context.Canceled) {
			state = StateStopped
		}
		s.setStatus(Status{ID: j.id, Game: j.game, Profile: j.profile, State: state, Error: err.Error()})
		return
	}
	result := &Result{}
	for _, mod := range found {
		original, ok := byID[mod.ID]
		if !ok {
			continue
		}
		result.Mods = append(result.Mods, ModResult{Key: original.Key, UniqueID: original.UniqueID, Name: original.Name})
	}
	s.setStatus(Status{
		ID:       j.id,
		Game:     j.game,
		Profile:  j.profile,
		State:    StateDone,
		Step:     progressSteps(len(dependencyGroups(mods))),
		Total:    progressSteps(len(dependencyGroups(mods))),
		ModsLeft: len(result.Mods),
		Result:   result,
	})
}

func progressSteps(groups int) int {
	steps := 0
	for groups > 1 {
		steps++
		groups = (groups + 1) / 2
	}
	return steps
}

func (s *Service) setStatus(status Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[status.ID]; ok {
		jobStatus := status
		jobStatus.Result = cloneResult(status.Result)
		job.status = jobStatus
	}
}

func cloneResult(result *Result) *Result {
	if result == nil {
		return nil
	}
	return &Result{Mods: append([]ModResult(nil), result.Mods...)}
}

// Status returns the current progress of a crash check.
func (s *Service) Status(id string) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return Status{}, fmt.Errorf("unknown crash check %q", id)
	}
	status := j.status
	status.Result = cloneResult(status.Result)
	return status, nil
}

// Stop cancels a crash check and its temporary launch.
func (s *Service) Stop(id string) error {
	s.mu.Lock()
	j, ok := s.jobs[id]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("unknown crash check %q", id)
	}
	j.cancel()
	return nil
}

// SwitchOff disables the identified mod in the user's real profile.
func (s *Service) SwitchOff(gameID, profileID, key, uniqueID string) error {
	if key == "" && uniqueID == "" {
		return errors.New("mod identity is required")
	}
	_, err := s.profiles.SetModEnabled(gameID, profileID, key, uniqueID, false)
	return err
}

func modID(mod profile.Mod) string {
	if mod.UniqueID != "" {
		return mod.UniqueID
	}
	return mod.Key
}
