package profile

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/bytefmt"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// InstallResult is the profile after an install and the names of the mods it added.
type InstallResult struct {
	Profile Profile  `json:"profile"`
	Added   []string `json:"added"`
	// Updated is true when the archive replaced a version of an entry already in the profile.
	Updated bool `json:"updated"`
	// VersionChanged is true when Updated and any replaced mod's version string differs.
	VersionChanged bool `json:"versionChanged"`
	// Fomod is set when the store item has install options and they have not been chosen yet.
	Fomod *FomodAsk `json:"fomod,omitempty"`
	// Remap is set when the archive has no SMAPI-reachable manifest, or several variants of one mod, and the user must
	// pick a folder.
	Remap *RemapAsk `json:"remap,omitempty"`
}

// InstallError is a failed install. Its message is fit to show the user; Err keeps the typed cause.
type InstallError struct {
	Msg string
	Err error
}

func (e *InstallError) Error() string { return e.Msg }

func (e *InstallError) Unwrap() error { return e.Err }

// InstallArchive unpacks the archive at path into the store and adds it to the profile as a local entry.
func (s *Store) InstallArchive(ctx context.Context, game, id, path string) (InstallResult, error) {
	return s.InstallSaved(ctx, game, id, path, Source{Kind: KindLocal, Name: filepath.Base(path)})
}

// InstallSaved unpacks an archive the player saved themselves and adds it as a local file would be, recorded under
// source: a page they were sent to (a Patreon post id, an itch.io "user/game" page), so the entry's page link and a
// share carry the page and no file.
func (s *Store) InstallSaved(ctx context.Context, game, id, path string, source Source) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	key, err := s.items.AddArchive(ctx, game, path)
	if err != nil {
		return InstallResult{}, installError(err)
	}
	if err := s.namePackage(game, key, filepath.Base(path), ""); err != nil {
		return InstallResult{}, installError(err)
	}
	return s.installKey(game, id, key, source)
}

// InstallFolder copies the folder at path into the store under a content key and adds it to the profile as a local entry.
func (s *Store) InstallFolder(ctx context.Context, game, id, path string) (InstallResult, error) {
	return s.InstallLocalFolder(ctx, game, id, path, true)
}

// InstallLocalFolder is InstallFolder for a mod that may arrive switched off, as another manager's local mod can.
func (s *Store) InstallLocalFolder(ctx context.Context, game, id, path string, enabled bool) (InstallResult, error) {
	return s.installDir(ctx, game, id, path, Source{Kind: KindLocal, Name: filepath.Base(path)}, enabled)
}

// InstallExtraFolder is InstallFolder for a mod in an extra mods folder, whose entry remembers the folder.
func (s *Store) InstallExtraFolder(ctx context.Context, game, id, path string) (InstallResult, error) {
	src := Source{Kind: KindLocal, Name: filepath.Base(path), Folder: path}
	if _, err := os.Stat(filepath.Join(path, ".mortar", "project.json")); err != nil {
		return s.installDir(ctx, game, id, path, src, true)
	}
	// An asset project's folder is its repository: the package is what is left without the repository's own files.
	scratch, err := os.MkdirTemp(filepath.Dir(s.root), "mortar-asset-*")
	if err != nil {
		return InstallResult{}, installError(err)
	}
	defer func() { _ = fsx.RemoveAll(scratch) }()
	staged := filepath.Join(scratch, filepath.Base(path))
	if err := copyWithout(path, staged, ".git", ".mortar", "artifacts"); err != nil {
		return InstallResult{}, installError(err)
	}
	return s.installDir(ctx, game, id, staged, src, true)
}

func (s *Store) installDir(ctx context.Context, game, id, path string, source Source, enabled bool) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	key, err := s.items.AddHashedDir(ctx, game, path)
	if err != nil {
		return InstallResult{}, installError(err)
	}
	res, err := s.installKey(game, id, key, source)
	if err != nil || enabled {
		return res, err
	}
	var refs []EnableRef
	for _, e := range res.Profile.Entries {
		if e.StoreKey() == key {
			for _, m := range e.Mods {
				refs = append(refs, EnableRef{Key: e.Key, ID: m.ID})
			}
		}
	}
	if len(refs) > 0 {
		res.Profile, err = s.SetModsEnabled(game, id, refs, false)
	}
	return res, err
}

