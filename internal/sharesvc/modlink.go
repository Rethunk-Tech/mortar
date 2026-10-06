package sharesvc

import (
	"context"
	"errors"

	"github.com/Rethunk-Tech/mortar/internal/game"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

// errModLinkNeedsPremium explains why a pasted mod page link cannot install on a free account: Nexus only hands
// a free account a file through its own Mod Manager Download button.
var errModLinkNeedsPremium = errors.New("installing from a mod page link needs Nexus Premium; on a free account, press Mod Manager Download on the mod's Files tab")

// previewModPage previews the mod's primary file (else its newest MAIN file) as a one-mod import.
func (s *Service) previewModPage(ctx context.Context, gameID, domain string, modID int, profileID string) (Preview, error) {
	if err := checkDomain(gameID, domain, "mod"); err != nil {
		return Preview{}, err
	}
	if !s.d.Premium() {
		return Preview{}, errModLinkNeedsPremium
	}
	t, err := game.NexusTitle(gameID)
	if err != nil {
		return Preview{}, err
	}
	files, err := s.d.Files(ctx, t, modID)
	if err != nil {
		return Preview{}, err
	}
	f := pickModPageFile(files)
	if f == nil {
		return Preview{}, errors.New("that mod has no main file to install")
	}
	ref := share.Ref{ModID: modID, FileID: f.FileID}
	return s.preview(ctx, gameID, share.Shared{Name: f.Name, Entries: []share.Ref{ref}}, "", nil, profileID, profile.OriginLink)
}

func pickModPageFile(files []nexus.File) *nexus.File {
	if f := substitute(files, ""); f != nil {
		return f
	}
	var best *nexus.File
	for i := range files {
		if files[i].Category == categoryMain && (best == nil || files[i].Uploaded.After(best.Uploaded)) {
			best = &files[i]
		}
	}
	return best
}
