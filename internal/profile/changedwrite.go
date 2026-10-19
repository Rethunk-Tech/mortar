package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// changedRoots are the profile folders a changed copy lives in: the content folder and the held copies.
func changedRoots(game string) []string {
	info, ok := components.Game(game)
	if !ok || info.Deploy != components.DeployProfile {
		return nil
	}
	t, ok := info.Target("mods")
	if !ok {
		return nil
	}
	return []string{t.ProfileFolder(), changedDir}
}

// ValidChangedPath reports whether rel, a slash path, is a place a changed copy may be written: a clean relative path
// inside the game's content folder or under changed/, with no empty, dot or dot-dot segment and no backslash.
func ValidChangedPath(game, rel string) bool {
	if rel == "" || strings.Contains(rel, `\`) || !filepath.IsLocal(filepath.FromSlash(rel)) {
		return false
	}
	for seg := range strings.SplitSeq(rel, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	for _, root := range changedRoots(game) {
		if strings.HasPrefix(strings.ToLower(rel), strings.ToLower(strings.TrimSuffix(root, "/")+"/")) {
			return true
		}
	}
	return false
}

// WriteChangedFiles writes changed copies that came from another computer into the profile, by slash path from its
// folder, where SyncPackages keeps them as it keeps any copy it did not lay out. Unless overwrite, a file the profile
// already has stays. A path outside the content folder and changed/, or one that reaches a link, is refused.
func (s *Store) WriteChangedFiles(game, id string, files map[string][]byte, overwrite bool) error {
	if err := s.unlocked(game, id); err != nil {
		return err
	}
	_, dir, err := s.readDir(game, id)
	if err != nil {
		return err
	}
	for rel := range files {
		if !ValidChangedPath(game, rel) {
			return fmt.Errorf("changed file %q is not a place the profile keeps one", rel)
		}
		if linkOnPath(dir, rel) {
			return fmt.Errorf("changed file %q reaches a link", rel)
		}
	}
	var written []string
	for rel, data := range files {
		to := filepath.Join(dir, filepath.FromSlash(rel))
		if _, err := os.Lstat(to); err == nil && !overwrite {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
			return err
		}
		if err := fsx.WriteFile(to, data, 0o600); err != nil {
			return err
		}
		written = append(written, rel)
	}
	return s.markCopiesChanged(game, id, dir, written)
}

// markCopiesChanged records the written copies that a package lays out as changed, with a hash no file has, so
// SyncPackages keeps them over the package's own file instead of treating them as an older install's leftover.
func (s *Store) markCopiesChanged(game, id, dir string, rels []string) error {
	files, _, err := s.packageFiles(game, id)
	if err != nil {
		return err
	}
	rec := readPlaced(dir)
	n := 0
	for _, rel := range rels {
		if _, ok := files[rel]; ok {
			rec[rel] = placedRec{Hash: "changed copy", Size: -1}
			n++
		}
	}
	if n == 0 {
		return nil
	}
	return writePlaced(dir, rec)
}

// linkOnPath reports a symbolic link among the folders and the file of rel below dir.
func linkOnPath(dir, rel string) bool {
	p := dir
	for seg := range strings.SplitSeq(rel, "/") {
		p = filepath.Join(p, seg)
		if st, err := os.Lstat(p); err == nil && st.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}