// InstallSource unpacks the archive at path into the store under the key of its source's file and adds it to the profile,
// replacing the version of a package the profile already holds. A Nexus file is keyed by mod and file id, a Thunderstore
// package by name and version.
func (s *Store) InstallSource(ctx context.Context, game, id, path string, source Source) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	var key string
	switch source.Kind {
	case KindNexus:
		key = store.NexusKey(source.ModID, source.FileID)
	case KindThunderstore:
		key = store.PackageKey(source.Name, source.Version)
	case KindCurseForge:
		// File display names are author-written and repeat, so the file id keys the store item.
		key = store.PackageKey(source.Kind+":"+source.Name, strconv.Itoa(source.FileID))
	case KindModrinth, KindItch, KindVSModDB, KindParadox:
		key = store.PackageKey(source.Kind+":"+source.Name, source.Version)
	default:
		return InstallResult{}, installError(fmt.Errorf("cannot install a %q archive", source.Kind))
	}
	if err := s.items.AddArchiveKey(ctx, game, key, path); err != nil {
		return InstallResult{}, installError(err)
	}
	if source.Kind != KindNexus {
		if err := s.items.Describe(game, key, source.Kind, source.Name, source.Version, ""); err != nil {
			return InstallResult{}, installError(err)
		}
	}
	return s.installKey(game, id, key, source)
}

// StageGitHub unpacks the archive at path into the store under the key of its GitHub asset and returns the key with
// the mod ids of the mods it holds, so the source can be checked before anything lands in a profile.
func (s *Store) StageGitHub(ctx context.Context, game string, source Source, path string) (key string, uniqueIDs []mod.ID, err error) {
	owner, repo, _ := strings.Cut(source.Repo, "/")
	key = github.Key(owner, repo, source.Tag, source.Asset)
	if err := s.items.AddArchiveKey(ctx, game, key, path); err != nil {
		return "", nil, installError(err)
	}
	if err := s.items.Describe(game, key, KindGitHub, source.Repo, source.Tag, source.Asset); err != nil {
		return "", nil, installError(err)
	}
	if err := s.namePackage(game, key, source.Asset, owner); err != nil {
		return "", nil, installError(err)
	}
	dir, err := s.items.Path(game, key)
	if err != nil {
		return "", nil, installError(err)
	}
	found, err := smapimanifest.Scan(dir)
	if err != nil {
		return "", nil, installError(err)
	}
	for _, m := range found {
		uniqueIDs = append(uniqueIDs, m.ModID())
	}
	return key, uniqueIDs, nil
}

// InstallStaged adds the store item key, staged by StageGitHub, to the profile, replacing the version of a mod the
// profile already holds.
func (s *Store) InstallStaged(game, id, key string, source Source) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	return s.installKey(game, id, key, source)
}

// installQuestion turns an install that stopped for the user's choice (FOMOD options or which folder
// is the mod's root) into the result that asks it, with the profile as it stands.
func (s *Store) installQuestion(game, id, key string, source Source, err error) (InstallResult, bool) {
	current := func() Profile {
		cur, rerr := s.read(game, id)
		if rerr != nil {
			return Profile{}
		}
		return cur
	}
	if need, ok := errors.AsType[*NeedChoicesError](err); ok {
		need.Ask.Key, need.Ask.Source = key, source
		return InstallResult{Profile: current(), Added: []string{}, Fomod: &need.Ask}, true
	}
	if need, ok := errors.AsType[*NeedRootError](err); ok {
		need.Ask.Key, need.Ask.Source = key, source
		return InstallResult{Profile: current(), Added: []string{}, Remap: &need.Ask}, true
	}
	return InstallResult{}, false
}

// addedNames lists the mods of the entries holding key; never nil, so it serialises as [].
func addedNames(p Profile, key string) []string {
	names := []string{}
	for _, e := range p.Entries {
		if e.StoreKey() == key {
			for _, m := range e.Mods {
				names = append(names, m.Name)
			}
			if e.IsOverlay() {
				names = append(names, entryLabel(e))
			}
		}
	}
	return names
}

