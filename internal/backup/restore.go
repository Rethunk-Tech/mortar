package backup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/saves"
)

// Restore copies the named saves from zipPath into l's folder. An empty folders list restores every save in the
// zip. The current saves are zipped first into backupsDir via Saves.
func Restore(ctx context.Context, zipPath string, l saves.Layout, backupsDir string, folders []string, keep int, now time.Time) error {
	parent := filepath.Dir(l.Dir)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, "restore-*")
	if err != nil {
		return err
	}
	defer func() { _ = safefs.RemoveAll(tmp) }()
	got := saves.Layout{Dir: filepath.Join(tmp, "Saves"), Files: l.Files, Companions: l.Companions, CompanionPaths: l.CompanionPaths, Linked: l.Linked, UserFolders: l.UserFolders}
	want, err := extractSaves(zipPath, got, folders)
	if err != nil {
		return err
	}
	if _, err := Saves(ctx, l, backupsDir, keep, now, Cause{Kind: KindRestore}); err != nil {
		return err
	}
	if err := os.MkdirAll(l.Dir, 0o750); err != nil {
		return err
	}
	for _, folder := range want {
		parts := got.Paths(folder)
		linked := got.LinkedOf(ctx, folder)
		if err := dropCompanions(l, folder, parts); err != nil {
			return err
		}
		for _, part := range parts {
			if err := replace(filepath.Join(got.Dir, filepath.FromSlash(part)), filepath.Join(l.Dir, filepath.FromSlash(part))); err != nil {
				return err
			}
		}
		// The world's minimap comes back with it when the backup holds one; an older backup leaves the current map.
		for _, rel := range linked {
			if err := replace(filepath.Join(tmp, filepath.FromSlash(rel)), filepath.Join(filepath.Dir(l.Dir), filepath.FromSlash(rel))); err != nil {
				return err
			}
		}
	}
	return nil
}

// RestoreCopy puts kept, one of the copies the game keeps of save (saves.Layout.CopiesOf), back over save, companions
// included, after zipping the current saves into backupsDir. The game's copy stays where it is.
func RestoreCopy(ctx context.Context, l saves.Layout, backupsDir, save, kept string, keep int, now time.Time) error {
	if !slices.ContainsFunc(l.CopiesOf(save), func(c saves.Copy) bool { return c.Name == kept }) {
		return fmt.Errorf("%q is not a copy the game keeps of %q", kept, save)
	}
	if _, err := Saves(ctx, l, backupsDir, keep, now, Cause{Kind: KindRestore}); err != nil {
		return err
	}
	pairs := l.CopyPaths(save, kept)
	restored := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		restored = append(restored, pair[1])
	}
	if err := dropCompanions(l, save, restored); err != nil {
		return err
	}
	for _, pair := range pairs {
		src, dst := filepath.Join(l.Dir, filepath.FromSlash(pair[0])), filepath.Join(l.Dir, filepath.FromSlash(pair[1]))
		if err := datadir.CopyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// dropCompanions deletes the companions of save in l's folder that the restored files do not replace: SQLite replays a
// leftover -wal over the restored database, undoing the restore. Companion paths stay: a backup made before they were
// carried holds none, and a restore must not delete a mod's data it cannot put back.
func dropCompanions(l saves.Layout, save string, restored []string) error {
	for _, p := range l.Paths(save)[1:] {
		if !slices.Contains(restored, p) && !slices.Contains(l.CompanionPathsOf(save), p) {
			if err := safefs.RemoveAll(filepath.Join(l.Dir, filepath.FromSlash(p))); err != nil {
				return err
			}
		}
	}
	return nil
}

// replace moves src to dst, putting a dst already there back when the move fails.
func replace(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		old := dst + ".mortar-restore"
		if err := safefs.Rename(dst, old); err != nil {
			return err
		}
		if err := safefs.Rename(src, dst); err != nil {
			_ = safefs.Rename(old, dst)
			return err
		}
		return safefs.RemoveAll(old)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return safefs.Rename(src, dst)
}

// extractSaves unpacks zipPath beside got.Dir, its Saves folder, and lists the saves in it that folders names (all
// of them when it names none).
func extractSaves(zipPath string, got saves.Layout, folders []string) ([]string, error) {
	folders = slices.DeleteFunc(slices.Clone(folders), func(f string) bool { return f == "" })
	if err := archive.Extract(zipPath, filepath.Dir(got.Dir)); err != nil {
		return nil, err
	}
	names, err := got.Names()
	if err != nil {
		return nil, err
	}
	for _, f := range folders {
		if f != "" && !slices.Contains(names, f) {
			return nil, fmt.Errorf("backup has no save %q", f)
		}
	}
	if len(folders) == 0 {
		return names, nil
	}
	return slices.DeleteFunc(names, func(n string) bool { return !slices.Contains(folders, n) }), nil
}
