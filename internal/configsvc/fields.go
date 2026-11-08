package configsvc

import (
	"context"
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// typedFile is a file Fields and SetField reach: BepInEx's .cfg and a mod config folder's JSON (Vintage Story).
func typedFile(f ConfigFile) bool { return f.Format == FormatBepInEx || f.Format == FormatJSON }

// Fields lists the typed settings of the mod's BepInEx and mod config folder files, each named
// "file/section/key" (the section is empty for a top-level key), for the command line. A mod with no such file
// yields none.
func (s *Service) Fields(game, profileID, modID string) ([]profile.ConfigField, error) {
	files, err := s.Files(game, profileID, modID)
	if err != nil {
		return nil, err
	}
	var out []profile.ConfigField
	for _, f := range files {
		if !typedFile(f) {
			continue
		}
		sc, err := s.Schema(game, profileID, modID, f.Name)
		if err != nil {
			return nil, err
		}
		for _, sec := range sc.Sections {
			for _, e := range sec.Entries {
				out = append(out, profile.ConfigField{
					Path:          f.Name + "/" + sec.Name + "/" + e.Key,
					Value:         e.Value,
					AllowValues:   e.Values,
					AllowMultiple: e.Flags,
					Default:       e.Default,
					Description:   e.Description,
					Section:       sec.Name,
				})
			}
		}
	}
	return out, nil
}

// SetField sets the setting a Fields path names. ok is false when the path names no typed file of the mod, so the
// caller can try another kind of config.
func (s *Service) SetField(ctx context.Context, game, profileID, modID, fieldPath, value string) (ok bool, err error) {
	file, rest, found := strings.Cut(fieldPath, "/")
	path, key, nested := strings.CutLast(rest, "/")
	if !found || !nested {
		return false, nil
	}
	files, err := s.Files(game, profileID, modID)
	if err != nil {
		return false, err
	}
	for _, f := range files {
		if f.Name == file && typedFile(f) {
			if err := s.Set(ctx, game, profileID, modID, file, path, key, value); err != nil {
				return true, fmt.Errorf("%s: %w", fieldPath, err)
			}
			return true, nil
		}
	}
	return false, nil
}
