package sharesvc

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/pack"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// readPack reads in with the pack formats: an r2modman code or key, a .r2z, a profile folder of r2modman or Gale, a
// Thunderstore modpack zip.
func (s *Service) readPack(ctx context.Context, in pack.Input) (pack.Draft, error) {
	folder := ""
	if s.d.PackFolder != nil {
		folder = s.d.PackFolder()
	}
	return pack.Read(ctx, in, s.d.PackCode, pack.GaleCode{HTTP: s.d.PackCode.HTTP}, pack.Profile{GameByFolder: gamepkg.ByR2modmanFolder},
		pack.Gale{GameBySlug: gameByThunderstoreKey, DB: pack.GaleDB(folder)}, pack.Modpack{})
}

// gameByThunderstoreKey is the catalog game whose Thunderstore community key is slug, which is what Gale names a game.
func gameByThunderstoreKey(slug string) (string, bool) {
	for _, g := range gamepkg.Catalog() {
		if src, ok := g.Source("thunderstore"); ok && src.Key == slug {
			return g.ID, true
		}
	}
	return "", false
}

// PreviewLocalProfile reads a profile folder of r2modman or Gale on this computer (a LocalProfile's Path) and
// resolves what it names against the profile it would join (profileID may be empty).
func (s *Service) PreviewLocalProfile(ctx context.Context, game, path, profileID string) (Preview, error) {
	d, err := s.readPack(ctx, pack.Input{Path: path})
	if err != nil {
		return Preview{}, packUnreadable(err)
	}
	return s.previewDraft(ctx, game, d, profileID, filepath.Clean(path))
}

// packUnreadable tags an input no pack format recognised, so the dialog says it is not a mod pack.
func packUnreadable(err error) error {
	if errors.Is(err, pack.ErrUnknown) {
		return usererr.Wrap(usererr.Damaged, err)
	}
	return err
}

// previewDraft resolves a pack as a share: its Thunderstore packages are refs, its config files and loose files
// are written into the profile on import, and what Mortar cannot carry is listed rather than failing the preview.
// importedFrom is the profile folder a local profile was read from.
func (s *Service) previewDraft(ctx context.Context, game string, d pack.Draft, profileID, importedFrom string) (Preview, error) {
	game = strings.TrimSpace(game)
	if game == "" {
		game = d.Game
	}
	if _, err := gamepkg.Require(game); err != nil {
		return Preview{}, usererr.New(usererr.Invalid, "choose which game this pack is for")
	}
	var refs []share.Ref
	var rows []Mod
	for i, r := range d.Packages {
		switch {
		case bridge.IsOwnListing(r.Native):
		case r.Source != "thunderstore":
			rows = append(rows, Mod{
				Key: packKey(i), Site: SiteLocal, Name: r.Native, Version: r.Version, State: StateUnavailable, Reason: ReasonNoSource,
				Enabled: !r.Disabled, IDs: []mod.ID{},
			})
		default:
			ref := share.Ref{Package: r.Native, Version: r.Version}
			if r.Disabled {
				ref.Disabled = []mod.ID{mod.NewID(mod.FormatThunderstore, r.Native)}
			}
			refs = append(refs, ref)
		}
	}
	files, skipped := pack.ProfileFiles(d, pack.ImportRoots(game))
	out, err := s.preview(ctx, game, share.Shared{Name: cmpName(d.Name), Entries: refs}, "", nil, profileID, "")
	if err != nil {
		return Preview{}, err
	}
	out.Mods = append(rows, out.Mods...)
	out.Game = game
	out.Settings += len(files)
	out.SkippedPrograms = skipped
	s.mu.Lock()
	if s.current != nil && s.current.id == out.Session {
		s.current.preview.Mods = out.Mods
		s.current.preview.Game = game
		s.current.preview.Settings = out.Settings
		s.current.preview.SkippedPrograms = skipped
		s.current.packFiles = files
		s.current.importedFrom = importedFrom
	}
	s.mu.Unlock()
	return out, nil
}

func packKey(index int) string { return "pack:" + strconv.Itoa(index) }

func cmpName(n string) string {
	if n = strings.TrimSpace(n); n != "" {
		return n
	}
	return "Imported pack"
}

// writePackFiles writes the pack's config and loose files into the profile. Unless overwrite, a file the profile
// already has stays as the player has it.
func (s *Service) writePackFiles(game, profileID string, files map[string][]byte, overwrite bool) error {
	if len(files) == 0 {
		return nil
	}
	dir, err := s.d.Profiles.ProfileDir(game, profileID)
	if err != nil {
		return err
	}
	out := map[string][]byte{}
	for rel, data := range files {
		if !overwrite && exists(filepath.Join(dir, filepath.FromSlash(rel))) {
			continue
		}
		out[rel] = data
	}
	if len(out) == 0 {
		return nil
	}
	return s.d.Profiles.WriteFiles(game, profileID, out)
}
