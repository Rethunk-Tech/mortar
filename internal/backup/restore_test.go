package backup

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func writeFarm(t *testing.T, saves, folder, farm, body string) {
	t.Helper()
	dir := filepath.Join(saves, folder)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	testfs.WriteFile(t, dir, folder, body)
	info := `<Farmer><name>Ann</name><farmName>` + farm + `</farmName></Farmer>`
	testfs.WriteFile(t, dir, "SaveGameInfo", info)
}

func TestSavesRecordsCauseBesideTheZip(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	writeFarm(t, saves, "Farm_1", "Sunny", "a")
	out := t.TempDir()
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	got, err := Saves(saves, out, DefaultKeep, start, Cause{Profile: "cookie", Kind: KindUpdate})
	if err != nil {
		t.Fatal(err)
	}
	listed, err := List(out)
	if err != nil || len(listed) != 1 {
		t.Fatalf("list = %v, %v", listed, err)
	}
	b := listed[0]
	if b.Name != filepath.Base(got) || b.Profile != "cookie" || b.Kind != KindUpdate {
		t.Fatalf("backup = %+v", b)
	}
	if len(b.Saves) != 1 || b.Saves[0] != (Snap{Folder: "Farm_1", Farm: "Sunny"}) {
		t.Fatalf("saves = %+v", b.Saves)
	}
	if b.Size == 0 || b.At != start.UnixMilli() {
		t.Fatalf("size/at = %d %d", b.Size, b.At)
	}
}

func TestListIsNewestFirst(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	writeFarm(t, saves, "Farm_1", "Sunny", "a")
	out := t.TempDir()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if _, err := Saves(saves, out, DefaultKeep, start, Cause{}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(saves, "Farm_1", "Farm_1"), start.Add(MinGap), start.Add(MinGap)); err != nil {
		t.Fatal(err)
	}
	if _, err := Saves(saves, out, DefaultKeep, start.Add(MinGap), Cause{Kind: KindUpdate}); err != nil {
		t.Fatal(err)
	}
	listed, err := List(out)
	if err != nil || len(listed) != 2 {
		t.Fatalf("list = %v, %v", listed, err)
	}
	if listed[0].Kind != KindUpdate || listed[1].Kind != "" {
		t.Fatalf("order kinds %q %q", listed[0].Kind, listed[1].Kind)
	}
	if listed[0].At <= listed[1].At {
		t.Fatalf("not newest first: %d %d", listed[0].At, listed[1].At)
	}
}

func TestRestoreOneLeavesTheOtherSave(t *testing.T) {
	root := t.TempDir()
	saves := filepath.Join(root, "Saves")
	backups := filepath.Join(root, "backups")
	writeFarm(t, saves, "Alpha_1", "Alpha", "old-a")
	writeFarm(t, saves, "Beta_1", "Beta", "old-b")
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	zipPath, err := Saves(saves, backups, DefaultKeep, start, Cause{})
	if err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(saves, "Alpha_1", "Alpha_1"), []byte("new-a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(saves, "Beta_1", "Beta_1"), []byte("new-b"), 0o600); err != nil {
		t.Fatal(err)
	}
	later := start.Add(MinGap)
	if err := os.Chtimes(filepath.Join(saves, "Alpha_1", "Alpha_1"), later, later); err != nil {
		t.Fatal(err)
	}
	if err := Restore(zipPath, saves, filepath.Dir(zipPath), []string{"Alpha_1"}, DefaultKeep, later); err != nil {
		t.Fatal(err)
	}
	a, err := fsx.ReadFile(filepath.Join(saves, "Alpha_1", "Alpha_1"))
	if err != nil || string(a) != "old-a" {
		t.Fatalf("alpha = %q, %v", a, err)
	}
	b, err := fsx.ReadFile(filepath.Join(saves, "Beta_1", "Beta_1"))
	if err != nil || string(b) != "new-b" {
		t.Fatalf("beta = %q, %v", b, err)
	}
}

func TestRestoreAllAndPreRestoreBackup(t *testing.T) {
	root := t.TempDir()
	saves := filepath.Join(root, "Saves")
	backups := filepath.Join(root, "backups")
	writeFarm(t, saves, "Alpha_1", "Alpha", "old-a")
	writeFarm(t, saves, "Beta_1", "Beta", "old-b")
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	zipPath, err := Saves(saves, backups, DefaultKeep, start, Cause{})
	if err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(saves, "Alpha_1", "Alpha_1"), []byte("new-a"), 0o600); err != nil {
		t.Fatal(err)
	}
	later := start.Add(MinGap)
	if err := os.Chtimes(filepath.Join(saves, "Alpha_1", "Alpha_1"), later, later); err != nil {
		t.Fatal(err)
	}
	before, err := List(backups)
	if err != nil {
		t.Fatal(err)
	}
	if err := Restore(zipPath, saves, filepath.Dir(zipPath), nil, DefaultKeep, later); err != nil {
		t.Fatal(err)
	}
	after, err := List(backups)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+1 {
		t.Fatalf("backups %d -> %d, want a pre-restore zip", len(before), len(after))
	}
	if after[0].Kind != KindRestore {
		t.Fatalf("newest kind = %q", after[0].Kind)
	}
	a, _ := fsx.ReadFile(filepath.Join(saves, "Alpha_1", "Alpha_1"))
	b, _ := fsx.ReadFile(filepath.Join(saves, "Beta_1", "Beta_1"))
	if string(a) != "old-a" || string(b) != "old-b" {
		t.Fatalf("restored %q %q", a, b)
	}
}

func TestRestoreRejectsZipSlip(t *testing.T) {
	root := t.TempDir()
	saves := filepath.Join(root, "Saves")
	if err := os.MkdirAll(saves, 0o750); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	zipPath := filepath.Join(root, "2026-06-01T00-00-00.000.zip")
	if err := writeSlipZip(zipPath); err != nil {
		t.Fatal(err)
	}
	err := Restore(zipPath, saves, filepath.Dir(zipPath), nil, DefaultKeep, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if !errors.Is(err, archive.ErrTraversal) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(outside); err == nil {
		t.Fatal("zip slip wrote outside Saves")
	}
}

func TestRestoreRejectsReservedName(t *testing.T) {
	root := t.TempDir()
	saves := filepath.Join(root, "Saves")
	if err := os.MkdirAll(saves, 0o750); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(root, "2026-06-01T00-00-00.000.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("Saves/Farm_1/CON")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(w, bytes.NewReader([]byte("nope"))); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(zipPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	err = Restore(zipPath, saves, filepath.Dir(zipPath), nil, DefaultKeep, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if !errors.Is(err, archive.ErrUnsafeName) {
		t.Fatalf("err = %v", err)
	}
}

func writeSlipZip(path string) error {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("../outside")
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, bytes.NewReader([]byte("nope"))); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return fsx.WriteFile(path, buf.Bytes(), 0o600)
}
