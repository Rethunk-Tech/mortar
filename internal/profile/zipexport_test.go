package profile

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// The profile zip has no production writer: Backup is the one export. These write the old format for the reader tests.

// ExportZip writes a zip of the profile (profile.json, notes, cover, mods as they are, configs) with per-file SHA-256.
func (s *Store) ExportZip(game, id, dest, mortarVersion string) error {
	s.mu.Lock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	p.Entries = cloneEntries(p.Entries)
	s.mu.Unlock()
	snap, err := os.MkdirTemp(filepath.Dir(dir), "mortar-export-*")
	if err != nil {
		return err
	}
	err = snapshotProfileExport(dir, snap, p)
	defer func() { _ = fsx.RemoveAll(snap) }()
	if err == nil {
		err = s.snapshotOverlays(game, id, snap, p)
	}
	if err != nil {
		return err
	}
	if err := datadir.WriteStream(dest, 0o600, func(w io.Writer) error {
		return writeProfileZip(w, mortarVersion, snap, p)
	}); err != nil {
		return err
	}
	return nil
}

func zipModName(rel string, skip map[string]bool) (string, bool) {
	rest := strings.TrimPrefix(rel, zipModsPrefix)
	top, more, ok := strings.Cut(rest, "/")
	key := strings.TrimPrefix(top, ".")
	if skip[key] {
		return "", false
	}
	if !ok {
		return zipModsPrefix + key, true
	}
	return zipModsPrefix + key + "/" + more, true
}

func omitFromProfileZip(rel string) bool {
	return rel == historyFile || rel == "runs" || rel == historyFilesDir || rel == snapshotsDir ||
		strings.HasPrefix(rel, "runs/") || strings.HasPrefix(rel, historyFilesDir+"/") ||
		strings.HasPrefix(rel, snapshotsDir+"/")
}

func snapshotProfileExport(src, dst string, p Profile) error {
	skip := map[string]bool{}
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			skip[e.Key] = true
		}
	}
	return filepath.WalkDir(src, func(fp string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, fp)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if omitFromProfileZip(rel) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(rel, zipModsPrefix) {
			_, keep := zipModName(rel, skip)
			if !keep {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		dest := filepath.Join(dst, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(dest, 0o700)
		}
		b, err := fsx.ReadFile(fp)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return err
		}
		return fsx.WriteFile(dest, b, 0o600)
	})
}

func writeProfileZip(w io.Writer, mortarVersion, profileDir string, p Profile) error {
	zw := zip.NewWriter(w)
	hashes := map[string]string{}
	add := func(name string, r io.Reader) error {
		name = store.CleanRoot(name)
		if name == "" {
			return fmt.Errorf("invalid zip path %q", name)
		}
		h := sha256.New()
		fh := &zip.FileHeader{Name: name, Method: zip.Deflate}
		fh.Modified = time.Now()
		fw, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}
		if _, err := io.Copy(io.MultiWriter(fw, h), r); err != nil {
			return err
		}
		if name != zipManifestName {
			hashes[name] = hex.EncodeToString(h.Sum(nil))
		}
		return nil
	}
	skip := map[string]bool{}
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			skip[e.Key] = true
		}
	}
	if err := filepath.WalkDir(profileDir, func(fp string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(profileDir, fp)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if omitFromProfileZip(rel) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if rel == zipManifestName {
			return nil
		}
		if strings.HasPrefix(rel, zipModsPrefix) {
			var keep bool
			rel, keep = zipModName(rel, skip)
			if !keep {
				return nil
			}
		}
		b, err := fsx.ReadFile(fp)
		if err != nil {
			return err
		}
		if err := add(rel, bytes.NewReader(b)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		_ = zw.Close()
		return err
	}
	mb, err := json.Marshal(zipManifest{MortarVersion: mortarVersion, Files: hashes})
	if err != nil {
		_ = zw.Close()
		return err
	}
	if err := add(zipManifestName, strings.NewReader(string(mb))); err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}

// snapshotOverlays takes the optional files back off each main entry's folder in the export snapshot, so it holds the
// main file as installed, and copies each optional file's store item beside it.
func (s *Store) snapshotOverlays(game, id, snap string, p Profile) error {
	modsDir := filepath.Join(snap, "mods")
	for _, e := range p.Entries {
		if e.IsOverlay() {
			dir, err := s.items.Path(game, e.Key)
			if err != nil {
				return err
			}
			if err := datadir.CopyTree(dir, filepath.Join(snap, zipOverlaysDir, e.Key)); err != nil {
				return err
			}
			continue
		}
		if was := overlaysOf(p.Entries, e.Key); len(was) > 0 {
			if err := s.layOverlays(game, id, e, liveEntryDir(modsDir, e.Key), was, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
