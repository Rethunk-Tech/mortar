// Package archivesvc serves the install-from-archive helpers: a listing preview and the not-yet-installed archives
// in the download folder.
package archivesvc

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/dlwatch"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/ids"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/itch"
	"github.com/Rethunk-Tech/mortar/internal/source/patreon"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// maxHashed bounds the archives hashed per listing, newest first; older ones are matched by name only.
const maxHashed = 200

// Deps wires the service to the download folder, the store, the profiles and the download history.
type Deps struct {
	// Dirs are the folders archives land in (Mortar's download folder and the user's Downloads folder); empty
	// entries are skipped and a path listed twice counts once.
	Dirs func() []string
	// Install adds a download to a profile; src carries the Nexus mod id when the file name has one.
	Install func(ctx context.Context, game, profileID, path string, src profile.Source) (profile.InstallResult, error)
	// HashCache is the file that remembers each archive's SHA-256 by path, size and mtime; empty keeps hashes in
	// memory for one listing only.
	HashCache string
	Keys      func(game string) ([]string, error)
	Profiles  func(game string) ([]profile.Profile, error)
	// NexusMods is the set of Nexus mod ids in the download history.
	NexusMods func() map[int]bool
	// Offer reports whether the game's "Offer new downloads" setting is on; Seen and SetSeen keep the newest archive
	// mtime already offered.
	Offer   func(game string) bool
	Seen    func(game string) int64
	SetSeen func(game string, mtime int64) error
}

// DownloadArchive is an archive in the download folder that no profile or store item accounts for.
type DownloadArchive struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Mtime int64  `json:"mtime"`
	// KnownNexus is true when the file name carries a Nexus mod id that the download history has seen.
	KnownNexus bool `json:"knownNexus"`
	ModID      int  `json:"modId"`
}

// Service exposes the helpers to the frontend.
type Service struct{ d Deps }

func NewService(d Deps) *Service { return &Service{d: d} }

// ArchivePreview lists the archive at path (files, SMAPI manifests, FOMOD) without extracting it. The UI calls it
// before the profile's InstallArchive.
func (s *Service) ArchivePreview(path string) (archive.Preview, error) {
	return archive.PreviewArchive(path)
}

// DownloadsArchives lists archives in the download folder that are not installed for the game, newest first. An
// archive counts as installed when its content key is in the store, or a profile holds a local entry of the same
// file name.
func (s *Service) DownloadsArchives(game string) ([]DownloadArchive, error) {
	var out []DownloadArchive
	seen := map[string]bool{}
	var nexus map[int]bool
	if s.d.NexusMods != nil {
		nexus = s.d.NexusMods()
	}
	for _, dir := range s.dirs() {
		ents, err := os.ReadDir(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range ents {
			path := filepath.Join(dir, e.Name())
			if e.IsDir() || !isArchive(e.Name()) || seen[path] {
				continue
			}
			info, err := e.Info()
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			seen[path] = true
			a := DownloadArchive{Path: path, Name: e.Name(), Size: info.Size(), Mtime: info.ModTime().UnixMilli()}
			if inf, ok := dlwatch.ParseNexusFilename(a.Name); ok {
				a.ModID, a.KnownNexus = inf.ModID, nexus[inf.ModID]
			}
			out = append(out, a)
		}
	}
	keys, err := s.storeKeys(game)
	if err != nil {
		return nil, err
	}
	names, err := s.profileNames(game)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(out, func(a, b DownloadArchive) int { return int(b.Mtime - a.Mtime) })
	cache := s.loadHashes()
	kept := make(map[string]hashEntry, len(cache))
	dirty := false
	pending := make([]DownloadArchive, 0, len(out))
	for i, a := range out {
		if names[a.Name] {
			continue
		}
		if i < maxHashed {
			h, ok := cache[a.Path]
			if !ok || h.Size != a.Size || h.Mtime != a.Mtime {
				sum, err := fsx.SHA256(a.Path)
				if err != nil {
					pending = append(pending, a)
					continue
				}
				h, dirty = hashEntry{Size: a.Size, Mtime: a.Mtime, Sum: sum}, true
			}
			kept[a.Path] = h
			if keys[store.LocalKey(h.Sum)] {
				continue
			}
		}
		pending = append(pending, a)
	}
	if dirty || len(kept) != len(cache) {
		s.saveHashes(kept)
	}
	return pending, nil
}

type hashEntry struct {
	Size  int64  `json:"size"`
	Mtime int64  `json:"mtime"`
	Sum   string `json:"sum"`
}

type hashFile struct {
	FormatVersion int                  `json:"formatVersion"`
	Archives      map[string]hashEntry `json:"archives"`
}

func (s *Service) loadHashes() map[string]hashEntry {
	f := hashFile{}
	if s.d.HashCache != "" {
		if _, err := datadir.ReadJSON(s.d.HashCache, &f); err != nil {
			f = hashFile{}
		}
	}
	if f.Archives == nil {
		f.Archives = map[string]hashEntry{}
	}
	return f.Archives
}

// saveHashes is best effort: a failed write only means the next listing hashes again.
func (s *Service) saveHashes(m map[string]hashEntry) {
	if s.d.HashCache == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.d.HashCache), 0o700)
	_ = datadir.WriteVersioned(s.d.HashCache, hashFile{FormatVersion: datadir.FormatVersion, Archives: m})
}

