package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// A profile kept outside the data folder (another drive) stays at its usual place as a link to its folder, so every
// path built from the profile's id reaches it. Mods are materialised into it as anywhere: clone and hardlink fall
// back to a copy across filesystems (datadir.MaterializeTree).

// isProfileDir is whether the listed entry is a profile folder, either a real one or a link to one.
func isProfileDir(path string, d fs.DirEntry) bool {
	if d.IsDir() {
		return true
	}
	info, err := d.Info()
	return err == nil && datadir.LinkedDir(path, info)
}

// folderOf is where the profile at its usual place really lives, empty when it lives there.
func folderOf(dir string) string {
	target, err := os.Readlink(dir)
	if err != nil {
		return ""
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(dir), target)
	}
	return target
}

// FolderCheck is whether a profile's folder can be used.
type FolderCheck struct {
	// Folder is the folder in use, empty for a profile under the data folder.
	Folder  string `json:"folder"`
	OK      bool   `json:"ok"`
	Problem string `json:"problem,omitempty"`
}

// CheckFolder reports whether the folder the profile lives in is there, holds its profile.json and can be written.
func (s *Store) CheckFolder(game, id string) (FolderCheck, error) {
	home, err := s.profileDir(game, id)
	if err != nil {
		return FolderCheck{}, err
	}
	out := FolderCheck{Folder: folderOf(home)}
	fail := func(format string, args ...any) (FolderCheck, error) {
		out.Problem = fmt.Sprintf(format, args...)
		return out, nil
	}
	if fi, err := os.Stat(home); err != nil || !fi.IsDir() {
		return fail("the folder %s is not there; if it is on a drive that is not connected, connect it", cmpFolder(out.Folder, home))
	}
	if _, err := os.Stat(filepath.Join(home, fileName)); err != nil {
		return fail("the folder %s holds no profile", cmpFolder(out.Folder, home))
	}
	probe, err := os.CreateTemp(home, ".mortar-write-check-*")
	if err != nil {
		return fail("Mortar cannot write to %s", cmpFolder(out.Folder, home))
	}
	_ = probe.Close()
	_ = os.Remove(probe.Name())
	out.OK = true
	return out, nil
}

func cmpFolder(folder, home string) string {
	if folder != "" {
		return folder
	}
	return home
}

// MoveToFolder moves the profile to <parent>/<game>-<id>, which may be on another drive, and leaves a link in its
// place. The copy is complete and in place before the original goes, so a failure leaves the profile where it was.
// Refused while the game runs the profile.
func (s *Store) MoveToFolder(game, id, parent string) (Profile, error) {
	if !filepath.IsAbs(parent) {
		return Profile{}, usererr.New(usererr.Invalid, "choose a folder by its full path")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	home, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	target := filepath.Join(parent, game+"-"+id)
	if fi, err := os.Stat(parent); err != nil || !fi.IsDir() {
		return Profile{}, usererr.New(usererr.Invalid, "that folder does not exist")
	}
	if datadir.UnderRoot(s.dataDir, target) {
		return Profile{}, usererr.New(usererr.Invalid, "choose a folder outside Mortar's data folder")
	}
	if _, err := os.Lstat(target); err == nil {
		return Profile{}, usererr.New(usererr.Invalid, "that folder already holds a profile folder of this name")
	}
	if err := s.relocate(game, id, home, target); err != nil {
		return Profile{}, err
	}
	return s.read(game, id)
}

// MoveHome brings a profile kept in another folder back under the data folder.
func (s *Store) MoveHome(game, id string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	home, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	if err := s.bringHome(game, id, home); err != nil {
		return Profile{}, err
	}
	return s.read(game, id)
}

// madeForProfile refuses a folder Mortar did not make for this profile. A profile's link is followed and its target
// removed afterwards, so a link the player pointed at some other folder must never reach that removal: the target must
// be named <game>-<id> and hold this profile's own profile.json.
func madeForProfile(dir, game, id string) error {
	refuse := func() error {
		return usererr.New(usererr.Invalid, fmt.Sprintf("%s is not a folder Mortar made for this profile, so it was left alone", dir))
	}
	if filepath.Base(dir) != game+"-"+id {
		return refuse()
	}
	b, err := fsx.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		return refuse()
	}
	var head struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(b, &head) != nil || head.ID != id {
		return refuse()
	}
	return nil
}

// relocate copies the profile's folder to target, then makes home a link to it.
func (s *Store) relocate(game, id, home, target string) error {
	cur, err := fsx.EvalSymlinks(home)
	if err != nil {
		return err
	}
	linked := folderOf(home) != ""
	if linked {
		if err := madeForProfile(cur, game, id); err != nil {
			return err
		}
	}
	tmp := datadir.CopyTempName(target)
	if err := fsx.RemoveAll(tmp); err != nil {
		return err
	}
	if err := datadir.CopyTreeResolvingLinks(cur, tmp); err != nil {
		return errors.Join(err, fsx.RemoveAll(tmp))
	}
	if err := fsx.Rename(tmp, target); err != nil {
		return errors.Join(err, fsx.RemoveAll(tmp))
	}
	forgetProfile(filepath.Join(home, fileName))
	if linked {
		// Moving between two custom folders: the old link goes, the old folder after the new link stands.
		if err := os.Remove(home); err != nil {
			return errors.Join(err, fsx.RemoveAll(target))
		}
	} else if err := fsx.Rename(home, home+".moving"); err != nil {
		return errors.Join(err, fsx.RemoveAll(target))
	}
	if err := linkDir(home, target); err != nil {
		if !linked {
			_ = fsx.Rename(home+".moving", home)
		}
		return errors.Join(usererr.Wrap(usererr.Invalid, err), fsx.RemoveAll(target))
	}
	forgetProfile(filepath.Join(home, fileName))
	if linked {
		return fsx.RemoveAll(cur)
	}
	return fsx.RemoveAll(home + ".moving")
}

// bringHome copies a linked profile back to its usual place and removes the other folder.
func (s *Store) bringHome(game, id, home string) error {
	cur := folderOf(home)
	if cur == "" {
		return nil
	}
	if err := madeForProfile(cur, game, id); err != nil {
		return err
	}
	tmp := datadir.CopyTempName(home)
	if err := fsx.RemoveAll(tmp); err != nil {
		return err
	}
	if err := datadir.CopyTreeResolvingLinks(cur, tmp); err != nil {
		return errors.Join(err, fsx.RemoveAll(tmp))
	}
	forgetProfile(filepath.Join(home, fileName))
	if err := os.Remove(home); err != nil {
		return errors.Join(err, fsx.RemoveAll(tmp))
	}
	if err := fsx.Rename(tmp, home); err != nil {
		return errors.Join(err, linkDir(home, cur))
	}
	return fsx.RemoveAll(cur)
}
