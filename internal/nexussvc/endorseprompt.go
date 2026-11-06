package nexussvc

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/go-nexusmods"

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

// promptStore keys progress "game:modId". A bare "modId" key is what releases before game scoping wrote; it names no
// game, so the first time a game touches that id the bare entry is copied to the game's key, and it is read only then.
type promptStore struct {
	mu   sync.Mutex
	path string
	mods map[string]promptProgress
}

func promptKey(gameID string, modID int) string {
	return gameID + ":" + strconv.Itoa(modID)
}

// progressLocked returns the game's progress for modID, seeded from the legacy bare entry when the game has none.
func (p *promptStore) progressLocked(gameID string, modID int) (string, promptProgress, bool) {
	key := promptKey(gameID, modID)
	if progress, ok := p.mods[key]; ok {
		return key, progress, true
	}
	progress, ok := p.mods[strconv.Itoa(modID)]
	return key, progress, ok
}

// pruneLocked drops the game's entries for mods not in installed and reports whether any went.
func (p *promptStore) pruneLocked(gameID string, installed map[int]bool) bool {
	pruned := false
	prefix := gameID + ":"
	for key := range p.mods {
		id, ok := strings.CutPrefix(key, prefix)
		if !ok {
			continue
		}
		if n, err := strconv.Atoi(id); err == nil && !installed[n] {
			delete(p.mods, key)
			pruned = true
		}
	}
	return pruned
}

func openPromptStore(dir string) (*promptStore, error) {
	p := &promptStore{path: filepath.Join(dir, promptFileName), mods: map[string]promptProgress{}}
	found, err := datadir.ReadJSONOrQuarantine(p.path, &p.mods)
	if err != nil {
		return nil, err
	}
	if !found {
		p.mods = map[string]promptProgress{}
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
	return !strings.EqualFold(status, string(nexusmods.EndorseEndorsed)) && !strings.EqualFold(status, string(nexusmods.EndorseAbstained))
}

// record counts one clean run for each mod in gameID. installed, when non-nil, is every Nexus mod id the game's
// profiles hold, and entries for other ids are dropped.
func (p *promptStore) record(gameID, runID string, mods []CleanMod, ask bool, installed map[int]bool) ([]EndorsePrompt, error) {
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
	if installed != nil {
		changed = p.pruneLocked(gameID, installed)
	}
	for _, modID := range order {
		key, progress, _ := p.progressLocked(gameID, modID)
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
			key := promptKey(gameID, modID)
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

func (p *promptStore) answer(gameID string, modID int, never bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key, progress, ok := p.progressLocked(gameID, modID)
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
func (s *Service) RecordCleanRun(gameID, runID string, mods []CleanMod) ([]EndorsePrompt, error) {
	if s.prompts == nil {
		return nil, errors.New("nexus: endorse prompt store not open")
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, errors.New("nexus: run id is required")
	}
	gameID = strings.TrimSpace(gameID)
	if gameID == "" {
		return nil, errors.New("nexus: game is required")
	}
	// Without the installed set nothing is pruned: a failed read must not erase a game's progress.
	installed, err := s.profileModIDs(gameID)
	if err != nil {
		installed = nil
	}
	current := s.store.Get()
	ask := current.AskEndorseMods != nil && *current.AskEndorseMods
	return s.prompts.record(gameID, runID, mods, ask && s.Account().SignedIn, installed)
}

// EndorsePromptNotNow postpones a prompt for five more clean sessions.
func (s *Service) EndorsePromptNotNow(gameID string, modID int) error {
	if s.prompts == nil {
		return errors.New("nexus: endorse prompt store not open")
	}
	return s.prompts.answer(gameID, modID, false)
}

// EndorsePromptNever stops prompts for a mod.
func (s *Service) EndorsePromptNever(gameID string, modID int) error {
	if s.prompts == nil {
		return errors.New("nexus: endorse prompt store not open")
	}
	return s.prompts.answer(gameID, modID, true)
}
