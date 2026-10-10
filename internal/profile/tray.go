package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// TrayFile is a file placed in the shared Tray folder and the SHA-256 of what was placed.
type TrayFile struct {
	Rel  string `json:"rel"`
	Hash string `json:"hash"`
}

// trayEntryKey is the key of the entry that holds an archive's Tray files beside its mods entries.
func trayEntryKey(item string) string { return item + fileKeySep + "tray" }

// isTrayEntry reports the entry that holds an archive's Tray files; it lays nothing out in the profile.
func (e Entry) isTrayEntry() bool { return e.Item != "" && e.Key == trayEntryKey(e.Item) }

// TrayRestoreError reports Tray files a restored profile's entries could not place again, because the Tray folder holds
// a different file of that name. The profile itself is restored.
type TrayRestoreError struct{ Files []string }

func (e *TrayRestoreError) Error() string {
	return fmt.Sprintf("the profile is restored, but these Tray files could not be placed again because the Tray folder has different files of the same name: %s", strings.Join(e.Files, ", "))
}

// trayOwners maps each Tray file an entry of the game's profiles holds to the hash it recorded. p is the profile
// being changed, taken as it stands in memory.
func (s *Store) trayOwners(game string, p *Profile) map[string]string {
	owners := map[string]string{}
	add := func(es []Entry) {
		for _, e := range es {
			for _, f := range e.TrayFiles {
				owners[f.Rel] = f.Hash
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

func (s *Store) trayFolder(game string) (string, error) {
	if s.TrayFolder == nil {
		return "", errors.New("this game has no Tray folder")
	}
	return s.TrayFolder(game)
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
	dir, err := s.trayFolder(game)
	if err != nil {
		return noTray(), err
	}
	owners := s.trayOwners(game, p)
	var placed []string
	type aside struct{ dst, held string }
	var asides []aside
	pl := trayPlacement{
		undo: func() {
			for _, r := range placed {
				_ = fsx.Remove(filepath.Join(dir, filepath.FromSlash(r)))
			}
			for _, a := range asides {
				_ = fsx.Rename(a.held, a.dst)
			}
		},
		commit: func() {
			for _, a := range asides {
				_ = fsx.Remove(a.held)
			}
		},
	}
	fail := func(err error) (trayPlacement, error) {
		pl.undo()
		return noTray(), err
	}
	for _, f := range files {
		dst := filepath.Join(dir, filepath.FromSlash(f.Rel))
		if !filepath.IsLocal(filepath.FromSlash(f.Rel)) {
			return fail(fmt.Errorf("%s leaves the Tray folder", f.Rel))
		}
		src := filepath.Join(arch.Dir, filepath.FromSlash(f.Src))
		h, err := fsx.SHA256(src)
		if err != nil {
			return fail(err)
		}
		if oh, ours := owners[f.Rel]; ours {
			if oh != h {
				return fail(refuseTray(f.Rel, "another mod in your profiles placed a different file there"))
			}
			pl.held = append(pl.held, TrayFile{Rel: f.Rel, Hash: h})
			continue
		}
		if _, err := os.Lstat(dst); err == nil {
			dh, herr := fsx.SHA256(dst)
			rec, mine := replace[f.Rel]
			switch {
			case herr == nil && mine && dh == rec && dh == h:
				pl.held = append(pl.held, TrayFile{Rel: f.Rel, Hash: h})
				continue
			case herr == nil && mine && dh == rec:
				held := dst + ".mortar-replaced"
				if err := fsx.Rename(dst, held); err != nil {
					return fail(err)
				}
				asides = append(asides, aside{dst, held})
			case herr == nil && dh == h:
				continue
			default:
				return fail(refuseTray(f.Rel, "it is a file of your own"))
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fail(err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return fail(err)
		}
		if err := datadir.CopyFile(src, dst); err != nil {
			return fail(err)
		}
		placed = append(placed, f.Rel)
		pl.held = append(pl.held, TrayFile{Rel: f.Rel, Hash: h})
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
	dir, err := s.trayFolder(game)
	if err != nil {
		log.Printf("profile: release Tray files: %v", err)
		return
	}
	owners := s.trayOwners(game, p)
	for _, f := range files {
		if _, held := owners[f.Rel]; held {
			continue
		}
		dst := filepath.Join(dir, filepath.FromSlash(f.Rel))
		if h, err := fsx.SHA256(dst); err == nil && h == f.Hash {
			removeUp(dir, dst)
		}
	}
}