func (s *Store) installKey(game, id, key string, source Source) (InstallResult, error) {
	p, updated, versionChanged, err := s.placeKey(game, id, key, source)
	if ask, ok := s.installQuestion(game, id, key, source, err); ok {
		return ask, nil
	}
	if err != nil {
		return InstallResult{}, installError(err)
	}
	res := InstallResult{Profile: p, Added: addedNames(p, key), Updated: updated, VersionChanged: versionChanged}
	if err := s.RecordModsSnapshot(game, id); err != nil {
		return InstallResult{}, err
	}
	if err := s.recordOverride(game, id, key); err != nil {
		return InstallResult{}, err
	}
	return res, nil
}

// placeKey adds key to the profile, or swaps it in for the one entry already holding its mods. The decision and the
// change share one lock, so two concurrent installs of the same mod cannot both add an entry.
func (s *Store) placeKey(game, id, key string, source Source) (Profile, bool, bool, error) {
	if err := s.lockUnlocked(game, id); err != nil {
		return Profile{}, false, false, err
	}
	defer s.mu.Unlock()
	if over, err := s.isOverlayItem(game, key, source); err != nil {
		return Profile{}, false, false, err
	} else if over {
		p, err := s.placeOverlayLocked(game, id, key, source)
		return p, false, false, err
	}
	if mods, isPackage, err := s.packageMods(game, key, source.ModName); err != nil {
		return Profile{}, false, false, err
	} else if isPackage {
		return s.placePackageLocked(game, id, key, source, mods)
	}
	if !readsManifests(game) {
		return Profile{}, false, false, &NoPackageError{Key: key}
	}
	source, ask, need, err := s.installAsk(game, id, key, source)
	if err != nil {
		return Profile{}, false, false, err
	} else if need {
		return Profile{}, false, false, &NeedChoicesError{Ask: ask}
	}
	if _, hasFomod, err := s.fomodOf(game, key); err != nil {
		return Profile{}, false, false, err
	} else if !hasFomod {
		if ask, need, err := s.remapAsk(game, id, key); err != nil {
			return Profile{}, false, false, err
		} else if need {
			return Profile{}, false, false, &NeedRootError{Ask: ask}
		}
	}
	held, err := s.holding(game, id, key, source.fomodMap())
	if err != nil {
		return Profile{}, false, false, err
	}
	if len(held) == 0 && (source.replacing > 0 || source.Folder != "") {
		p, err := s.read(game, id)
		if err != nil {
			return Profile{}, false, false, err
		}
		if i := slices.IndexFunc(p.Entries, source.replaces); i >= 0 {
			held = p.Entries[i : i+1]
		}
	}
	switch len(held) {
	case 0:
		p, err := s.addEntryLocked(game, id, key, source)
		return p, false, false, err
	case 1:
		old := held[0].Mods
		p, err := s.moveToLocked(game, id, held[0].Key, key, &source)
		if err != nil {
			return Profile{}, true, false, err
		}
		var neu []Component
		for _, e := range p.Entries {
			if e.Key == key {
				neu = e.Mods
				break
			}
		}
		return p, true, modsVersionChanged(old, neu), nil
	}
	labels := make([]string, len(held))
	for i, e := range held {
		labels[i] = entryLabel(e)
	}
	return Profile{}, false, false, &SpansEntriesError{Labels: labels}
}

// replaces reports whether e is the entry this install updates though none of its mods match: the Nexus file it is a
// newer version of (replacing: one the queue matched by file_updates or as the page's only retired main file, or one
// the user is updating), or the mod last installed from the same extra mods folder.
func (s Source) replaces(e Entry) bool {
	if e.IsOverlay() {
		return false
	}
	if s.replacing > 0 {
		return e.Source.Kind == KindNexus && e.Source.ModID == s.ModID && e.Source.FileID == s.replacing
	}
	return s.Folder != "" && e.Source.Kind == KindLocal && e.Source.Folder == s.Folder
}

func modsVersionChanged(old, neu []Component) bool {
	prev := make(map[string]string, len(old))
	for _, m := range old {
		prev[m.ID.Fold()] = m.Version
	}
	for _, m := range neu {
		if prev[m.ID.Fold()] != m.Version {
			return true
		}
		delete(prev, m.ID.Fold())
	}
	return len(prev) > 0
}

