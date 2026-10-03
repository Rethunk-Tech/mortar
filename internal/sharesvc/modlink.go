package sharesvc

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/share"
)

// errModLinkNeedsPremium explains why a pasted mod page link cannot install on a free account: Nexus only hands
// a free account a file through its own Mod Manager Download button.
var errModLinkNeedsPremium = errors.New("installing from a mod page link needs Nexus Premium; on a free account, press Mod Manager Download on the mod's Files tab")

// parseModPageURL reads a Nexus mod page link, in the old (/<domain>/mods/<id>) or new (/games/<domain>/mods/<id>)
// form; tabs, queries and fragments are ignored.
func parseModPageURL(text string) (domain string, modID int, ok bool) {
	u, err := url.Parse(strings.TrimSpace(text))
	if err != nil || u.Scheme != "https" {
		return "", 0, false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if host != "nexusmods.com" && host != "next.nexusmods.com" {
		return "", 0, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 1 && parts[0] == "games" {
		parts = parts[1:]
	}
	if len(parts) != 3 || parts[0] == "" || parts[1] != "mods" {
		return "", 0, false
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil || id < 1 || strconv.Itoa(id) != parts[2] {
		return "", 0, false
	}
	return parts[0], id, true
}

// previewModPage previews the mod's primary file (else its newest MAIN file) as a one-mod import.
func (s *Service) previewModPage(ctx context.Context, game, domain string, modID int, profileID string) (Preview, error) {
	if err := checkDomain(game, domain, "mod"); err != nil {
		return Preview{}, err
	}
	if !s.d.Premium() {
		return Preview{}, errModLinkNeedsPremium
	}
	files, err := s.d.Files(ctx, modID)
	if err != nil {
		return Preview{}, err
	}
	f := pickModPageFile(files)
	if f == nil {
		return Preview{}, errors.New("that mod has no main file to install")
	}
	ref := share.Ref{ModID: modID, FileID: f.FileID}
	return s.preview(ctx, game, share.Shared{Name: f.Name, Entries: []share.Ref{ref}}, "", nil, profileID, profile.OriginLink)
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
