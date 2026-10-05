package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

// restoreDir holds a restored profile's mod config files until their mods are placed, by entry key: a mod restored
// from a backup is downloaded again, and its folder only exists once its store item does.
const restoreDir = "restore"

// maxBackupConfig caps one mod config file in a backup, as a share caps it.
const maxBackupConfig = 1 << 20

// backupTrees are the profile's own folders a backup carries whole: its history and the loader's config folder.
var backupTrees = []string{snapshotsDir, historyFilesDir, "BepInEx/config"}

// backupFiles are the profile's own files a backup carries.
var backupFiles = []string{historyFile, snapshotKeysFile}

// Backup returns the profile and its files that are not mod files: history, snapshots and kept config captures, the
// cover, the loader's config folder, and each mod folder's .json config files (as mods/<key>/<path>), by slash path.
func (s *Store) Backup(game, id string) (Profile, map[string][]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return Profile{}, nil, err
	}
	files := map[string][]byte{}
	read := func(rel string) error {
		b, err := fsx.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		files[rel] = b
		return err
	}
	names := slices.Clone(backupFiles)
	if coverType(p.Cover) != "" {
		names = append(names, p.Cover)
	}
	for _, rel := range names {
		if err := read(rel); err != nil {
			return Profile{}, nil, err
		}
	}
	walk := func(root, prefix string, keep func(rel string, size int64) bool) error {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || !d.Type().IsRegular() {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil || !keep(filepath.ToSlash(rel), info.Size()) {
				return err
			}
			b, err := fsx.ReadFile(path)
			files[prefix+filepath.ToSlash(rel)] = b
			return err
		})
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, tree := range backupTrees {
		if err := walk(filepath.Join(dir, filepath.FromSlash(tree)), tree+"/", func(string, int64) bool { return true }); err != nil {
			return Profile{}, nil, err
		}
	}
	modsDir := filepath.Join(dir, "mods")
	for _, e := range p.Entries {
		if e.Source.Bundled() || e.IsOverlay() || !e.hasFolder() {
			continue
		}
		folder := filepath.Join(modsDir, e.Key)
		if !exists(folder) {
			folder = filepath.Join(modsDir, "."+e.Key)
		}
		err := walk(folder, "mods/"+e.Key+"/", func(rel string, size int64) bool {
			return strings.EqualFold(filepath.Ext(rel), ".json") && !strings.EqualFold(filepath.Base(rel), manifest.FileName) && size <= maxBackupConfig
		})
		if err != nil {
			return Profile{}, nil, err
		}
	}
	return p, files, nil
}

// RestoreBackup makes a new profile from a backup, under a name no profile of the game has, and returns it with the entries
// whose store items are missing here, which the caller downloads again. Every setting and entry of p comes back except
// the bundled entries, which are this computer's own. A mod's config files wait in restore/ until its folder is placed.
func (s *Store) RestoreBackup(game string, p Profile, files map[string][]byte) (Profile, []Entry, error) {
	for rel := range files {
		if !backupPath(rel, p) {
			return Profile{}, nil, fmt.Errorf("the backup holds %q, which is not a profile file", rel)
		}
	}
	all, err := s.listOK(game)
	if err != nil {
		return Profile{}, nil, err
	}
	taken := make([]string, 0, len(all))
	for _, q := range all {
		taken = append(taken, q.Name)
	}
	created, err := s.Create(game, UniqueName(taken, p.Name))
	if err != nil {
		return Profile{}, nil, err
	}
	out, missing, err := s.restoreInto(game, created, p, files)
	if err != nil {
		return Profile{}, nil, errors.Join(err, s.Delete(game, created.ID))
	}
	return out, missing, nil
}

func (s *Store) restoreInto(game string, created, p Profile, files map[string][]byte) (Profile, []Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, dir, err := s.readDir(game, created.ID)
	if err != nil {
		return Profile{}, nil, err
	}
	out := p
	out.ID, out.Name, out.Order = cur.ID, cur.Name, cur.Order
	out.Created, out.Updated = time.Now().UTC().Truncate(time.Second), time.Now().UTC().Truncate(time.Second)
	out.Error, out.RepairError = "", ""
	if _, ok := files[p.Cover]; !ok || coverType(p.Cover) == "" {
		out.Cover = ""
	}
	out.Entries = slices.DeleteFunc(slices.Clone(cur.Entries), func(e Entry) bool { return !e.Source.Bundled() })
	var missing []Entry
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		out.Entries = append(out.Entries, e)
		if _, err := s.items.Path(game, e.Key); err != nil {
			missing = append(missing, e)
		}
	}
	for rel, data := range files {
		if key, rest, ok := strings.Cut(strings.TrimPrefix(rel, "mods/"), "/"); ok && strings.HasPrefix(rel, "mods/") {
			rel = restoreDir + "/" + key + "/" + rest
		}
		to := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
			return Profile{}, nil, err
		}
		if err := fsx.WriteFile(to, data, 0o600); err != nil {
			return Profile{}, nil, err
		}
	}
	return out, missing, writeProfile(dir, out)
}

// backupPath reports whether rel is a file a backup of p may carry.
func backupPath(rel string, p Profile) bool {
	if !filepath.IsLocal(filepath.FromSlash(rel)) || strings.Contains(rel, "\\") {
		return false
	}
	if slices.Contains(backupFiles, rel) || (rel == p.Cover && coverType(rel) != "") {
		return true
	}
	if slices.ContainsFunc(backupTrees, func(t string) bool { return strings.HasPrefix(rel, t+"/") }) {
		return true
	}
	key, rest, ok := strings.Cut(strings.TrimPrefix(rel, "mods/"), "/")
	return ok && strings.HasPrefix(rel, "mods/") && rest != "" && strings.EqualFold(filepath.Ext(rest), ".json") &&
		slices.ContainsFunc(p.Entries, func(e Entry) bool { return e.Key == key })
}

// applyRestored lays the config files a restore kept for the entry key over its newly placed folder, then drops them.
func applyRestored(dir, modsDir, key string) error {
	src := filepath.Join(dir, restoreDir, key)
	if !exists(src) {
		return nil
	}
	dst := filepath.Join(modsDir, key)
	if !exists(dst) {
		dst = filepath.Join(modsDir, "."+key)
	}
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		b, err := fsx.ReadFile(path)
		if err != nil {
			return err
		}
		to := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
			return err
		}
		return fsx.WriteFile(to, b, 0o600)
	})
	if err != nil {
		return err
	}
	if err := fsx.RemoveAll(src); err != nil {
		return err
	}
	// The folder goes once the last mod it waited for is placed; until then it still holds the others.
	if left, err := os.ReadDir(filepath.Join(dir, restoreDir)); err == nil && len(left) == 0 {
		return fsx.RemoveAll(filepath.Join(dir, restoreDir))
	}
	return nil
}
