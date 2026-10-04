package store

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// RootFile is the sidecar beside extracted files that names the chosen mod folder.
const RootFile = "mortar-root"

func contentRoot(dir string) string {
	b, err := fsx.ReadFile(filepath.Join(dir, RootFile))
	if err != nil {
		return ""
	}
	return CleanRoot(string(b))
}

// CleanRoot turns a mod root relative to its archive into a clean slash path, or "" when it escapes the archive.
func CleanRoot(rel string) string {
	rel = path.Clean("/" + strings.TrimSpace(strings.ReplaceAll(rel, `\`, "/")))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return ""
	}
	return rel
}

func resolveRoot(dir, rel string) (string, bool) {
	rel = CleanRoot(rel)
	if rel == "" {
		return "", false
	}
	sub := filepath.Join(dir, filepath.FromSlash(rel))
	if !datadir.UnderRoot(dir, sub) {
		return "", false
	}
	if _, err := os.Stat(sub); err != nil {
		return "", false
	}
	return sub, true
}

// SetRoot records the relative folder inside the item to copy into mods/. An empty
// rel clears a previous choice.
func (s *Store) SetRoot(game, key, rel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := s.folder(game, key)
	if err != nil {
		return err
	}
	side := filepath.Join(dir, RootFile)
	rel = CleanRoot(rel)
	if rel == "" {
		err := os.Remove(side)
		if err != nil && !os.IsNotExist(err) {
			return &Error{Game: game, Key: key, Err: err}
		}
		return nil
	}
	if _, ok := resolveRoot(dir, rel); !ok {
		return &Error{Game: game, Key: key, Err: fmt.Errorf("root %q is not in this item", rel)}
	}
	if err := fsx.WriteFile(side, []byte(rel+"\n"), 0o600); err != nil {
		return &Error{Game: game, Key: key, Err: err}
	}
	return nil
}

// Root is the stored relative content folder, or empty when none is set or it no longer exists.
func (s *Store) Root(game, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := s.folder(game, key)
	if err != nil {
		return "", err
	}
	rel := contentRoot(dir)
	if _, ok := resolveRoot(dir, rel); !ok {
		return "", nil
	}
	return rel, nil
}
