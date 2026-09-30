package sharesvc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

const pendingFile = "pending-configs.json"

// loadPending reads the imports whose config files were still waiting when Mortar last quit. A missing or
// unreadable file means none: the config files are a convenience and the mods themselves are already queued.
func loadPending(dir string) []*pending {
	if dir == "" {
		return nil
	}
	b, err := fsx.ReadFile(filepath.Join(dir, pendingFile))
	if err != nil {
		return nil
	}
	var saved []*pending
	if json.Unmarshal(b, &saved) != nil {
		return nil
	}
	return saved
}

// savePending writes the pending imports, or removes the file when none are left, so a restart resumes them.
func (s *Service) savePending() {
	if s.d.Dir == "" {
		return
	}
	s.mu.Lock()
	list := slices.Clone(s.pending)
	s.mu.Unlock()
	path := filepath.Join(s.d.Dir, pendingFile)
	if len(list) == 0 {
		_ = os.Remove(path)
		return
	}
	// A pending list that cannot be saved still works this session; it only starts over after a restart.
	_ = datadir.WriteJSON(path, list)
}
