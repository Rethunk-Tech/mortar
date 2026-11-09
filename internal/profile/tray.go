package profile

import (
	"cmp"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// trayRole is the path role of the Tray folder, which a TrayFile with no role belongs to.
const trayRole = "tray"

// TrayFile is a file placed in a shared folder of the game and the SHA-256 of what was placed. Role is the path role
// of that folder, empty for the Tray folder.
type TrayFile struct {
	Rel  string `json:"rel"`
	Hash string `json:"hash"`
	Role string `json:"role,omitempty"`
}

// folded is the file's identity across the profiles of a game: its role and its path as the file system folds it.
func (f TrayFile) folded() string { return trayFold(cmp.Or(f.Role, trayRole) + "|" + f.Rel) }

// trayFileRole is the TrayFile role for a path role: empty for the Tray folder, so older entries read the same.
func trayFileRole(role string) string {
	if role == trayRole {
		return ""
	}
	return role
}

// trayEntryKey is the key of the entry that holds an archive's Tray files beside its mods entries. A laid-out file's
// path is relative and never starts with a slash, so no file's key can equal it.
func trayEntryKey(item string) string { return item + fileKeySep + "/tray" }

// isTrayEntry reports the entry that holds an archive's Tray files; it lays nothing out in the profile.
func (e Entry) isTrayEntry() bool { return e.Tray }

// InTray reports whether the entry's shared files are in the game's Tray folder, rather than another shared folder.
func (e Entry) InTray() bool {
	return e.Tray && !slices.ContainsFunc(e.TrayFiles, func(f TrayFile) bool { return cmp.Or(f.Role, trayRole) != trayRole })
}

// TrayRestoreError reports Tray files a restored profile's entries could not place again, because the Tray folder holds
// a different file of that name. The profile itself is restored.
type TrayRestoreError struct{ Files []string }

func (e *TrayRestoreError) Error() string {
	return fmt.Sprintf("the profile is restored, but these Tray files could not be placed again because the Tray folder has different files of the same name: %s", strings.Join(e.Files, ", "))
}

// trayFold folds a Tray file's relative path as the file system does, so on Windows `House.trayitem` and
// `house.trayitem` are one file with one set of owners. Tests replace it to fold on any system.
var trayFold = safefs.FoldCase

// trayOwners maps each Tray file an entry of the game's profiles holds to the hash it recorded. p is the profile
// being changed, taken as it stands in memory.
func (s *Store) trayOwners(game string, p *Profile) map[string]string {
	owners := map[string]string{}
	add := func(es []Entry) {
		for _, e := range es {
			for _, f := range e.TrayFiles {
				owners[f.folded()] = f.Hash
			}
		}
	}
	add(p.Entries)
	if all, err := s.listOK(game); err == nil {
		for _, q := range all {
			if q.ID != p.ID {
				add(q.Entries)
			}
		}
	}
	return owners
}

func (s *Store) trayFolder(game, role string) (string, error) {
	if s.TrayFolder == nil {
		return "", errors.New("this game has no shared folder")
	}
	return s.TrayFolder(game, cmp.Or(role, trayRole))
}

// trayPlacement is the outcome of placeTray: what the entry holds, and how to finish or undo the placement.
type trayPlacement struct {
	held []TrayFile
	// undo takes away what placeTray placed and puts back what it set aside; commit drops what it set aside.
	undo, commit func()
}

func noTray() trayPlacement { return trayPlacement{undo: func() {}, commit: func() {}} }

// placeTray puts an archive's Tray files in the game's Tray folder. A file already there that no entry placed is the
// player's own and is never replaced: it is left (when identical) or the install is refused. replace names files an
// archive being updated placed (by recorded hash): one still unchanged is set aside and replaced, and comes back on
// undo. Nothing is released here, so a failure leaves every earlier file as it was.
func (s *Store) placeTray(game string, p *Profile, arch installer.Archive, files []installer.File, replace map[string]string) (trayPlacement, error) {
	if len(files) == 0 {
		return noTray(), nil
	}
	owners := s.trayOwners(game, p)
	g := installerGame(game)
	var placed []string
	type aside struct{ dst, held string }
	var asides []aside
	pl := trayPlacement{
		undo: func() {
			for _, r := range placed {
				_ = safefs.Remove(r)
			}
			for _, a := range asides {
				_ = safefs.Rename(a.held, a.dst)
			}
		},
		commit: func() {
			for _, a := range asides {
				_ = safefs.Remove(a.held)
			}
		},
	}
	fail := func(err error) (trayPlacement, error) {
		pl.undo()
		return noTray(), err
	}
	for _, f := range files {
		role := trayFileRole(g.RoleOf(f.Target))
		dst, err := s.trayDest(game, role, f.Rel)
		if err != nil {
			return fail(err)
		}
		held := TrayFile{Rel: f.Rel, Role: role}
		key := held.folded()
		src := filepath.Join(arch.Dir, filepath.FromSlash(f.Src))
		h, err := safefs.HashFile(src, sha256.New())
		if err != nil {
			return fail(err)
		}
		held.Hash = h
		if oh, ours := owners[key]; ours {
			if oh != h {
				return fail(refuseTray(f.Rel, "another mod in your profiles placed a different file there"))
			}
			pl.held = append(pl.held, held)
			continue
		}
		skip, held2, setAside, err := trayExisting(dst, f.Rel, key, h, replace)
		if err != nil {
			return fail(err)
		}
		if setAside != "" {
			asides = append(asides, aside{dst, setAside})
		}
		if skip {
			if held2 {
				pl.held = append(pl.held, held)
			}
			continue
		}
		if err := datadir.MkdirParent(dst); err != nil {
			return fail(err)
		}
		if err := datadir.CopyFile(src, dst); err != nil {
			return fail(err)
		}
		placed = append(placed, dst)
		pl.held = append(pl.held, held)
	}
	return pl, nil
}

func refuseTray(rel, why string) error {
	return usererr.Wrap(usererr.Invalid, fmt.Errorf("the Tray folder already has %s and Mortar never replaces it (%s): rename or move it, then add the archive again", rel, why))
}

// releaseTray takes away the Tray files an entry that left p held, unless another entry still holds them or the file
// is no longer the one Mortar placed.
func (s *Store) releaseTray(game string, p *Profile, files []TrayFile) {
	if len(files) == 0 {
		return
	}
	owners := s.trayOwners(game, p)
	for _, f := range files {
		if _, held := owners[f.folded()]; held {
			continue
		}
		dir, err := s.trayFolder(game, f.Role)
		if err != nil {
			slog.Warn("profile: release shared files", "err", err)
			continue
		}
		dst := filepath.Join(dir, filepath.FromSlash(f.Rel))
		if h, err := safefs.HashFile(dst, sha256.New()); err == nil && h == f.Hash {
			removeUp(dir, dst)
		}
	}
}

// replaceTrayLocked places again, from the store, the Tray files of the profile's Tray entries, for a profile that came
// back from the trash or a backup. A file the Tray folder holds under another content is left alone, its entry
// forgets it, and the profile comes back with a TrayRestoreError naming it.
func (s *Store) replaceTrayLocked(game, id string) (Profile, error) {
	p, err := s.read(game, id)
	if err != nil || !slices.ContainsFunc(p.Entries, Entry.isTrayEntry) {
		return p, err
	}
	var failed []string
	p, err = s.updateLocked(game, id, func(p *Profile, _ string) error {
		for i := range p.Entries {
			if !p.Entries[i].isTrayEntry() {
				continue
			}
			if _, err := s.items.Path(game, p.Entries[i].Item); errors.Is(err, store.ErrNotFound) {
				continue
			}
			// The entry's own record would make placeTray take the files for placed already.
			held := p.Entries[i]
			p.Entries[i].TrayFiles = nil
			res := []Entry{held}
			var pl trayPlacement
			if err := s.placeRestoredTray(game, id, p, res, nil, &pl); err != nil {
				for _, f := range held.TrayFiles {
					failed = append(failed, f.Rel)
				}
				continue
			}
			pl.commit()
			p.Entries[i].TrayFiles = res[0].TrayFiles
		}
		return nil
	})
	if err == nil && len(failed) > 0 {
		err = &TrayRestoreError{Files: failed}
	}
	return p, err
}

// trayExisting decides what an occupied Tray path means for a file about to be placed (hash h): skip reports that
// nothing is copied, held that the entry still holds the file, and setAside names the moved-away old copy.
func trayExisting(dst, rel, key, h string, replace map[string]string) (skip, held bool, setAside string, err error) {
	if _, err := os.Lstat(dst); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, false, "", nil
		}
		return false, false, "", err
	}
	dh, herr := safefs.HashFile(dst, sha256.New())
	rec, mine := replace[key]
	switch {
	case herr == nil && mine && dh == rec && dh == h:
		return true, true, "", nil
	case herr == nil && mine && dh == rec:
		setAside = dst + ".mortar-replaced"
		if err := safefs.Rename(dst, setAside); err != nil {
			return false, false, "", err
		}
		return false, false, setAside, nil
	case herr == nil && dh == h:
		return true, false, "", nil
	default:
		return false, false, "", refuseTray(rel, "it is a file of your own")
	}
}

// trayDest is where rel goes in the Tray folder of role, refusing a path that leaves it.
func (s *Store) trayDest(game, role, rel string) (string, error) {
	dir, err := s.trayFolder(game, role)
	if err != nil {
		return "", err
	}
	if !filepath.IsLocal(filepath.FromSlash(rel)) {
		return "", fmt.Errorf("%s leaves the shared folder", rel)
	}
	return filepath.Join(dir, filepath.FromSlash(rel)), nil
}
