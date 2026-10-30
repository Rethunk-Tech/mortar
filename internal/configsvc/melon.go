package configsvc

import (
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-cfgedit"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// MelonLoader keeps every melon's settings in UserData/MelonPreferences.cfg, a TOML file (MelonLoader/Preferences,
// Tomlet): one [Category] table per melon or settings group, and per entry its description as the comment lines above
// `Key = value` and an optional inline `# comment`. The file names no types, ranges or defaults, so the type of an entry
// is read off its value.
const melonName = "MelonPreferences.cfg"

func (s *Service) melonPath(game, id string) (string, error) {
	dir, err := s.Profiles.ProfileDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "UserData", melonName), nil
}

// ownsMelonCategory reports whether a category is the settings of a mod known by names: the category's name and the
// mod's name or package name contain one another once case and punctuation are dropped.
func ownsMelonCategory(names []string, section string) bool {
	c := mod.LooseName(section)
	if len(c) < minConfigName {
		return false
	}
	for _, n := range names {
		_, pkg, _ := strings.Cut(n, ":")
		_, name, _ := strings.Cut(pkg, "-")
		for _, k := range []string{mod.LooseName(pkg), mod.LooseName(name)} {
			if len(k) >= minConfigName && (strings.Contains(k, c) || strings.Contains(c, k)) {
				return true
			}
		}
	}
	return false
}

// melonFile lists MelonPreferences.cfg for the mod when one of its categories is in the file, or for the whole profile
// when modID is empty.
func (s *Service) melonFile(game, profileID, modID string, names []string) []ConfigFile {
	path, err := s.melonPath(game, profileID)
	if err != nil || !safefs.IsFile(path) {
		return nil
	}
	if modID != "" {
		raw, err := safefs.ReadFile(path)
		if err != nil {
			return nil
		}
		found := false
		for _, e := range cfgedit.ParseMelon(string(raw)).Entries() {
			if ownsMelonCategory(names, e.Section) {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return []ConfigFile{{Name: melonName, Format: FormatMelon, Label: melonName}}
}

func (s *Service) melonSchema(game, profileID, modID string, f ConfigFile) (Schema, error) {
	path, err := s.melonPath(game, profileID)
	if err != nil {
		return Schema{}, err
	}
	raw, err := safefs.ReadFile(path)
	if err != nil {
		return Schema{}, err
	}
	keep := func(string) bool { return true }
	if modID != "" {
		keep = func(section string) bool { return ownsMelonCategory([]string{modID}, section) }
	}
	return schemaOf(f, cfgedit.ParseMelon(string(raw)).Entries(), keep), nil
}

func (s *Service) writeMelon(game, profileID string, edits []edit) error {
	path, err := s.melonPath(game, profileID)
	if err != nil {
		return err
	}
	raw, err := safefs.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(raw)
	for _, e := range edits {
		if text, err = cfgedit.ParseMelon(text).Set(e.section, e.key, e.value); err != nil {
			return valueErr(err)
		}
	}
	return safefs.WriteFileAtomic(path, []byte(text), 0o600)
}
