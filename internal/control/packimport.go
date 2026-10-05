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
