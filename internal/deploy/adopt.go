package deploy

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

const (
	// maxAdoptFile and maxAdoptTotal bound what one purge copies into the profile: a larger file a mod wrote (a cache or
	// a world) stays shared instead of being copied on every exit.
	maxAdoptFile  = 64 << 20
	maxAdoptTotal = 512 << 20
)

// listOwned lists the regular files already in the owned folders, before Apply moves anything.
func listOwned(owned []Owned) []string {
	var out []string
	for _, ow := range owned {
		_ = filepath.WalkDir(ow.Dst, func(path string, d fs.DirEntry, err error) error {
			if err == nil && d.Type().IsRegular() {
				out = append(out, path)
			}
			return nil
		})
	}
	return out
}

// adopt moves each file created during play in a folder a profile entry owns into the profile's folder of that name:
// the bytes are copied (temp file and rename), logged, and only then removed from the shared folder, so a crash leaves
// them in one place or both, never neither. Files the player already had there when the deploy began (whatever the game did to them), files the plan placed, and
// files outside an owned folder (the role's root, a folder no entry owns) stay as the player's own. A profile folder
// that is gone adopts nothing and deletes nothing.
func adopt(m *Manifest, l *opLog) error {
	if len(m.Owned) == 0 {
		return nil
	}
	ad := &adopter{m: m, l: l, placed: map[string]bool{}, existing: map[string]bool{}}
	for _, o := range m.Ops {
		ad.placed[safefs.FoldCase(o.Dst)] = true
	}
	for _, f := range m.Existing {
		ad.existing[safefs.FoldCase(f)] = true
	}
	for _, ow := range m.Owned {
		if _, err := os.Stat(filepath.Dir(ow.Src)); err != nil {
			continue
		}
		ad.adopted = nil
		err := filepath.WalkDir(ow.Dst, func(path string, d fs.DirEntry, err error) error {
			return ad.visit(ow, path, d, err)
		})
		if err != nil {
			return fmt.Errorf("adopt %s: %w", ow.Dst, err)
		}
		removeEmptyBelow(ow.Dst, ad.adopted)
	}
	return nil
}

// adopter carries one adopt pass across its owned folders.
type adopter struct {
	m        *Manifest
	l        *opLog
	placed   map[string]bool
	existing map[string]bool
	total    int64
	adopted  []string
}

// visit adopts one entry of an owned folder's walk.
func (ad *adopter) visit(ow Owned, path string, d fs.DirEntry, err error) error {
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if d.IsDir() {
		return nil
	}
	info, err := d.Info()
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || ad.placed[safefs.FoldCase(path)] || ad.existing[safefs.FoldCase(path)] {
		return nil
	}
	if info.Size() > maxAdoptFile || ad.total+info.Size() > maxAdoptTotal {
		log.Printf("deploy: %s stays shared: too large to adopt into the profile (%d bytes)", path, info.Size())
		ad.m.notes = append(ad.m.notes, Note{Path: path, Size: info.Size(), Kind: NoteTooLarge})
		return nil
	}
	if err := adoptFile(ow, path, ad.l); err != nil {
		return err
	}
	ad.total += info.Size()
	ad.adopted = append(ad.adopted, path)
	return nil
}

// adoptFile copies one shared-folder file into the owning profile folder, logs it, then removes the original.
func adoptFile(ow Owned, path string, l *opLog) error {
	rel, err := filepath.Rel(ow.Dst, path)
	if err != nil {
		return err
	}
	target := filepath.Join(ow.Src, rel)
	if err := datadir.MkdirParent(target); err != nil {
		return err
	}
	if err := datadir.CopyFile(path, target); err != nil {
		return err
	}
	at("adopted-copy:" + rel)
	if err := l.add(step{A: path}, false); err != nil {
		return err
	}
	if err := safefs.Remove(path); err != nil {
		return err
	}
	at("adopted-removed:" + rel)
	return nil
}

// removeEmptyBelow removes the folders adopted files leave empty, deepest first, up to and including root.
func removeEmptyBelow(root string, files []string) {
	var dirs []string
	for _, f := range files {
		for d := filepath.Dir(f); len(d) >= len(root); d = filepath.Dir(d) {
			if !slices.Contains(dirs, d) {
				dirs = append(dirs, d)
			}
			if d == root {
				break
			}
		}
	}
	slices.SortFunc(dirs, func(a, b string) int { return len(b) - len(a) })
	for _, d := range dirs {
		_ = safefs.Remove(d)
	}
}
