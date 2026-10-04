package profile

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestExportRestoreRoundTrip(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{
		"Pack/A/manifest.json": manifestJSON("X.A"),
		"Pack/A/hello.txt":     "from-store",
	})
	src := mustCreate(t, e, "Farm")
	var err error
	src, err = e.AddEntry("stardew", src.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetNotes("stardew", src.ID, "keep these notes"); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(e.mods(src.ID), "local-a", "Pack", "A", "config.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(cfg, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", src.ID, "local-a", "X.A", false); err != nil {
		t.Fatal(err)
	}
	img := filepath.Join(t.TempDir(), "c.png")
	if err := os.WriteFile(img, pngHeader, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetCover("stardew", src.ID, img); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "farm.zip")
	if err := e.ExportZip("stardew", src.ID, zipPath, "0.0.1"); err != nil {
		t.Fatal(err)
	}
	got, err := e.RestoreZip("stardew", zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == src.ID {
		t.Fatal("restore reused the source profile id")
	}
	if got.Name != "Farm (2)" {
		t.Fatalf("name = %q", got.Name)
	}
	if got.Notes != "keep these notes" {
		t.Fatalf("notes = %q", got.Notes)
	}
	src, err = e.read("stardew", src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if src.Name != "Farm" {
		t.Fatalf("source renamed to %q", src.Name)
	}
	assertSameUserMods(t, src, got)
	key := ""
	for _, en := range got.Entries {
		if !en.Source.Bundled() {
			key = en.Key
			break
		}
	}
	cfgText, err := e.ReadConfig("stardew", got.ID, key, "X.A")
	if err != nil || cfgText != "{\n  \"ok\": true\n}\n" {
		t.Fatalf("config = %q %v", cfgText, err)
	}
	cover, err := os.ReadFile(filepath.Join(e.root, "stardew", got.ID, got.Cover))
	if err != nil || !bytes.Equal(cover, pngHeader) {
		t.Fatalf("cover = %v %v", len(cover), err)
	}
}

func TestRestoreZipRejectsTamperedHash(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	p := mustCreate(t, e, "P")
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "ok.zip")
	if err := e.ExportZip("stardew", p.ID, src, "0.0.1"); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(t.TempDir(), "bad.zip")
	tamperZip(t, src, bad, func(name string, body []byte) []byte {
		if name == zipProfileName {
			return append(body, ' ')
		}
		return body
	})
	if _, err := e.RestoreZip("stardew", bad); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("err = %v", err)
	}
	list, err := e.List("stardew")
	if err != nil || len(list) != 1 {
		t.Fatalf("profiles = %d %v", len(list), err)
	}
}

func TestRestoreZipRejectsZipSlip(t *testing.T) {
	e := newEnv(t)
	evil := filepath.Join(t.TempDir(), "slip.zip")
	writeRawZip(t, evil, map[string][]byte{"../evil": []byte("nope")})
	if _, err := e.RestoreZip("stardew", evil); err == nil || !errors.Is(err, archive.ErrTraversal) {
		t.Fatalf("err = %v", err)
	}
	list, err := e.List("stardew")
	if err != nil || len(list) != 0 {
		t.Fatalf("profiles = %d %v", len(list), err)
	}
}

func assertSameUserMods(t *testing.T, a, b Profile) {
	t.Helper()
	type row struct {
		id, ver string
		off     bool
	}
	collect := func(p Profile) []row {
		var out []row
		for _, e := range p.Entries {
			if e.Source.Bundled() {
				continue
			}
			off := map[string]bool{}
			for _, id := range e.Disabled {
				off[strings.ToLower(id)] = true
			}
			for _, m := range e.Mods {
				out = append(out, row{strings.ToLower(m.UniqueID), m.Version, off[strings.ToLower(m.UniqueID)]})
			}
		}
		return out
	}
	left, right := collect(a), collect(b)
	if len(left) != len(right) {
		t.Fatalf("mod count %d vs %d", len(left), len(right))
	}
	for i := range left {
		if left[i] != right[i] {
			t.Fatalf("mod %d = %+v vs %+v", i, left[i], right[i])
		}
	}
}

func tamperZip(t *testing.T, src, dest string, mut func(name string, body []byte) []byte) {
	t.Helper()
	zr, err := zip.OpenReader(src)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	files := map[string][]byte{}
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[f.Name] = mut(f.Name, bytes.Clone(body))
	}
	writeRawZip(t, dest, files)
}

func writeRawZip(t *testing.T, dest string, files map[string][]byte) {
	t.Helper()
	f, err := os.CreateTemp(filepath.Dir(dest), "zip-*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := errors.Join(zw.Close(), f.Close()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(f.Name(), dest); err != nil {
		t.Fatal(err)
	}
}

func TestExportZipOmitsRunsAndHistory(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	p := mustCreate(t, e, "P")
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	dir, err := e.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "runs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "runs", "SMAPI-latest.txt"), []byte("/home/user/game"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, historyFilesDir, historyBlobsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, historyFilesDir, historyBlobsDir, "deadbeef"), []byte("cfg"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, snapshotsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, snapshotsDir, "abcd.json"), []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "p.zip")
	if err := e.ExportZip("stardew", p.ID, zipPath, "0.0.1"); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "runs/") || f.Name == historyFile || strings.HasPrefix(f.Name, historyFilesDir+"/") ||
			strings.HasPrefix(f.Name, snapshotsDir+"/") {
			t.Fatalf("export included %s", f.Name)
		}
	}
}

func TestRestoreZipIgnoresAbsoluteCover(t *testing.T) {
	e := newEnv(t)
	img := filepath.Join(t.TempDir(), "c.png")
	if err := os.WriteFile(img, pngHeader, 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(Profile{Name: "FromZip", Cover: img})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	man, err := json.Marshal(zipManifest{MortarVersion: "0.0.1", Files: map[string]string{zipProfileName: hex.EncodeToString(sum[:])}})
	if err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "abs.zip")
	writeRawZip(t, zipPath, map[string][]byte{zipProfileName: raw, zipManifestName: man})
	got, err := e.RestoreZip("stardew", zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.Cover != "" {
		t.Fatalf("absolute cover was applied: %q", got.Cover)
	}
}