// NewDownloads returns the archives that arrived since the last call and moves the mark past them. Archives the
// queue wrote (its files are named by item id) are not new downloads. The first call only sets the mark, so a
// folder that already holds archives does not offer them all; a game with the offer off sets no mark.
func (s *Service) NewDownloads(game string) ([]DownloadArchive, error) {
	if s.d.Offer == nil || s.d.Seen == nil || s.d.SetSeen == nil || !s.d.Offer(game) {
		return []DownloadArchive{}, nil
	}
	all, err := s.DownloadsArchives(game)
	if err != nil || len(all) == 0 {
		return []DownloadArchive{}, err
	}
	seen := s.d.Seen(game)
	newest := all[0].Mtime
	if newest <= seen {
		return []DownloadArchive{}, nil
	}
	if err := s.d.SetSeen(game, newest); err != nil {
		return nil, err
	}
	fresh := []DownloadArchive{}
	for _, a := range all {
		if seen != 0 && a.Mtime > seen && !ids.Is(strings.TrimSuffix(a.Name, filepath.Ext(a.Name))) {
			fresh = append(fresh, a)
		}
	}
	return fresh, nil
}

// DownloadsFolders lists the folders DownloadsArchives reads, so the dialog can say where to put an archive.
func (s *Service) DownloadsFolders() []string { return s.dirs() }

func (s *Service) dirs() []string {
	if s.d.Dirs == nil {
		return nil
	}
	return slices.DeleteFunc(s.d.Dirs(), func(d string) bool { return d == "" })
}

// InstallDownload adds the archive at path to the profile, recording its Nexus mod id when the file name has one so
// the mod is tracked for updates.
func (s *Service) InstallDownload(ctx context.Context, game, profileID, path string) (profile.InstallResult, error) {
	if s.d.Install == nil {
		return profile.InstallResult{}, errors.New("install is not available")
	}
	var src profile.Source
	if inf := dlwatch.Identify(path); inf.ModID > 0 {
		src = profile.Source{Kind: profile.KindNexus, Name: inf.Name, ModID: inf.ModID}
	}
	return s.d.Install(ctx, game, profileID, path, src)
}

// HandoffPage is a page a pasted address names, on a site Mortar sends the player to and fetches nothing from: the
// source kind, the id a file saved from it is recorded under (a Patreon post id, an itch.io "user/game" page name), and
// the address to open, rebuilt from the id alone.
type HandoffPage struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	URL  string `json:"url"`
}

var handoffSites = []struct {
	kind  string
	parse func(text string) (string, bool)
	url   func(id string) string
	valid func(id string) bool
}{
	{profile.KindPatreon, patreon.ParsePostURL, patreon.PostURL, patreon.ValidID},
	{profile.KindItch, itch.ParsePageURL, itch.PageURL, itch.ValidPage},
}

// HandoffPage reads a pasted Patreon post or itch.io game page address. The window opens URL in the browser, the player
// saves the file from the page, and InstallHandoffDownload adds it.
func (s *Service) HandoffPage(text string) (HandoffPage, error) {
	for _, site := range handoffSites {
		if id, ok := site.parse(text); ok {
			return HandoffPage{Kind: site.kind, ID: id, URL: site.url(id)}, nil
		}
	}
	return HandoffPage{}, usererr.New(usererr.Invalid, "this is not the address of a Patreon post or an itch.io game page")
}

// InstallHandoffDownload is InstallDownload for a file the player saved from page id of a HandoffPage kind: the entry
// records the page, so its page link and a share carry the page and no file.
func (s *Service) InstallHandoffDownload(ctx context.Context, game, profileID, path, kind, id string) (profile.InstallResult, error) {
	if s.d.Install == nil {
		return profile.InstallResult{}, errors.New("install is not available")
	}
	for _, site := range handoffSites {
		if site.kind == kind && site.valid(id) {
			return s.d.Install(ctx, game, profileID, path, profile.Source{Kind: kind, Name: id})
		}
	}
	return profile.InstallResult{}, usererr.New(usererr.Invalid, "not a page a saved file can be recorded under")
}

func (s *Service) storeKeys(game string) (map[string]bool, error) {
	out := map[string]bool{}
	if s.d.Keys == nil {
		return out, nil
	}
	keys, err := s.d.Keys(game)
	for _, k := range keys {
		out[k] = true
	}
	return out, err
}

func (s *Service) profileNames(game string) (map[string]bool, error) {
	out := map[string]bool{}
	if s.d.Profiles == nil {
		return out, nil
	}
	all, err := s.d.Profiles(game)
	if err != nil {
		return nil, err
	}
	for _, p := range all {
		for _, e := range p.Entries {
			if e.Source.Kind == profile.KindLocal {
				out[e.Source.Name] = true
			}
		}
	}
	return out, nil
}

func isArchive(name string) bool { return archive.HasExtension(name) }
