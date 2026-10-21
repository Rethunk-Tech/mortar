// Package packsvc imports other managers' mod lists (r2modman codes and profiles, Thunderstore modpacks) into a
// Mortar profile, and exports a profile's Thunderstore packages as an r2modman code or .r2z file.
package packsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/pack"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

// profiles is the part of profile.Store the service uses.
type profiles interface {
	List(game string) ([]profile.Profile, error)
	ProfileDir(game, id string) (string, error)
	SetModpackDraft(game, id string, d profile.ModpackDraft) (profile.Profile, error)
	Backup(game, id string) (profile.Profile, map[string][]byte, error)
	BackupDirs(game string, p profile.Profile, keep func(profile.Entry) bool) (map[string]string, error)
	RestoreBackup(ctx context.Context, game string, p profile.Profile, files map[string][]byte, dirs map[string]string) (profile.Profile, []profile.Entry, error)
}

// downloads is the part of queue.Service the service uses.
type downloads interface {
	Add(ctx context.Context, reqs []queue.Request) ([]queue.Item, error)
}

// Service reads packs and puts their packages in the download queue.
type Service struct {
	Profiles profiles
	Queue    downloads
	// Code reads r2modman codes; the zero value talks to the real Thunderstore.
	Code pack.Code
	// App asks where to save an exported modpack; nil outside the window.
	App winhost.Host
	// Folder is the player's own r2modman or Gale data folder, for managers kept outside their default places; nil
	// or empty means the defaults only.
	Folder func() string
}

func (s *Service) folder() string {
	if s.Folder == nil {
		return ""
	}
	return s.Folder()
}

// Result is what MirrorClient queued. Unsupported names packages from a source Mortar cannot install a pack's package
// from; Disabled names packages the pack switched off, which install in the profile switched off too.
type Result struct {
	Game        string   `json:"game"`
	Profile     string   `json:"profile"`
	Queued      int      `json:"queued"`
	Unsupported []string `json:"unsupported"`
	Disabled    []string `json:"disabled"`
}

func (s *Service) read(ctx context.Context, path string) (pack.Draft, error) {
	return pack.Read(ctx, pack.Input{Path: path},
		s.Code, pack.GaleCode{HTTP: s.Code.HTTP}, pack.Profile{GameByFolder: game.ByR2modmanFolder}, pack.Gale{GameBySlug: gameByThunderstoreKey, DB: pack.GaleDB(s.folder())}, pack.Modpack{})
}

// gameByThunderstoreKey is the catalog game whose Thunderstore community key is slug, which is what Gale names a game.
func gameByThunderstoreKey(slug string) (string, bool) {
	for _, g := range game.Catalog() {
		if src, ok := g.Source("thunderstore"); ok && src.Key == slug {
			return g.ID, true
		}
	}
	return "", false
}

// LocalProfile is a profile folder of r2modman, the Thunderstore Mod Manager or Gale on this computer.
type LocalProfile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	// Source is the manager that keeps the profile: "r2modman" or "Gale".
	Source string `json:"source"`
	// Mods counts its packages; 0 when the profile could not be read.
	Mods int `json:"mods"`
	// Imported is set when a profile of this game was already imported from it.
	Imported bool `json:"imported"`
}

// LocalProfiles lists the game's profiles in the data folders of r2modman (the folders below
// <data>/<the game's r2modman folder>/profiles that hold a mods.yml) and in Gale's database, found on this computer or
// in the folder the player chose.
func (s *Service) LocalProfiles(gameID string) ([]LocalProfile, error) {
	info, ok := components.Game(gameID)
	if !ok {
		return []LocalProfile{}, nil
	}
	out := []LocalProfile{}
	scan := func(data, folder string, isProfile func(string) bool) {
		if folder == "" {
			return
		}
		root := filepath.Join(data, folder, "profiles")
		entries, err := os.ReadDir(root)
		if err != nil {
			return
		}
		for _, e := range entries {
			dir := filepath.Join(root, e.Name())
			if isProfile(dir) {
				out = append(out, s.local(e.Name(), dir, "r2modman"))
			}
		}
	}
	for _, data := range pack.DataDirs(s.folder()) {
		scan(data, info.R2modmanFolder, pack.IsProfileFolder)
	}
	if src, ok := info.Source("thunderstore"); ok && src.Key != "" {
		if db := pack.GaleDB(s.folder()); db != "" {
			all, err := pack.GaleProfiles(context.Background(), db)
			if err != nil {
				return out, err
			}
			for _, p := range all {
				if p.Slug == src.Key {
					out = append(out, s.local(p.Name, p.Path, "Gale"))
				}
			}
		}
	}
	if s.Profiles == nil {
		return out, nil
	}
	if have, err := s.Profiles.List(gameID); err == nil {
		for i := range out {
			out[i].Imported = slices.ContainsFunc(have, func(p profile.Profile) bool {
				return p.ImportedFrom != "" && p.ImportedFrom == filepath.Clean(out[i].Path)
			})
		}
	}
	return out, nil
}

