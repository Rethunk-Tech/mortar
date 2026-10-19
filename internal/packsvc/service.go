// Package packsvc imports other managers' mod lists (r2modman codes and profiles, Thunderstore modpacks) into a
// Mortar profile, and exports a profile's Thunderstore packages as an r2modman code or .r2z file.
package packsvc

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/pack"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
	"github.com/Rethunk-Tech/mortar/internal/winname"
)

// profiles is the part of profile.Store the service uses.
type profiles interface {
	Create(game, name string) (profile.Profile, error)
	List(game string) ([]profile.Profile, error)
	WriteFiles(game, id string, files map[string][]byte) error
	ProfileDir(game, id string) (string, error)
	SetModpackDraft(game, id string, d profile.ModpackDraft) (profile.Profile, error)
	Backup(game, id string) (profile.Profile, map[string][]byte, error)
	BackupDirs(game string, p profile.Profile, keep func(profile.Entry) bool) (map[string]string, error)
	RestoreZip(ctx context.Context, game, zipPath string) (profile.Profile, error)
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

// Source is what the player handed over: a file or folder path, or pasted text (a code or a bare key).
type Source struct {
	Path string `json:"path"`
	Text string `json:"text"`
	// Replace lets an import into an existing profile overwrite the config files the player already has there.
	Replace bool `json:"replace"`
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
	// Kept names the pack's files an existing profile already holds with other contents, left as the player has them.
	Kept []string `json:"kept"`
}

func (s *Service) read(ctx context.Context, src Source) (pack.Draft, error) {
	return pack.Read(ctx, pack.Input{Path: src.Path, Text: src.Text},
		s.Code, pack.Profile{GameByFolder: game.ByR2modmanFolder}, pack.Gale{GameBySlug: gameByThunderstoreKey, DB: pack.GaleDB(s.folder())}, pack.Modpack{})
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
}

// LocalProfiles lists the game's profiles in the data folders of r2modman (the folders below
// <data>/<the game's r2modman folder>/profiles that hold a mods.yml) and in Gale's database, found on this computer or
// in the folder the player chose.
// Pass a profile's Path as Source.Path.
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
	return out, nil
}

func (s *Service) local(name, path, from string) LocalProfile {
	d, _ := s.read(context.Background(), Source{Path: path})
	return LocalProfile{Name: name, Path: path, Source: from, Mods: len(d.Packages)}
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
	return s.importPack(ctx, src, gameID, profileID, "")
}

// ImportAs is Import into a new profile called name (the pack's own name when name is blank).
func (s *Service) ImportAs(ctx context.Context, src Source, gameID, name string) (Result, error) {
	return s.importPack(ctx, src, gameID, "", name)
}

func (s *Service) importPack(ctx context.Context, src Source, gameID, profileID, name string) (Result, error) {
	d, err := s.read(ctx, src)
	if err != nil {
		return Result{}, err
	}
	gameID = strings.TrimSpace(gameID)
	gameID = cmp.Or(gameID, d.Game)
	if !slices.ContainsFunc(game.Catalog(), func(g components.GameInfo) bool { return g.ID == gameID }) {
		return Result{}, errors.New("choose which game this pack is for")
	}
	res := Result{Game: gameID, Unsupported: []string{}, Disabled: []string{}, Kept: []string{}}
	var reqs []queue.Request
	for _, r := range d.Packages {
		if bridge.IsOwnListing(r.Native) {
			continue
		}
		if r.Source != "thunderstore" {
			res.Unsupported = append(res.Unsupported, r.Native)
			continue
		}
		req := queue.Request{Kind: queue.KindInstall, Game: gameID, Package: r.Native, Version: r.Version, InferReason: true}
		if r.Disabled {
			res.Disabled = append(res.Disabled, r.Native)
			req.Disabled = []mod.ID{mod.NewID(mod.FormatThunderstore, r.Native)}
		}
		reqs = append(reqs, req)
	}
	if len(reqs) == 0 {
		return res, errors.New("the pack holds no Thunderstore packages to install")
	}
	files := packFiles(d, importRoots(gameID))
	if profileID == "" {
		p, err := s.Profiles.Create(gameID, cmpName(cmp.Or(strings.TrimSpace(name), d.Name)))
		if err != nil {
			return res, err
		}
		profileID = p.ID
	} else if !src.Replace {
		dir, err := s.Profiles.ProfileDir(gameID, profileID)
		if err != nil {
			return res, err
		}
		res.Kept = dropDiffering(dir, files)
	}
	res.Profile = profileID
	if err := s.Profiles.WriteFiles(gameID, profileID, files); err != nil {
		return res, err
	}
	for i := range reqs {
		reqs[i].Profile = profileID
	}
	items, err := s.Queue.Add(ctx, reqs)
	res.Queued = len(items)
	return res, err
}

