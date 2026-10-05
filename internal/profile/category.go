package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/ids"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
)

const (
	maxCategoryName = 60
	categoriesDir   = "categories"
)

// CustomCategory is one user-defined mod category for a game.
type CustomCategory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type categoriesFile struct {
	Categories []CustomCategory `json:"categories"`
}

func categoriesPath(root, gameID string) (string, error) {
	if !game.Valid(gameID) {
		return "", fmt.Errorf("unknown game %q", gameID)
	}
	return filepath.Join(root, categoriesDir, gameID+".json"), nil
}

func readCategories(root, game string) ([]CustomCategory, error) {
	path, err := categoriesPath(root, game)
	if err != nil {
		return nil, err
	}
	var file categoriesFile
	if _, err := datadir.ReadJSON(path, &file); err != nil {
		return nil, fmt.Errorf("read categories: %w", err)
	}
	out := make([]CustomCategory, 0, len(file.Categories))
	for _, c := range file.Categories {
		if cleaned, ok := sanitizeCategory(c); ok {
			out = append(out, cleaned)
		}
	}
	return out, nil
}

func writeCategories(root, game string, cats []CustomCategory) error {
	path, err := categoriesPath(root, game)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(path, categoriesFile{Categories: cats})
}

func sanitizeCategory(c CustomCategory) (CustomCategory, bool) {
	c.ID = strings.TrimSpace(c.ID)
	c.Name = strings.TrimSpace(c.Name)
	c.Color = strings.TrimSpace(c.Color)
	if c.Name == "" {
		return CustomCategory{}, false
	}
	if utf8.RuneCountInString(c.Name) > maxCategoryName {
		return CustomCategory{}, false
	}
	if c.ID != "" && !idPattern.MatchString(c.ID) {
		return CustomCategory{}, false
	}
	if !validColor(c.Color) {
		c.Color = ""
	}
	return c, true
}

func cleanCategoryName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("category name is empty")
	}
	if n := utf8.RuneCountInString(name); n > maxCategoryName {
		return fmt.Errorf("category name is longer than %d characters", maxCategoryName)
	}
	return nil
}

// ListCustomCategories returns the game's custom mod categories.
func (s *Store) ListCustomCategories(game string) ([]CustomCategory, error) {
	return readCategories(s.dataDir, game)
}

// SaveCustomCategories replaces the game's custom categories. Removed ids clear that override on every profile entry.
func (s *Store) SaveCustomCategories(game string, next []CustomCategory) ([]CustomCategory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev, err := readCategories(s.dataDir, game)
	if err != nil {
		return nil, err
	}
	cleaned, err := cleanCategoryList(next)
	if err != nil {
		return nil, err
	}
	removed := removedCategoryIDs(prev, cleaned)
	if err := writeCategories(s.dataDir, game, cleaned); err != nil {
		return nil, err
	}
	if len(removed) > 0 {
		if err := s.clearCategoryOverridesLocked(game, removed); err != nil {
			return nil, err
		}
	}
	return cleaned, nil
}

func cleanCategoryList(next []CustomCategory) ([]CustomCategory, error) {
	out := make([]CustomCategory, 0, len(next))
	seenName := map[string]bool{}
	seenID := map[string]bool{}
	for _, raw := range next {
		c, ok := sanitizeCategory(raw)
		if !ok {
			continue
		}
		nameKey := strings.ToLower(c.Name)
		if seenName[nameKey] {
			return nil, fmt.Errorf("duplicate category name %q", c.Name)
		}
		seenName[nameKey] = true
		if c.ID == "" {
			c.ID = ids.New()
		}
		if seenID[c.ID] {
			return nil, fmt.Errorf("duplicate category id %q", c.ID)
		}
		seenID[c.ID] = true
		out = append(out, c)
	}
	return out, nil
}

func removedCategoryIDs(prev, next []CustomCategory) []string {
	keep := map[string]bool{}
	for _, c := range next {
		keep[c.ID] = true
	}
	var out []string
	for _, c := range prev {
		if !keep[c.ID] {
			out = append(out, c.ID)
		}
	}
	return out
}

func (s *Store) clearCategoryOverridesLocked(game string, customIDs []string) error {
	drop := map[string]bool{}
	for _, id := range customIDs {
		drop[id] = true
	}
	all, err := s.listOK(game)
	if err != nil {
		return err
	}
	for _, p := range all {
		needs := false
		for _, e := range p.Entries {
			if drop[e.CategoryOverride] {
				needs = true
				break
			}
		}
		if !needs {
			continue
		}
		if s.historyQuietIDs == nil {
			s.historyQuietIDs = map[string]int{}
		}
		s.historyQuietIDs[p.ID]++
		_, err := s.updateLocked(game, p.ID, func(prof *Profile, _ string) error {
			for i := range prof.Entries {
				if drop[prof.Entries[i].CategoryOverride] {
					prof.Entries[i].CategoryOverride = ""
				}
			}
			return nil
		})
		s.historyQuietIDs[p.ID]--
		if s.historyQuietIDs[p.ID] <= 0 {
			delete(s.historyQuietIDs, p.ID)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// SetEntryCategory sets or clears the entry's primary category override (custom category id or Nexus category name).
func (s *Store) SetEntryCategory(game, id, key, override string) (Profile, error) {
	override = strings.TrimSpace(override)
	if override != "" {
		if idPattern.MatchString(override) {
			cats, err := readCategories(s.dataDir, game)
			if err != nil {
				return Profile{}, err
			}
			if !slices.ContainsFunc(cats, func(c CustomCategory) bool { return c.ID == override }) {
				return Profile{}, fmt.Errorf("unknown custom category %q", override)
			}
		} else if err := cleanCategoryName(override); err != nil {
			return Profile{}, err
		}
	}
	return s.patchEntry(game, id, key, func(e *Entry) error {
		e.CategoryOverride = override
		return nil
	})
}
