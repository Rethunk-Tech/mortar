// Package packsvc imports other managers' mod lists (r2modman codes and profiles, Thunderstore modpacks) into a
// Mortar profile, and exports a profile's Thunderstore packages as an r2modman code once the player confirms.
package packsvc

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/pack"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
)

// profiles is the part of profile.Store the service uses.
type profiles interface {
	Create(game, name string) (profile.Profile, error)
	List(game string) ([]profile.Profile, error)
	WriteFiles(game, id string, files map[string][]byte) error
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
}

// Source is what the player handed over: a file or folder path, or pasted text (a code or a bare key).
type Source struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

// Package is one package of a pack.
type Package struct {
	Source   string `json:"source"`
	Native   string `json:"native"`
	Version  string `json:"version"`
	Disabled bool   `json:"disabled"`
}

// Preview is what a pack holds, before anything is imported.
type Preview struct {
	Name     string    `json:"name"`
	Game     string    `json:"game"`
	Packages []Package `json:"packages"`
	// Configs counts the pack's config files.
	Configs int `json:"configs"`
}

// Result is what Import queued. Unsupported names packages from a source Mortar cannot install a pack's package
// from; Disabled names packages the pack switched off, which install in the profile switched off too.
type Result struct {
	Game        string   `json:"game"`
	Profile     string   `json:"profile"`
	Queued      int      `json:"queued"`
	Unsupported []string `json:"unsupported"`
	Disabled    []string `json:"disabled"`
}

func (s *Service) read(ctx context.Context, src Source) (pack.Draft, error) {
	return pack.Read(ctx, pack.Input{Path: src.Path, Text: src.Text},
		s.Code, pack.Profile{GameByFolder: game.ByR2modmanFolder}, pack.Gale{GameBySlug: gameByThunderstoreKey}, pack.Modpack{})
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
}

// LocalProfiles lists the game's profiles in the data folders of r2modman and Gale found on this computer: the
// folders below <data>/<the game's r2modman folder or Thunderstore community key>/profiles that hold a mods.yml or a
// profile.json. Pass a profile's Path as Source.Path.
func (s *Service) LocalProfiles(gameID string) ([]LocalProfile, error) {
	info, ok := components.BundledGame(gameID)
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
				out = append(out, LocalProfile{Name: e.Name(), Path: dir})
			}
		}
	}
	for _, data := range pack.DataDirs() {
		scan(data, info.R2modmanFolder, pack.IsProfileFolder)
	}
	key := ""
	if src, ok := info.Source("thunderstore"); ok {
		key = src.Key
	}
	for _, data := range pack.GaleDataDirs() {
		scan(data, key, pack.IsGaleProfileFolder)
	}
	return out, nil
}

// Preview reads the pack without importing it.
func (s *Service) Preview(ctx context.Context, src Source) (Preview, error) {
	d, err := s.read(ctx, src)
	if err != nil {
		return Preview{}, err
	}
	p := Preview{Name: d.Name, Game: d.Game, Packages: make([]Package, len(d.Packages)), Configs: len(d.Configs)}
	for i, r := range d.Packages {
		p.Packages[i] = Package{Source: r.Source, Native: r.Native, Version: r.Version, Disabled: r.Disabled}
	}
	return p, nil
}

// Import queues the pack's Thunderstore packages, with their dependencies, into profileID of gameID, or into a new
// profile named after the pack when profileID is empty. gameID may be empty when the pack names its game.
func (s *Service) Import(ctx context.Context, src Source, gameID, profileID string) (Result, error) {
	d, err := s.read(ctx, src)
	if err != nil {
		return Result{}, err
	}
	gameID = strings.TrimSpace(gameID)
	if gameID == "" {
		gameID = d.Game
	}
	if !slices.ContainsFunc(game.Catalog(), func(g components.GameInfo) bool { return g.ID == gameID }) {
		return Result{}, errors.New("choose which game this pack is for")
	}
	res := Result{Game: gameID, Unsupported: []string{}, Disabled: []string{}}
	var reqs []queue.Request
	for _, r := range d.Packages {
		if r.Source != "thunderstore" {
			res.Unsupported = append(res.Unsupported, r.Native)
			continue
		}
		req := queue.Request{Kind: queue.KindInstall, Game: gameID, Package: r.Native, Version: r.Version}
		if r.Disabled {
			res.Disabled = append(res.Disabled, r.Native)
			req.Disabled = []mod.ID{mod.NewID(mod.FormatThunderstore, r.Native)}
		}
		reqs = append(reqs, req)
	}
	if len(reqs) == 0 {
		return res, errors.New("the pack holds no Thunderstore packages to install")
	}
	if profileID == "" {
		p, err := s.Profiles.Create(gameID, cmpName(d.Name))
		if err != nil {
			return res, err
		}
		profileID = p.ID
	}
	res.Profile = profileID
	if err := s.Profiles.WriteFiles(gameID, profileID, packFiles(d)); err != nil {
		return res, err
	}
	for i := range reqs {
		reqs[i].Profile = profileID
	}
	items, err := s.Queue.Add(ctx, reqs)
	res.Queued = len(items)
	return res, err
}

// packFiles is the pack's config files and loose files as the profile holds them: below BepInEx/, where a pack's paths
// start (config/, plugins/, ...). A file the pack lists twice keeps its config.
func packFiles(d pack.Draft) map[string][]byte {
	out := map[string][]byte{}
	for _, f := range slices.Concat(d.Loose, d.Configs) {
		out["BepInEx/"+f.Path] = f.Data
	}
	return out
}

func cmpName(n string) string { return cmp.Or(strings.TrimSpace(n), "Imported pack") }

// ExportCode publishes the profile's Thunderstore packages to thunderstore.io as an r2modman code and returns the
// code's key. It sends the mod list to a public service, so it refuses unless the player confirmed.
func (s *Service) ExportCode(ctx context.Context, gameID, profileID string, confirmed bool) (string, error) {
	if !confirmed {
		return "", errors.New("exporting a code publishes this profile's mod list to Thunderstore: confirm first")
	}
	all, err := s.Profiles.List(gameID)
	if err != nil {
		return "", err
	}
	for _, p := range all {
		if p.ID != profileID {
			continue
		}
		d := pack.Draft{Name: p.Name}
		for _, e := range p.Entries {
			if e.Source.Kind == "thunderstore" {
				d.Packages = append(d.Packages, pack.Ref{Source: "thunderstore", Native: e.Source.Name, Version: e.Source.Version, Disabled: allOff(e)})
			}
		}
		if len(d.Packages) == 0 {
			return "", errors.New("the profile holds no Thunderstore packages")
		}
		return s.Code.ExportCode(ctx, d)
	}
	return "", fmt.Errorf("no profile %q", profileID)
}

// allOff reports whether the entry has mods and every one is switched off.
func allOff(e profile.Entry) bool {
	return len(e.Mods) > 0 && !slices.ContainsFunc(e.Mods, func(m profile.Component) bool {
		return !slices.ContainsFunc(e.Disabled, func(d mod.ID) bool { return mod.Equal(d, m.ID) })
	})
}
