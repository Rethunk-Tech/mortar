package profile

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
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
func (s *Store) InstallArchive(game, id, path string) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	key, err := s.items.AddArchive(game, path)
	if err != nil {
		return InstallResult{}, installError(err)
	}
	return s.installKey(game, id, key, Source{Kind: KindLocal, Name: filepath.Base(path)})
}

// InstallFolder copies the folder at path into the store under a content key and adds it to the profile as a local entry.
func (s *Store) InstallFolder(game, id, path string) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	key, err := s.items.AddHashedDir(game, path)
	if err != nil {
		return InstallResult{}, installError(err)
	}
	return s.installKey(game, id, key, Source{Kind: KindLocal, Name: filepath.Base(path)})
}

// InstallNexus unpacks the archive at path into the store under the key of its Nexus file and adds it to the
// profile, replacing the version of a mod the profile already holds.
func (s *Store) InstallNexus(game, id, path string, source Source) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	key := store.NexusKey(source.ModID, source.FileID)
	if err := s.items.AddArchiveKey(game, key, path); err != nil {
		return InstallResult{}, installError(err)
	}
	return s.installKey(game, id, key, source)
}

// StageGitHub unpacks the archive at path into the store under the key of its GitHub asset and returns the key with
// the mod ids of the mods it holds, so the source can be checked before anything lands in a profile.
func (s *Store) StageGitHub(game string, source Source, path string) (key string, uniqueIDs []mod.ID, err error) {
	owner, repo, _ := strings.Cut(source.Repo, "/")
	key = github.Key(owner, repo, source.Tag, source.Asset)
	if err := s.items.AddArchiveKey(game, key, path); err != nil {
		return "", nil, installError(err)
	}
	dir, err := s.items.Path(game, key)
	if err != nil {
		return "", nil, installError(err)
	}
	found, err := manifest.Scan(dir)
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
		if e.Key == key {
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
	return res, nil
}

// placeKey adds key to the profile, or swaps it in for the one entry already holding its mods. The decision and the
// change share one lock, so two concurrent installs of the same mod cannot both add an entry.
func (s *Store) placeKey(game, id, key string, source Source) (Profile, bool, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, false, false, err
	}
	if over, err := s.isOverlayItem(game, key, source); err != nil {
		return Profile{}, false, false, err
	} else if over {
		p, err := s.placeOverlayLocked(game, id, key, source)
		return p, false, false, err
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
		if slices.ContainsFunc(e.Mods, func(m Component) bool {
			return slices.ContainsFunc(found, func(f manifest.Mod) bool { return mod.Equal(f.ModID(), m.ID) })
		}) {
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
		msg = "Mortar reads zip, RAR and 7z archives"
	case errors.Is(err, archive.ErrEncrypted):
		msg = "The archive is encrypted, and Mortar cannot open it"
	case errors.Is(err, archive.ErrChecksum):
		msg = "The archive is damaged"
	case errors.Is(err, archive.ErrTraversal), errors.Is(err, archive.ErrUnsafeName), errors.Is(err, archive.ErrCaseCollision),
		errors.Is(err, archive.ErrLink), errors.Is(err, archive.ErrSpecialFile):
		msg = "The archive holds files that are not safe to unpack"
	case errors.Is(err, archive.ErrEntryTooLarge), errors.Is(err, archive.ErrArchiveTooLarge), errors.Is(err, archive.ErrTooManyEntries):
		msg = "The archive is too large to unpack"
	case errors.As(err, &full):
		msg = fmt.Sprintf("Not enough disk space: about %d MB is needed", full.NeedMB)
	case errors.As(err, &dup):
		msg = fmt.Sprintf("%s is already in this profile", dup.Label)
	case errors.As(err, &span):
		msg = fmt.Sprintf("This archive's mods are in several entries of this profile (%s): remove all but one, then install again", strings.Join(span.Labels, "; "))
	case errors.As(err, new(*rawXNBError)):
		msg = "This file replaces game files directly (raw .xnb). Mortar installs SMAPI mods; use the mod's Content Patcher version."
	case errors.As(err, new(*NoBaseError)):
		return &InstallError{Msg: err.Error(), Err: usererr.Wrap(usererr.NotFound, err)}
	case errors.As(err, new(*NoModError)):
		msg = "No SMAPI mod was found in this archive"
	case errors.Is(err, store.ErrIncomplete):
		msg = "This mod is only partly in Mortar's store; add the archive again"
	default:
		return err
	}
	return &InstallError{Msg: msg, Err: err}
}
