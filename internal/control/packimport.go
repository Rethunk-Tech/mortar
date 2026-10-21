package control

import (
	"cmp"
	"context"
	"errors"
	"os"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
)

// packImport reads another manager's pack (an r2modman code or key, a .r2z, a profile folder, a Thunderstore
// modpack) through the share import: with Preview it only reports what the pack holds; otherwise it queues the pack's
// packages into Profile, or into a new profile named Name (the pack's own name when Name is empty). Value is pasted
// text, Path a file or folder.
func (s *Services) packImport(ctx context.Context, p Params) (any, error) {
	if s.Shares == nil {
		return nil, errors.New("pack import is unavailable")
	}
	profileID := ""
	if p.Profile != "" {
		prof, err := s.resolve(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		profileID = prof.ID
	}
	var pv sharesvc.Preview
	var err error
	switch st, serr := os.Stat(p.Path); {
	case p.Path == "":
		pv, err = s.Shares.PreviewLink(ctx, p.Game, p.Value, profileID)
	case serr == nil && st.IsDir():
		pv, err = s.Shares.PreviewLocalProfile(ctx, p.Game, p.Path, profileID)
	default:
		pv, err = s.Shares.PreviewFile(ctx, p.Game, p.Path, profileID)
	}
	if err != nil || p.Preview {
		return pv, err
	}
	game := cmp.Or(p.Game, pv.Game)
	res, err := s.Shares.Import(ctx, game, pv.Session, profileID, nil)
	if err == nil && profileID == "" && strings.TrimSpace(p.Name) != "" {
		_, err = s.Profiles.Rename(game, res.Profile.ID, p.Name)
	}
	if res.Profile.ID != "" && s.Emit != nil {
		s.Emit(ChangedEvent, game)
	}
	return res, err
}

// packExportModpack writes the profile as a Thunderstore modpack zip at Path from the form's remembered fields (a
// patch bump of the last version); All adds its config folder.
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
	form, err := s.Packs.ModpackForm(p.Game, prof.ID)
	if err != nil {
		return nil, err
	}
	o := form.Options
	if !p.All {
		o.Configs = nil
	}
	return s.Packs.ExportModpack(p.Game, prof.ID, p.Path, o)
}

// packExportR2z writes the profile as an r2modman .r2z file at Path.
func (s *Services) packExportR2z(p Params) (any, error) {
	if s.Packs == nil {
		return nil, errors.New("pack export is unavailable")
	}
	if strings.TrimSpace(p.Path) == "" {
		return nil, errors.New("name the .r2z file to write")
	}
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	path, err := s.Packs.ExportR2z(p.Game, prof.ID, p.Path)
	return map[string]string{"path": path}, err
}

// profileBackup writes the profile to the backup file at Path; All carries the mod files, otherwise only the ones no
// source can download again.
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
	return map[string]string{"path": p.Path}, s.Packs.Backup(p.Game, prof.ID, p.Path, p.All)
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
