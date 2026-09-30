package profile

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/archive"
	"github.com/Rethunk-AI/mortar/internal/github"
	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/store"
)

// InstallResult is the profile after an install and the names of the mods it added.
type InstallResult struct {
	Profile Profile  `json:"profile"`
	Added   []string `json:"added"`
	// Updated is true when the archive replaced a version of an entry already in the profile.
	Updated bool `json:"updated"`
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
// the UniqueIDs of the mods it holds, so the source can be checked before anything lands in a profile.
func (s *Store) StageGitHub(game string, source Source, path string) (key string, uniqueIDs []string, err error) {
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
		uniqueIDs = append(uniqueIDs, m.UniqueID)
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

func (s *Store) installKey(game, id, key string, source Source) (InstallResult, error) {
	p, updated, err := s.placeKey(game, id, key, source)
	if err != nil {
		return InstallResult{}, installError(err)
	}
	res := InstallResult{Profile: p, Added: []string{}, Updated: updated}
	for _, e := range p.Entries {
		if e.Key == key {
			for _, m := range e.Mods {
				res.Added = append(res.Added, m.Name)
			}
		}
	}
	return res, nil
}

// placeKey adds key to the profile, or swaps it in for the one entry already holding its mods. The decision and the
// change share one lock, so two concurrent installs of the same mod cannot both add an entry.
func (s *Store) placeKey(game, id, key string, source Source) (Profile, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, false, err
	}
	held, err := s.holding(game, id, key)
	if err != nil {
		return Profile{}, false, err
	}
	switch len(held) {
	case 0:
		p, err := s.addEntryLocked(game, id, key, source)
		return p, false, err
	case 1:
		p, err := s.moveToLocked(game, id, held[0].Key, key, &source)
		return p, true, err
	}
	labels := make([]string, len(held))
	for i, e := range held {
		labels[i] = entryLabel(e)
	}
	return Profile{}, false, &SpansEntriesError{Labels: labels}
}

// holding returns the profile's entries that hold any mod of the store item key. An entry that is already key is a
// DuplicateError.
func (s *Store) holding(game, id, key string) ([]Entry, error) {
	src, err := s.items.Path(game, key)
	if err != nil {
		return nil, err
	}
	found, err := manifest.Scan(src)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, &NoModError{Key: key}
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
		if slices.ContainsFunc(e.Mods, func(m EntryMod) bool {
			return slices.ContainsFunc(found, func(f manifest.Mod) bool { return sameID(f.UniqueID, m.UniqueID) })
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
	case errors.As(err, new(*NoModError)):
		msg = "No SMAPI mod was found in this archive"
	default:
		return err
	}
	return &InstallError{Msg: msg, Err: err}
}
