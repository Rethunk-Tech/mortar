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
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/archive"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

const (
	zipManifestName = "mortar-export.json"
	zipProfileName  = "profile.json"
	zipModsPrefix   = "mods/"
)

type zipManifest struct {
	MortarVersion string            `json:"mortarVersion"`
	Files         map[string]string `json:"files"`
}

// ExportZip writes a zip of the profile (profile.json, notes, cover, mods as they are, configs) with per-file SHA-256.
func (s *Store) ExportZip(game, id, dest, mortarVersion string) error {
	s.mu.Lock()
	p, err := s.read(game, id)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	dir, err := s.profileDir(game, id)
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
	defer func() { _ = os.RemoveAll(snap) }()
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
		if isBundled(e) {
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
		name = path.Clean("/" + strings.ReplaceAll(name, `\`, "/"))
		name = strings.TrimPrefix(name, "/")
		if name == "" || name == ".." || strings.HasPrefix(name, "../") {
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
		if isBundled(e) {
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

// RestoreZip imports a profile zip as a new profile with a unique name. It never overwrites an existing profile.
func (s *Store) RestoreZip(game, zipPath string) (Profile, error) {
	tmp, err := os.MkdirTemp(filepath.Dir(s.root), "mortar-restore-*")
	if err != nil {
		return Profile{}, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	if err := archive.Extract(zipPath, tmp, archive.Options{}); err != nil {
		return Profile{}, err
	}
	if err := verifyExtractedZip(tmp); err != nil {
		return Profile{}, err
	}
	raw, err := fsx.ReadFile(filepath.Join(tmp, zipProfileName))
	if err != nil {
		return Profile{}, fmt.Errorf("profile.json: %w", err)
	}
	var src Profile
	if err := json.Unmarshal(raw, &src); err != nil {
		return Profile{}, fmt.Errorf("profile.json: %w", err)
	}
	all, err := s.listOK(game)
	if err != nil {
		return Profile{}, err
	}
	taken := make([]string, 0, len(all))
	for _, p := range all {
		taken = append(taken, p.Name)
	}
	created, err := s.Create(game, UniqueName(taken, src.Name))
	if err != nil {
		return Profile{}, err
	}
	ok := false
	defer func() {
		if !ok {
			_ = s.Delete(game, created.ID)
		}
	}()
	if _, err := s.SetNotes(game, created.ID, src.Notes); err != nil {
		return Profile{}, err
	}
	out, err := s.SetAppearance(game, created.ID, src.Color, src.Icon, src.Description)
	if err != nil {
		return Profile{}, err
	}
	if src.LaunchOptions != "" {
		out, err = s.SetLaunchOptions(game, created.ID, src.LaunchOptions)
		if err != nil {
			return Profile{}, err
		}
	}
	if src.Hidden {
		out, err = s.SetHidden(game, created.ID, true)
		if err != nil {
			return Profile{}, err
		}
	}
	if src.Cover != "" && filepath.IsLocal(src.Cover) {
		cover := filepath.Join(tmp, src.Cover)
		if exists(cover) {
			out, err = s.SetCover(game, created.ID, cover)
			if err != nil {
				return Profile{}, err
			}
		}
	}
	stage := filepath.Join(tmp, "stage")
	s.setHistoryQuiet(created.ID, true)
	defer s.setHistoryQuiet(created.ID, false)
	restored := 0
	for _, e := range src.Entries {
		if isBundled(e) {
			continue
		}
		srcDir := filepath.Join(tmp, "mods", e.Key)
		if !exists(srcDir) {
			return Profile{}, fmt.Errorf("missing mods/%s", e.Key)
		}
		dst := filepath.Join(stage, e.Key)
		if err := datadir.CopyTree(srcDir, dst); err != nil {
			return Profile{}, err
		}
		if err := undotEntry(stage, e); err != nil {
			return Profile{}, err
		}
		key, err := s.items.AddHashedDir(game, dst)
		if err != nil {
			return Profile{}, err
		}
		out, err = s.AddEntry(game, created.ID, key, e.Source)
		if err != nil {
			return Profile{}, err
		}
		restored++
		for _, uid := range e.Disabled {
			out, err = s.SetModEnabled(game, created.ID, key, uid, false)
			if err != nil {
				return Profile{}, err
			}
		}
		if e.Pinned {
			out, err = s.SetPinned(game, created.ID, key, true, "")
			if err != nil {
				return Profile{}, err
			}
		}
		if e.SkipVersion != "" {
			out, err = s.SetSkipVersion(game, created.ID, key, e.SkipVersion)
			if err != nil {
				return Profile{}, err
			}
		}
		for _, source := range e.SkipSources {
			out, err = s.SetSkipSource(game, created.ID, key, source, true)
			if err != nil {
				return Profile{}, err
			}
		}
		if e.Note != "" || len(e.Tags) > 0 {
			out, err = s.SetEntryNoteTags(game, created.ID, key, e.Note, e.Tags)
			if err != nil {
				return Profile{}, err
			}
		}
	}
	ok = true
	if restored > 0 {
		if err := s.recordSnapshot(game, created.ID, historyRestored, fmt.Sprintf("Restored %d mods", restored), restored); err != nil {
			return Profile{}, err
		}
	}
	return out, nil
}

func undotEntry(stage string, e Entry) error {
	for _, m := range e.Mods {
		plain, dotted, err := ModPaths(stage, e.Key, m.Folder)
		if err != nil {
			return err
		}
		if exists(dotted) && !exists(plain) {
			if err := fsx.Rename(dotted, plain); err != nil {
				return err
			}
		}
	}
	return nil
}

func verifyExtractedZip(root string) error {
	raw, err := fsx.ReadFile(filepath.Join(root, zipManifestName))
	if err != nil {
		return fmt.Errorf("%s: %w", zipManifestName, err)
	}
	var man zipManifest
	if err := json.Unmarshal(raw, &man); err != nil {
		return fmt.Errorf("%s: %w", zipManifestName, err)
	}
	if man.Files == nil {
		return fmt.Errorf("missing file hashes")
	}
	seen := map[string]struct{}{}
	err = filepath.WalkDir(root, func(fp string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, fp)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if name == zipManifestName || strings.HasPrefix(name, "stage/") || strings.HasPrefix(name, "work-") {
			return nil
		}
		sum, ok := man.Files[name]
		if !ok {
			return fmt.Errorf("unlisted file %s", name)
		}
		b, err := fsx.ReadFile(fp)
		if err != nil {
			return err
		}
		got := sha256.Sum256(b)
		if !strings.EqualFold(hex.EncodeToString(got[:]), sum) {
			return fmt.Errorf("hash mismatch for %s", name)
		}
		seen[name] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}
	for name := range man.Files {
		if _, ok := seen[name]; !ok {
			return fmt.Errorf("missing %s", name)
		}
	}
	return nil
}
