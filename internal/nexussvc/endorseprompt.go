package nexussvc

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

const (
	promptFileName     = "endorse-prompts.json"
	cleanSessionsToAsk = 5
)

// CleanMod is a Nexus mod that had no errors in a completed run.
type CleanMod struct {
	ModID       int    `json:"modId"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Endorsement string `json:"endorsement"`
}

// EndorsePrompt is a mod that has reached the clean-session threshold.
type EndorsePrompt struct {
	ModID   int    `json:"modId"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type promptProgress struct {
	Clean   int    `json:"clean"`
	Asked   bool   `json:"asked"`
	Never   bool   `json:"never"`
	AskedAt int    `json:"askedAt,omitempty"`
	LastRun string `json:"lastRun,omitempty"`
}

type promptStore struct {
	mu   sync.Mutex
	path string
	mods map[string]promptProgress
}

func openPromptStore(dir string) (*promptStore, error) {
	p := &promptStore{path: filepath.Join(dir, promptFileName), mods: map[string]promptProgress{}}
	found, err := datadir.ReadJSON(p.path, &p.mods)
	if err != nil {
		return nil, err
	}
	if !found {
		return p, nil
	}
	if p.mods == nil {
		p.mods = map[string]promptProgress{}
	}
	return p, nil
}

func (p *promptStore) writeLocked() error {
	return datadir.WriteJSON(p.path, p.mods)
}

func endorsementUndecided(status string) bool {
	status = strings.TrimSpace(status)
	return !strings.EqualFold(status, "Endorsed") && !strings.EqualFold(status, "Abstained")
}

func (p *promptStore) record(runID string, mods []CleanMod, ask bool) ([]EndorsePrompt, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	unique := make(map[int]CleanMod, len(mods))
	order := make([]int, 0, len(mods))
	for _, mod := range mods {
		if mod.ModID <= 0 {
			continue
		}
		if _, ok := unique[mod.ModID]; ok {
			continue
		}
		unique[mod.ModID] = mod
		order = append(order, mod.ModID)
	}

	changed := false
	for _, modID := range order {
		key := fmt.Sprint(modID)
		progress := p.mods[key]
		if progress.LastRun != runID {
			progress.Clean++
			progress.LastRun = runID
			changed = true
		}
		if progress.Asked && progress.Clean >= progress.AskedAt+cleanSessionsToAsk {
			progress.Asked = false
			progress.AskedAt = 0
			changed = true
		}
		p.mods[key] = progress
	}

	prompts := make([]EndorsePrompt, 0, 3)
	if ask {
		for _, modID := range order {
			mod := unique[modID]
			key := fmt.Sprint(modID)
			progress := p.mods[key]
			if progress.Clean < cleanSessionsToAsk || progress.Asked || progress.Never || !endorsementUndecided(mod.Endorsement) {
				continue
			}
			progress.Asked = true
			progress.AskedAt = progress.Clean
			p.mods[key] = progress
			prompts = append(prompts, EndorsePrompt{ModID: mod.ModID, Name: mod.Name, Version: mod.Version})
			changed = true
			if len(prompts) == 3 {
				break
			}
		}
	}
	if changed {
		if err := p.writeLocked(); err != nil {
			return nil, err
		}
	}
	return prompts, nil
}

func (p *promptStore) answer(modID int, never bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := fmt.Sprint(modID)
	progress, ok := p.mods[key]
	if !ok || modID <= 0 {
		return errors.New("nexus: unknown endorsement prompt")
	}
	progress.Asked = true
	if never {
		progress.Never = true
	} else {
		progress.AskedAt = progress.Clean
	}
	p.mods[key] = progress
	return p.writeLocked()
}

// RecordCleanRun adds one clean session for each mod and returns at most three prompts.
func (s *Service) RecordCleanRun(runID string, mods []CleanMod) ([]EndorsePrompt, error) {
	if s.prompts == nil {
		return nil, errors.New("nexus: endorse prompt store not open")
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, errors.New("nexus: run id is required")
	}
	current := s.store.Get()
	ask := current.AskEndorseMods == nil || *current.AskEndorseMods
	return s.prompts.record(runID, mods, ask && s.Account().SignedIn)
}

// EndorsePromptNotNow postpones a prompt for five more clean sessions.
func (s *Service) EndorsePromptNotNow(modID int) error {
	if s.prompts == nil {
		return errors.New("nexus: endorse prompt store not open")
	}
	return s.prompts.answer(modID, false)
}

// EndorsePromptNever stops prompts for a mod.
func (s *Service) EndorsePromptNever(modID int) error {
	if s.prompts == nil {
		return errors.New("nexus: endorse prompt store not open")
	}
	return s.prompts.answer(modID, true)
}
