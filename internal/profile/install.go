package profile

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/archive"
	"github.com/Rethunk-AI/mortar/internal/store"
)

// InstallResult is the profile after an install and the names of the mods it added.
type InstallResult struct {
	Profile Profile  `json:"profile"`
	Added   []string `json:"added"`
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
	p, err := s.AddEntry(game, id, key, Source{Kind: "local", Name: filepath.Base(path)})
	if err != nil {
		return InstallResult{}, installError(err)
	}
	res := InstallResult{Profile: p, Added: []string{}}
	for _, e := range p.Entries {
		if e.Key == key {
			for _, m := range e.Mods {
				res.Added = append(res.Added, m.Name)
			}
		}
	}
	return res, nil
}

func installError(err error) error {
	var dup *DuplicateError
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
	case errors.As(err, new(*NoModError)):
		msg = "No SMAPI mod was found in this archive"
	default:
		return err
	}
	return &InstallError{Msg: msg, Err: err}
}