// holding returns the profile's entries that hold any mod of the store item key. An entry that is already key is a
// DuplicateError.
func (s *Store) holding(game, id, key string, choices map[string]map[string][]string) ([]Entry, error) {
	_, found, done, err := s.scanItem(game, id, key, choices)
	defer done()
	if err != nil {
		return nil, err
	}
	p, err := s.read(game, id)
	if err != nil {
		return nil, err
	}
	var held []Entry
	for _, e := range p.Entries {
		if e.Key == key {
			return nil, &DuplicateError{Key: key, Label: entryLabel(e)}
		}
		if holdsAny(e, found) {
			held = append(held, e)
		}
	}
	return held, nil
}

func installError(err error) error {
	var dup *DuplicateError
	var span *SpansEntriesError
	var full *store.DiskFullError
	var msg string
	switch {
	case errors.Is(err, archive.ErrUnsupportedFormat):
		msg = "Mortar reads zip, RAR, 7z, tar, gzip, xz, lzma, zstd and bzip2 archives"
	case errors.Is(err, archive.ErrEncrypted):
		msg = "The archive is encrypted, and Mortar cannot open it"
	case errors.Is(err, archive.ErrChecksum):
		msg = "The archive is damaged"
	case errors.Is(err, archive.ErrCaseCollision):
		// Windows, and Wine under Proton, cannot keep both names, so either copy could silently be the one that loads.
		msg = "The archive holds two files whose names differ only in capital letters, which Windows cannot keep apart"
		if ae, ok := errors.AsType[*archive.Error](err); ok && ae.Entry != "" {
			msg += ": " + ae.Entry
		}
	case errors.Is(err, archive.ErrTraversal), errors.Is(err, archive.ErrUnsafeName),
		errors.Is(err, archive.ErrLink), errors.Is(err, archive.ErrSpecialFile):
		msg = "The archive holds files that are not safe to unpack"
	case errors.Is(err, archive.ErrEntryTooLarge):
		msg = fmt.Sprintf("A file in the archive is over %d GB, more than Mortar unpacks", archive.DefaultMaxEntryBytes>>30)
	case errors.Is(err, archive.ErrRatio):
		msg = "The archive unpacks to far more than its size should hold, so it is not safe to unpack"
	case errors.Is(err, archive.ErrArchiveTooLarge):
		msg = fmt.Sprintf("The archive unpacks to over %d GB, more than Mortar unpacks", archive.DefaultMaxTotalBytes>>30)
	case errors.Is(err, archive.ErrTooManyEntries):
		msg = fmt.Sprintf("The archive holds over %d files, more than Mortar unpacks", archive.DefaultMaxEntries)
	case errors.As(err, &full):
		msg = fmt.Sprintf("Not enough disk space: about %s is needed", bytefmt.Format(full.NeedMB<<20))
	case errors.As(err, &dup):
		msg = fmt.Sprintf("%s is already in this profile", dup.Label)
	case errors.As(err, &span):
		msg = fmt.Sprintf("This archive's mods are in several entries of this profile (%s): remove all but one, then install again", strings.Join(span.Labels, "; "))
	case errors.As(err, new(*rawXNBError)):
		msg = "This file replaces game files directly (raw .xnb). Mortar installs SMAPI mods; use the mod's Content Patcher version."
	case errors.As(err, new(*NoBaseError)):
		return &InstallError{Msg: err.Error(), Err: usererr.Wrap(usererr.NotFound, err)}
	case errors.As(err, new(*NoPackageError)):
		msg = "Mortar found no mod for this game in this archive. It expects a Thunderstore package (a manifest.json with a name and version), or for a BepInEx or MelonLoader game a plugin DLL or the loader's own folder"
	case errors.As(err, new(*NoModError)):
		msg = "No SMAPI mod was found in this archive"
	case errors.Is(err, store.ErrIncomplete):
		msg = "This mod is only partly in Mortar's store; add the archive again"
	default:
		return err
	}
	return &InstallError{Msg: msg, Err: err}
}

// holdsAny reports whether e holds any of the mods found.
func holdsAny(e Entry, found []smapimanifest.Mod) bool {
	return slices.ContainsFunc(e.Mods, func(m Component) bool {
		return slices.ContainsFunc(found, func(f smapimanifest.Mod) bool { return mod.Equal(f.ModID(), m.ID) })
	})
}

// copyWithout copies the regular files and folders under src to dst, leaving out the named top-level entries.
func copyWithout(src, dst string, skip ...string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if slices.Contains(skip, rel) {
			return filepath.SkipDir
		}
		to := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(to, 0o700)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := fsx.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(to, b, 0o600)
	})
}
