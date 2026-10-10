package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"

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

// placeTray puts an archive's Tray files in the game's Tray folder and returns what the entry holds. A file already
// there that no entry placed is the player's own and is never replaced: it is left (when identical) or the install is
// refused. The names of files this call placed come back too, for the caller to take away again if the install fails.
func (s *Store) placeTray(game string, p *Profile, arch installer.Archive, files []installer.File) (held []TrayFile, placed []string, err error) {
	if len(files) == 0 {
		return nil, nil, nil
	}
	dir, err := s.trayFolder(game)
	if err != nil {
		return nil, nil, err
	}
	owners := s.trayOwners(game, p)
	undo := func(err error) ([]TrayFile, []string, error) {
		for _, r := range placed {
			_ = fsx.Remove(filepath.Join(dir, filepath.FromSlash(r)))
		}
		return nil, nil, err
	}
	for _, f := range files {
		dst := filepath.Join(dir, filepath.FromSlash(f.Rel))
		if !filepath.IsLocal(filepath.FromSlash(f.Rel)) {
			return undo(fmt.Errorf("%s leaves the Tray folder", f.Rel))
		}
		src := filepath.Join(arch.Dir, filepath.FromSlash(f.Src))
		h, err := fsx.SHA256(src)
		if err != nil {
			return undo(err)
		}
		if oh, ours := owners[f.Rel]; ours {
			if oh != h {
				return undo(refuseTray(f.Rel, "another mod in your profiles placed a different file there"))
			}
			held = append(held, TrayFile{Rel: f.Rel, Hash: h})
			continue
		}
		if _, err := os.Lstat(dst); err == nil {
			if dh, err := fsx.SHA256(dst); err == nil && dh == h {
				continue
			}
			return undo(refuseTray(f.Rel, "it is a file of your own"))
		} else if !errors.Is(err, fs.ErrNotExist) {
			return undo(err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return undo(err)
		}
		if err := datadir.CopyFile(src, dst); err != nil {
			return undo(err)
		}
		placed = append(placed, f.Rel)
		held = append(held, TrayFile{Rel: f.Rel, Hash: h})
	}
	return held, placed, nil
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

// withoutTray is entries minus those that hold Tray files.
func withoutTray(entries []Entry) []Entry {
	return slices.DeleteFunc(slices.Clone(entries), func(e Entry) bool { return len(e.TrayFiles) > 0 })
}
