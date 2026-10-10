package packsvc

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func dataHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("LOCALAPPDATA", dir)
}

func TestRestoreOnAnotherComputerQueuesTheModsFromTheirSources(t *testing.T) {
	dataHome(t)
	items, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	for key, id := range map[string]string{"nexus-12-34": "X.A", "local-b": "X.B"} {
		src := t.TempDir()
		if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte(`{"Name":"`+id+`","Version":"1.0.0","UniqueID":"`+id+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := items.AddDir(t.Context(), "stardew", key, src); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := profiles.AddEntry("stardew", p.ID, "nexus-12-34", profile.Source{Kind: profile.KindNexus, Name: "a.zip", ModID: 12, FileID: 34}); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AddEntry("stardew", p.ID, "local-b", profile.Source{Kind: profile.KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.SetSeparateSaves("stardew", p.ID, true, false); err != nil {
		t.Fatal(err)
	}
	saves, err := profiles.SavesFolder("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(saves, "Farm_1"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(saves, "Farm_1", "Farm_1"), []byte("save"), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "farm.zip")
	size, err := (&Service{Profiles: profiles}).BackupSize("stardew", p.ID, false)
	if err != nil || size < int64(len("save")) {
		t.Fatalf("size %d, %v", size, err)
	}
	if err := (&Service{Profiles: profiles}).Backup("stardew", p.ID, dest, false); err != nil {
		t.Fatal(err)
	}

	dataHome(t)
	freshItems, fresh := testenv.Stores(t)
	q := &fakeQueue{}
	s := &Service{Profiles: fresh, Queue: q}
	if _, err := s.Restore(context.Background(), dest, "lethal-company"); err == nil {
		t.Fatal("a backup of another game's profile must be refused")
	}
	res, err := s.Restore(context.Background(), dest, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Queued != 1 || len(q.got) != 1 || q.got[0].ModID != 12 || q.got[0].FileID != 34 || q.got[0].Profile != res.Profile {
		t.Fatalf("queued %+v (result %+v)", q.got, res)
	}
	// The archive-installed mod came back from the backup itself, so nothing is left to install by hand.
	if len(res.Unavailable) != 0 {
		t.Fatalf("unavailable %v", res.Unavailable)
	}
	all, err := fresh.List("stardew")
	if err != nil || len(all) != 1 || len(all[0].Entries) != 2 || !all[0].SeparateSaves {
		t.Fatalf("restored profiles %+v, %v", all, err)
	}
	dir, err := freshItems.Path("stardew", "local-b")
	if err != nil {
		t.Fatal(err)
	}
	if b, err := fsx.ReadFile(filepath.Join(dir, "manifest.json")); err != nil || !strings.Contains(string(b), "X.B") {
		t.Fatalf("restored store item: %q, %v", b, err)
	}
	modsDir, err := fresh.ProfileDir("stardew", res.Profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(modsDir, "mods", "local-b", "manifest.json")); err != nil {
		t.Fatalf("the restored mod was not placed: %v", err)
	}
	restored, err := fresh.SavesFolder("stardew", res.Profile)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := fsx.ReadFile(filepath.Join(restored, "Farm_1", "Farm_1")); err != nil || string(b) != "save" {
		t.Fatalf("restored save: %q, %v", b, err)
	}
}

func TestRestoreRequestPinsCurseForgeFile(t *testing.T) {
	e := profile.Entry{Source: profile.Source{Kind: profile.KindCurseForge, Name: "309243", Version: "Content Patcher 2.9.1", FileID: 555}}
	r, ok := restoreRequest("stardew", "p", e)
	if !ok || r.Source != "curseforge" || r.Package != "309243" || r.PackageFile != 555 {
		t.Errorf("request %+v ok %v", r, ok)
	}
}

func TestBackupWithModsRestoresOffline(t *testing.T) {
	dataHome(t)
	items, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte(`{"Name":"X.A","Version":"1.0.0","UniqueID":"X.A"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := items.AddDir(t.Context(), "stardew", "nexus-12-34", src); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AddEntry("stardew", p.ID, "nexus-12-34", profile.Source{Kind: profile.KindNexus, Name: "a.zip", ModID: 12, FileID: 34}); err != nil {
		t.Fatal(err)
	}
	svc := &Service{Profiles: profiles}
	with, without := filepath.Join(t.TempDir(), "with.zip"), filepath.Join(t.TempDir(), "without.zip")
	if err := svc.Backup("stardew", p.ID, with, true); err != nil {
		t.Fatal(err)
	}
	if err := svc.Backup("stardew", p.ID, without, false); err != nil {
		t.Fatal(err)
	}
	sizeWith, _ := svc.BackupSize("stardew", p.ID, true)
	sizeWithout, _ := svc.BackupSize("stardew", p.ID, false)
	if sizeWith <= sizeWithout {
		t.Fatalf("size with mods %d, without %d", sizeWith, sizeWithout)
	}

	for path, queued := range map[string]int{with: 0, without: 1} {
		dataHome(t)
		freshItems, fresh := testenv.Stores(t)
		q := &fakeQueue{}
		res, err := (&Service{Profiles: fresh, Queue: q}).Restore(context.Background(), path, "")
		if err != nil || res.Queued != queued || len(res.Unavailable) != 0 {
			t.Fatalf("%s: %+v, %v", path, res, err)
		}
		if _, err := freshItems.Path("stardew", "nexus-12-34"); (err == nil) != (queued == 0) {
			t.Fatalf("%s: store item present: %v", path, err == nil)
		}
	}
}

func TestRestoreReadsAProfileZip(t *testing.T) {
	dataHome(t)
	_, profiles := testenv.Stores(t)
	profileJSON := []byte(`{"id":"old","name":"Old farm"}`)
	sum := sha256.Sum256(profileJSON)
	manifest := `{"mortarVersion":"0.0.1","files":{"profile.json":"` + hex.EncodeToString(sum[:]) + `"}}`
	path := filepath.Join(t.TempDir(), "old.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range map[string][]byte{"profile.json": profileJSON, "mortar-export.json": []byte(manifest)} {
		w, _ := zw.Create(name)
		_, _ = w.Write(data)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Service{Profiles: profiles}
	if _, err := s.Restore(context.Background(), path, ""); err == nil {
		t.Fatal("a profile zip names no game, so Restore must ask for one")
	}
	res, err := s.Restore(context.Background(), path, "stardew")
	if err != nil || res.Name != "Old farm" || res.Game != "stardew" || res.Profile == "" {
		t.Fatalf("%+v, %v", res, err)
	}
}

func TestBackupOfAFolderGameProfileCarriesTheSplitArchiveOnceAndReplacesTray(t *testing.T) {
	dataHome(t)
	_, profiles := testenv.Stores(t)
	tray := filepath.Join(t.TempDir(), "Tray")
	profiles.TrayFolder = func(string) (string, error) { return tray, nil }
	p := testenv.Profile(t, profiles, "sims4", "Sims")
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "h.zip"), map[string]string{"a.package": "1", "b.package": "2", "Smith.trayitem": "t"})
	src := profile.Source{Kind: profile.KindCurseForge, Name: "Pack", ModID: 7, FileID: 10}
	if _, err := profiles.InstallSource(t.Context(), "sims4", p.ID, zip, src); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "s.zip")
	if err := (&Service{Profiles: profiles}).Backup("sims4", p.ID, dest, true); err != nil {
		t.Fatal(err)
	}

	dataHome(t)
	_, fresh := testenv.Stores(t)
	freshTray := filepath.Join(t.TempDir(), "Tray")
	fresh.TrayFolder = func(string) (string, error) { return freshTray, nil }
	res, err := (&Service{Profiles: fresh, Queue: &fakeQueue{}}).Restore(context.Background(), dest, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unavailable) != 0 {
		t.Fatalf("unavailable %v", res.Unavailable)
	}
	all, err := fresh.List("sims4")
	if err != nil || len(all) != 1 || len(all[0].Entries) != 3 {
		t.Fatalf("restored profiles %+v, %v", all, err)
	}
	owners, err := fresh.PackageFileOwners("sims4", res.Profile)
	if err != nil || len(owners) != 2 {
		t.Fatalf("owners = %v, %v", owners, err)
	}
	if b, err := fsx.ReadFile(filepath.Join(freshTray, "Smith.trayitem")); err != nil || string(b) != "t" {
		t.Fatalf("tray file: %q, %v", b, err)
	}
}
