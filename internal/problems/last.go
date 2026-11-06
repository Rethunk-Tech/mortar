package problems

import (
	"crypto/sha256"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// LastResult is the last completed check of a profile, read back without checking anything.
type LastResult struct {
	Result Result `json:"result"`
	// At is when the check finished, in Unix milliseconds.
	At int64 `json:"at"`
}

// lastName is the cache file holding a profile's last completed check; "" for ids that would leave the cache folder.
func lastName(gameID, id string) string {
	if gameID == "" || id == "" || strings.ContainsAny(gameID+id, `/\`) {
		return ""
	}
	return meta.ProblemsLastPrefix + gameID + "-" + id + ".json"
}

func (s *Service) lastCache() *diskcache.Cache {
	return (&meta.Client{CacheDir: filepath.Join(s.home, "cache")}).Cache()
}

// saveLast stores a completed check for Last. An answer equal to the one stored last is not written again.
func (s *Service) saveLast(gameID, id string, r Result) {
	name := lastName(gameID, id)
	if name == "" {
		return
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return
	}
	sum := sha256.Sum256(raw)
	s.mu.Lock()
	if s.lastSaved == nil {
		s.lastSaved = map[string][sha256.Size]byte{}
	}
	same := s.lastSaved[name] == sum
	s.lastSaved[name] = sum
	s.mu.Unlock()
	if same {
		return
	}
	_ = diskcache.Put(s.lastCache(), name, r)
}

// Last returns the profile's last completed check from the data folder, however old and without checking anything:
// nil when none was stored. The window draws it while the live check runs, so a restart does not start from a blank
// Problems tab.
func (s *Service) Last(gameID, id string) *LastResult {
	name := lastName(gameID, id)
	if name == "" {
		return nil
	}
	e, ok := diskcache.Peek[Result](s.lastCache(), name)
	if !ok {
		return nil
	}
	return &LastResult{Result: e.Value, At: e.Fetched.UnixMilli()}
}
