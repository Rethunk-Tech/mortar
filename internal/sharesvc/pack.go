package sharesvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/pack"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/Rethunk-Tech/mortar/internal/winname"
)

// packCodes are the pack formats a pasted text can be: an r2modman code or key, or a Gale code.
func (s *Service) packCodes() []pack.Format {
	return []pack.Format{s.d.PackCode, pack.GaleCode{HTTP: s.d.PackCode.HTTP}}
}

// isPackCode reports whether pasted text is a profile code one of the pack formats reads.
func (s *Service) isPackCode(in pack.Input) bool {
	return slices.ContainsFunc(s.packCodes(), func(f pack.Format) bool { return f.Detect(in) })
}

// readPack reads in with the pack formats: the profile codes, a .r2z, a profile folder of r2modman or Gale, a
// Thunderstore modpack zip.
func (s *Service) readPack(ctx context.Context, in pack.Input) (pack.Draft, error) {
	folder := ""
	if s.d.PackFolder != nil {
		folder = s.d.PackFolder()
	}
	formats := append(s.packCodes(), pack.Profile{GameByFolder: gamepkg.ByR2modmanFolder},
		pack.Gale{GameBySlug: gameByThunderstoreKey, DB: pack.GaleDB(folder)}, pack.Modpack{})
	return pack.Read(ctx, in, formats...)
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
	adoptLocalPlugins(&out, &d, importedFrom)
	rows = append(rows, carriedRows(d.Local)...)
	out.Mods = append(rows, out.Mods...)
	out.Game = game
	out.Settings += len(files)
	out.SkippedPrograms = skipped
	s.mu.Lock()
	if s.current != nil && s.current.id == out.Session {
		s.current.preview.Mods = out.Mods
		s.current.preview.Problems = out.Problems
		s.current.preview.Game = game
		s.current.preview.Settings = out.Settings
		s.current.preview.SkippedPrograms = skipped
		s.current.packFiles = files
		s.current.carried = d.Local
		s.current.importedFrom = importedFrom
	}
	s.mu.Unlock()
	return out, nil
}

// adoptLocalPlugins turns a package the index does not list into a carried local mod when the r2modman profile folder
// the pack was read from holds its folder: "Import local mod" names such a mod Author-Name like a package, which
// nothing in mods.yml tells apart. Its files install as a Local entry, as a Gale local mod's do.
func adoptLocalPlugins(out *Preview, d *pack.Draft, importedFrom string) {
	if importedFrom == "" {
		return
	}
	adopted := map[string]bool{}
	kept := out.Mods[:0]
	for _, m := range out.Mods {
		dir := filepath.Join(importedFrom, "BepInEx", "plugins", m.Package)
		if m.Reason != ReasonNotInIndex || !winname.Valid(m.Package) || !fsx.IsDir(dir) {
			kept = append(kept, m)
			continue
		}
		d.Local = append(d.Local, pack.LocalMod{Name: m.Name, Author: m.Author, Version: m.Version, Disabled: len(m.Disabled) > 0, Dir: dir})
		adopted[m.Key] = true
	}
	out.Mods = kept
	out.Problems = slices.DeleteFunc(out.Problems, func(p Problem) bool { return adopted[p.Key] })
}

func packKey(index int) string { return "pack:" + strconv.Itoa(index) }

func carriedKey(index int) string { return "carried:" + strconv.Itoa(index) }

// carriedRows are a pack's local mods as installable local rows.
func carriedRows(local []pack.LocalMod) []Mod {
	out := make([]Mod, 0, len(local))
	for i, m := range local {
		from := FilesFromCode
		if m.Dir != "" {
			from = FilesFromComputer
		}
		out = append(out, Mod{
			Key: carriedKey(i), Site: SiteLocal, Name: m.Name, Author: m.Author, Version: m.Version, State: StateDownload,
			Enabled: !m.Disabled, IDs: []mod.ID{}, SizeKB: (m.Size() + 1023) / 1024, FilesFrom: from,
		})
	}
	return out
}

// installCarried adds a pack's local mod to the profile as a local entry, switched off when it was; one the profile
// already holds is left as it is.
func (s *Service) installCarried(ctx context.Context, game, profileID string, m pack.LocalMod) (string, error) {
	// The data folder rather than the system temp folder, which may be in memory and too small for a large mod.
	scratch, err := os.MkdirTemp(s.d.Dir, "mortar-carried-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	dir := filepath.Join(scratch, m.Name)
	if err := m.Stage(dir); err != nil {
		return "", err
	}
	res, err := s.d.Profiles.InstallLocalFolder(ctx, game, profileID, dir, !m.Disabled)
	if errors.As(err, new(*profile.DuplicateError)) {
		return "", nil
	}
	return res.Profile.LastChange, err
}

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