func (s *Service) local(name, path, from string) LocalProfile {
	d, _ := s.read(context.Background(), path)
	return LocalProfile{Name: name, Path: path, Source: from, Mods: len(d.Packages)}
}

// ExportCode publishes the profile's Thunderstore packages to thunderstore.io as an r2modman code and returns the
// code's key. It sends the mod list to a public service, so it refuses unless the player confirmed.
func (s *Service) ExportCode(ctx context.Context, gameID, profileID string, confirmed bool) (string, error) {
	if !confirmed {
		return "", errors.New("exporting a code publishes this profile's mod list to Thunderstore: confirm first")
	}
	d, err := s.r2zDraft(gameID, profileID)
	if err != nil {
		return "", err
	}
	return s.Code.ExportCode(ctx, d)
}

// ExportR2zDialog asks where to save, then writes the profile as an .r2z file. An empty path means the player cancelled.
func (s *Service) ExportR2zDialog(ctx context.Context, gameID, profileID string) (string, error) {
	if s.App == nil {
		return "", errors.New("no window to ask where to save")
	}
	p, err := s.find(gameID, profileID)
	if err != nil {
		return "", err
	}
	dest, err := s.App.SaveFile(winhost.Dialog{Filename: packageName(p.Name) + ".r2z", Filters: []winhost.Filter{{Name: "r2modman or Gale export (.r2z)", Pattern: "*.r2z"}}})
	if err != nil || dest == "" {
		return dest, err
	}
	return s.ExportR2z(gameID, profileID, dest)
}

// ExportR2z writes the profile's Thunderstore packages as an r2modman .r2z file at dest. It does not post a public code.
func (s *Service) ExportR2z(gameID, profileID, dest string) (string, error) {
	d, err := s.r2zDraft(gameID, profileID)
	if err != nil {
		return "", err
	}
	return dest, pack.WriteR2z(dest, d)
}

// r2zDraft is the profile as an r2modman export: Thunderstore packages plus the BepInEx config folder.
func (s *Service) r2zDraft(gameID, profileID string) (pack.Draft, error) {
	p, err := s.find(gameID, profileID)
	if err != nil {
		return pack.Draft{}, err
	}
	d := pack.Draft{Name: p.Name}
	for _, e := range p.Entries {
		if e.Source.Kind == profile.KindThunderstore {
			d.Packages = append(d.Packages, pack.Ref{Source: "thunderstore", Native: e.Source.Name, Version: e.Source.Version, Disabled: allOff(e)})
		}
	}
	if len(d.Packages) == 0 {
		return pack.Draft{}, errors.New("the profile holds no Thunderstore packages")
	}
	if ref, ok := s.loaderPack(gameID, p.ID); ok {
		d.Packages = append([]pack.Ref{ref}, d.Packages...)
	}
	if d.Configs, err = s.configFiles(gameID, p.ID); err != nil {
		return pack.Draft{}, err
	}
	return d, nil
}

// loaderPack is the profile's loader as the Thunderstore package r2modman lists it among a profile's mods: r2modman
// installs exactly the mods a code lists, so a code without it would give its importer a profile that never loads them.
// ok is false for a loader that lives in the game folder or is missing from the profile.
func (s *Service) loaderPack(gameID, profileID string) (pack.Ref, bool) {
	info, ok := components.Game(gameID)
	l, hasLoader := game.PrimaryLoader(gameID)
	if !ok || !hasLoader {
		return pack.Ref{}, false
	}
	if _, perProfile := l.(loader.InProfile); !perProfile {
		return pack.Ref{}, false
	}
	dir, err := s.Profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return pack.Ref{}, false
	}
	st, err := l.Status(loader.Target{Game: gameID, ProfileDir: dir})
	if err != nil || !st.Installed || st.Version == "" {
		return pack.Ref{}, false
	}
	return pack.Ref{Source: "thunderstore", Native: info.LoaderPackage(), Version: st.Version}, true
}

// allOff reports whether the entry has mods and every one is switched off.
func allOff(e profile.Entry) bool {
	return len(e.Mods) > 0 && !slices.ContainsFunc(e.Mods, func(m profile.Component) bool {
		return !slices.ContainsFunc(e.Disabled, func(d mod.ID) bool { return mod.Equal(d, m.ID) })
	})
}
