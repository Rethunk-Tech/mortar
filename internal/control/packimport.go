package control

import (
	"context"
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/packsvc"
)

// packImport reads another manager's pack (an r2modman code or key, a .r2z, a profile folder, a Thunderstore
// modpack): with Preview it only reports what the pack holds; otherwise it queues the pack's packages into Profile, or
// into a new profile named Name (the pack's own name when Name is empty). Value is pasted text, Path a file or folder.
func (s *Services) packImport(ctx context.Context, p Params) (any, error) {
	if s.Packs == nil {
		return nil, errors.New("pack import is unavailable")
	}
	src := packsvc.Source{Path: p.Path, Text: p.Value}
	if p.Preview {
		return s.Packs.Preview(ctx, src)
	}
	profileID := ""
	if p.Profile != "" {
		prof, err := s.resolve(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		profileID = prof.ID
	}
	res, err := s.Packs.Import(ctx, src, p.Game, profileID)
	if err == nil && profileID == "" && strings.TrimSpace(p.Name) != "" {
		_, err = s.Profiles.Rename(res.Game, res.Profile, p.Name)
	}
	if res.Profile != "" && s.Emit != nil {
		s.Emit(ChangedEvent, res.Game)
	}
	return res, err
}

// packExportModpack writes the profile as a Thunderstore modpack zip at Path; All adds its config folder.
func (s *Services) packExportModpack(p Params) (any, error) {
	if s.Packs == nil {
		return nil, errors.New("pack export is unavailable")
	}
	if strings.TrimSpace(p.Path) == "" {
		return nil, errors.New("name the modpack zip to write")
	}
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	return s.Packs.ExportModpack(p.Game, prof.ID, p.Path, p.All)
}

// profileBackup writes the profile to the backup file at Path.
func (s *Services) profileBackup(p Params) (any, error) {
	if s.Packs == nil {
		return nil, errors.New("profile backup is unavailable")
	}
	if strings.TrimSpace(p.Path) == "" {
		return nil, errors.New("name the backup file to write")
	}
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	return map[string]string{"path": p.Path}, s.Packs.Backup(p.Game, prof.ID, p.Path)
}

// profileRestore makes a new profile from the backup file at Path; Game, when given, must be the backup's game.
func (s *Services) profileRestore(ctx context.Context, p Params) (any, error) {
	if s.Packs == nil {
		return nil, errors.New("profile restore is unavailable")
	}
	res, err := s.Packs.Restore(ctx, p.Path, p.Game)
	if res.Profile != "" && s.Emit != nil {
		s.Emit(ChangedEvent, res.Game)
	}
	return res, err
}