// dropDiffering removes from files, and returns sorted, each file the profile at dir already holds with other contents.
// A path that leaves dir stays in files for WriteFiles to refuse.
func dropDiffering(dir string, files map[string][]byte) []string {
	kept := []string{}
	for rel, data := range files {
		if !filepath.IsLocal(filepath.FromSlash(rel)) {
			continue
		}
		have, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err == nil && !bytes.Equal(have, data) {
			kept = append(kept, rel)
			delete(files, rel)
		}
	}
	slices.Sort(kept)
	return kept
}

// Extra is an installed Thunderstore package a pack does not list.
type Extra struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Extras lists the packages profileID holds that the pack does not: what Import into that profile would leave
// beside the pack's own, for the player to remove when the profile should match the pack. Packages Mortar brought
// in as dependencies are left out, since a pack that names only its top-level mods still needs them.
func (s *Service) Extras(ctx context.Context, src Source, gameID, profileID string) ([]Extra, error) {
	d, err := s.read(ctx, src)
	if err != nil {
		return nil, err
	}
	listed := map[string]bool{}
	for _, r := range d.Packages {
		listed[strings.ToLower(r.Native)] = true
	}
	all, err := s.Profiles.List(gameID)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(all, func(p profile.Profile) bool { return p.ID == profileID })
	if i < 0 {
		return nil, errors.New("that profile does not exist")
	}
	out := []Extra{}
	for _, e := range all[i].Entries {
		if !e.Package || e.Implicit || e.Source.Kind != profile.KindThunderstore || listed[strings.ToLower(e.Source.Name)] || bridge.IsOwnListing(e.Source.Name) {
			continue
		}
		out = append(out, Extra{Key: e.Key, Name: e.Source.Name, Version: e.Source.Version})
	}
	return out, nil
}

// blockedExtensions are the executable types r2modman refuses to extract from an imported profile
// (ProfileUtils.ts:44-50).
var blockedExtensions = []string{
	".dll", ".exe", ".scr", ".com", ".pif", ".bat", ".cmd", ".ps1", ".vbs", ".vbe", ".js", ".jse", ".wsf", ".wsh",
	".hta", ".msi", ".msix", ".sys", ".drv", ".cpl", ".ocx", ".lnk", ".reg", ".inf",
}

// packFiles is the pack's config files and loose files as the profile holds them, by r2modman's import rule
// (ProfileUtils.ts:56-67): a config/ entry goes below BepInEx/, any other path is relative to the profile, and
// executable types are skipped. Mortar's profile folder also holds its own records (profile.json, history, saves), so
// only a file below one of roots, the game's loaders' import folders, is kept, written under the root's own spelling. A
// file the pack lists twice keeps its config.
func packFiles(d pack.Draft, roots []string) map[string][]byte {
	out := map[string][]byte{}
	keep := func(p string, data []byte) {
		p, ok := winPath(p)
		if !ok {
			return
		}
		first, rest, ok := strings.Cut(p, "/")
		if !ok {
			return
		}
		i := slices.IndexFunc(roots, func(r string) bool { return strings.EqualFold(r, first) })
		if i < 0 || slices.ContainsFunc(blockedExtensions, func(e string) bool { return strings.HasSuffix(strings.ToLower(p), e) }) {
			return
		}
		out[roots[i]+"/"+rest] = data
	}
	for _, f := range d.Loose {
		keep(f.Path, f.Data)
	}
	for _, f := range d.Configs {
		keep("BepInEx/"+f.Path, f.Data)
	}
	return out
}

// winPath is p with each segment's trailing dots and spaces removed, as Windows stores it, so "evil.dll." is judged as
// the evil.dll it becomes there. A segment with nothing else, or a name Windows cannot hold, refuses the path.
func winPath(p string) (string, bool) {
	segs := strings.Split(p, "/")
	for i, seg := range segs {
		segs[i] = strings.TrimRight(seg, ". ")
		if !winname.Valid(segs[i]) {
			return "", false
		}
	}
	return strings.Join(segs, "/"), true
}

// importRoots are the folders the game's loaders let an imported pack write loose files into.
func importRoots(gameID string) []string {
	info, ok := components.Game(gameID)
	if !ok {
		return nil
	}
	var out []string
	for _, l := range loader.For(info) {
		if r, ok := l.(loader.ImportRoots); ok {
			out = append(out, r.ImportRoots()...)
		}
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
