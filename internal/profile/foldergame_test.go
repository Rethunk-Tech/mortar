package profile

import (
	"errors"
	"fmt"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const folderGame = "sims4"

func zipOf(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	return testfs.WriteZip(t, filepath.Join(t.TempDir(), name), files)
}

func keysOf(p Profile) []string {
	var out []string
	for _, e := range p.Entries {
		out = append(out, e.Key)
	}
	slices.Sort(out)
	return out
}

func cfSource(fileID int) Source {
	return Source{Kind: KindCurseForge, Name: "Pack", ModID: 7, FileID: fileID}
}

func TestFolderGameSplitsAScriptFreeArchivePerFile(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create(folderGame, "S")
	if err != nil {
		t.Fatal(err)
	}
	zip := zipOf(t, "a.zip", map[string]string{"A/one.package": "1", "two.package": "2", "three.package": "3"})
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Profile.Entries) != 3 {
		t.Fatalf("entries = %v", keysOf(res.Profile))
	}
	for _, en := range res.Profile.Entries {
		if en.Source.FileID != 10 || en.Item == "" || en.Key != en.Item+"#"+en.File {
			t.Fatalf("entry = %+v", en)
		}
	}
	owners, err := e.PackageFileOwners(folderGame, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(owners) != 3 || owners["Mods/A/one.package"] == "" {
		t.Fatalf("owners = %v", owners)
	}
}

func TestFolderGameKeepsAScriptArchiveWhole(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "m.zip", map[string]string{"m.ts4script": "s", "a.package": "1", "b.package": "2"})
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(11))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Profile.Entries) != 1 || res.Profile.Entries[0].File != "" {
		t.Fatalf("entries = %v", keysOf(res.Profile))
	}
	owners, _ := e.PackageFileOwners(folderGame, p.ID)
	if len(owners) != 3 {
		t.Fatalf("owners = %v", owners)
	}
}

func TestFolderGameUpdateReplacesTheArchiveAndRemovalTakesOneFile(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	v1 := zipOf(t, "a.zip", map[string]string{"one.package": "1", "two.package": "2"})
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, v1, cfSource(10)); err != nil {
		t.Fatal(err)
	}
	other := zipOf(t, "o.zip", map[string]string{"x.package": "x"})
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, other, Source{Kind: KindCurseForge, Name: "Other", ModID: 8, FileID: 1}); err != nil {
		t.Fatal(err)
	}
	cur, _ := e.Get(folderGame, p.ID)
	for _, en := range cur.Entries {
		if en.File == "two.package" {
			if _, err := e.SetModEnabled(folderGame, p.ID, en.Key, en.Mods[0].ID, false); err != nil {
				t.Fatal(err)
			}
		}
	}
	v2 := zipOf(t, "b.zip", map[string]string{"one.package": "1b", "three.package": "3"})
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, v2, cfSource(12).WithReplacing(10))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || len(res.Profile.Entries) != 3 {
		t.Fatalf("updated = %v, entries = %v", res.Updated, keysOf(res.Profile))
	}
	for _, en := range res.Profile.Entries {
		if en.Source.ModID == 7 && en.Source.FileID != 12 {
			t.Fatalf("old file's entry left: %+v", en)
		}
	}
	var drop string
	for _, en := range res.Profile.Entries {
		if en.File == "three.package" {
			drop = en.Key
		}
	}
	after, err := e.RemoveEntries(folderGame, p.ID, []string{drop})
	if err != nil {
		t.Fatal(err)
	}
	owners, _ := e.PackageFileOwners(folderGame, p.ID)
	if len(after.Entries) != 2 || len(owners) != 2 || owners["Mods/three.package"] != "" {
		t.Fatalf("entries = %v, owners = %v", keysOf(after), owners)
	}
}

func TestFolderGameAnotherFileOfThePageIsAddedAndOnlyItsUpdateReplacesIt(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	nx := func(file int) Source { return Source{Kind: KindNexus, Name: "f.zip", ModID: 5, FileID: file} }
	main := zipOf(t, "m.zip", map[string]string{"main.package": "m"})
	opt := zipOf(t, "o.zip", map[string]string{"opt.package": "o"})
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, main, nx(1)); err != nil {
		t.Fatal(err)
	}
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, opt, nx(2))
	if err != nil || res.Updated || len(res.Profile.Entries) != 2 {
		t.Fatalf("optional beside main: %v, %v", keysOf(res.Profile), err)
	}
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, opt, nx(2)); err == nil {
		t.Fatal("the same file twice must be refused")
	}
	opt2 := zipOf(t, "o2.zip", map[string]string{"opt.package": "o2", "extra.package": "x"})
	res, err = e.InstallSource(t.Context(), folderGame, p.ID, opt2, nx(3).WithReplacing(2))
	if err != nil || !res.Updated || len(res.Profile.Entries) != 3 {
		t.Fatalf("optional update: %v, %v", keysOf(res.Profile), err)
	}
	for _, en := range res.Profile.Entries {
		if en.File == "main.package" && en.Source.FileID != 1 {
			t.Fatalf("main replaced: %+v", en)
		}
	}
}

func TestANewProfileFollowsTheGamesSeparateSavesDefault(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	for game, want := range map[string]bool{folderGame: true, "stardew": false} {
		p, err := e.Create(game, "P")
		if err != nil {
			t.Fatal(err)
		}
		if p.SeparateSaves != want {
			t.Fatalf("%s: SeparateSaves = %v, want %v", game, p.SeparateSaves, want)
		}
		if got, _ := e.Get(game, p.ID); got.SeparateSaves != want {
			t.Fatalf("%s: stored SeparateSaves = %v", game, got.SeparateSaves)
		}
	}
}

func TestFolderGameRollBackRestoresTheSupersededArchiveAndItsSwitches(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	v1 := zipOf(t, "a.zip", map[string]string{"one.package": "1", "two.package": "2"})
	first, err := e.InstallSource(t.Context(), folderGame, p.ID, v1, cfSource(10))
	if err != nil {
		t.Fatal(err)
	}
	for _, en := range first.Profile.Entries {
		if en.File == "two.package" {
			if _, err := e.SetModEnabled(folderGame, p.ID, en.Key, en.Mods[0].ID, false); err != nil {
				t.Fatal(err)
			}
		}
	}
	oldKeys := keysOf(first.Profile)
	v2 := zipOf(t, "b.zip", map[string]string{"three.package": "3"})
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, v2, cfSource(12).WithReplacing(10))
	if err != nil || len(res.Profile.Entries) != 1 {
		t.Fatalf("update: %v, %v", keysOf(res.Profile), err)
	}
	back, err := e.RollBack(folderGame, p.ID, res.Profile.Entries[0].Key)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(keysOf(back), oldKeys) {
		t.Fatalf("rolled back to %v, want %v", keysOf(back), oldKeys)
	}
	for _, en := range back.Entries {
		if off := !en.hasPackageEnabled(); off != (en.File == "two.package") || en.Source.FileID != 10 {
			t.Fatalf("entry %+v", en)
		}
	}
	fwd, err := e.RollBack(folderGame, p.ID, back.Entries[0].Key)
	if err != nil || len(fwd.Entries) != 1 || fwd.Entries[0].File != "three.package" {
		t.Fatalf("roll forward: %v, %v", keysOf(fwd), err)
	}
}

func TestFolderGameBackupRoundTripKeepsPerFileEntriesWithTheirItem(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "a.zip", map[string]string{"one.package": "1", "two.package": "2"})
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10))
	if err != nil {
		t.Fatal(err)
	}
	dirs, err := e.BackupDirs(folderGame, res.Profile, func(Entry) bool { return true })
	if err != nil || len(dirs) != 1 {
		t.Fatalf("one store item once: %v, %v", dirs, err)
	}
	other := newEnv(t)
	back, missing, err := other.RestoreBackup(t.Context(), folderGame, res.Profile, nil, dirs)
	if err != nil || len(missing) != 0 || len(back.Entries) != 2 {
		t.Fatalf("restore: entries %v, missing %v, %v", keysOf(back), missing, err)
	}
	owners, err := other.PackageFileOwners(folderGame, back.ID)
	if err != nil || len(owners) != 2 {
		t.Fatalf("owners = %v, %v", owners, err)
	}
}

func TestFolderGameRefusesTwoFilesThatDifferOnlyByCase(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "a.zip", map[string]string{"A.package": "1", "a.package": "2"})
	_, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10))
	if !errors.Is(err, archive.ErrCaseCollision) || !strings.Contains(err.Error(), "package") {
		t.Fatalf("err = %v", err)
	}
}

func TestFolderGameAnotherArchiveLayingOutTheSamePathIsCountedAsAnOverride(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	for i, src := range []Source{cfSource(1), {Kind: KindCurseForge, Name: "Other", ModID: 8, FileID: 1}} {
		zip := zipOf(t, "a.zip", map[string]string{"same.package": fmt.Sprint(i)})
		if _, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, src); err != nil {
			t.Fatal(err)
		}
	}
	wins, err := e.PackageOverrides(folderGame, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, n := range wins {
		total += n
	}
	if total != 1 {
		t.Fatalf("overrides = %v", wins)
	}
}

func TestFolderGameLoadOrderListsAnArchiveOnce(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "a.zip", map[string]string{"one.package": "1", "two.package": "2"})
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10))
	if err != nil {
		t.Fatal(err)
	}
	dirs, err := e.enabledFolders(folderGame, "", res.Profile)
	if err != nil || len(dirs) != 1 {
		t.Fatalf("dirs = %v, %v", dirs, err)
	}
}

func trayTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, _ error) error {
		if d != nil && d.Type().IsRegular() {
			b, _ := os.ReadFile(p)
			rel, _ := filepath.Rel(dir, p)
			out[rel] = string(b)
		}
		return nil
	})
	return out
}

func TestFolderGameTrayFilesInstallAndUninstallBesideThePlayersOwn(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	tray := filepath.Join(t.TempDir(), "Tray")
	e.TrayFolder = func(string) (string, error) { return tray, nil }
	if err := os.MkdirAll(tray, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tray, "mine.trayitem"), []byte("my household"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := trayTree(t, tray)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "h.zip", map[string]string{"Smith.trayitem": "t", "Smith.bpi": "b", "outfit.package": "p"})
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10))
	if err != nil {
		t.Fatal(err)
	}
	if got := trayTree(t, tray); len(got) != 3 || got["Smith.trayitem"] != "t" || got["mine.trayitem"] != "my household" {
		t.Fatalf("tray after install = %v", got)
	}
	owners, _ := e.PackageFileOwners(folderGame, p.ID)
	if len(owners) != 1 || owners["Mods/outfit.package"] == "" {
		t.Fatalf("tray files must not be laid out per profile: %v", owners)
	}
	var trayKey string
	for _, en := range res.Profile.Entries {
		if len(en.TrayFiles) > 0 {
			trayKey = en.Key
		}
	}
	if trayKey == "" {
		t.Fatalf("no entry holds the tray files: %v", keysOf(res.Profile))
	}
	if _, err := e.RemoveEntries(folderGame, p.ID, []string{trayKey}); err != nil {
		t.Fatal(err)
	}
	if got := trayTree(t, tray); !maps.Equal(got, before) {
		t.Fatalf("tray after uninstall = %v, want %v", got, before)
	}
}

func TestFolderGameTrayNeverReplacesAFileOfThePlayers(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	tray := filepath.Join(t.TempDir(), "Tray")
	e.TrayFolder = func(string) (string, error) { return tray, nil }
	if err := os.MkdirAll(tray, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tray, "Smith.trayitem"), []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "h.zip", map[string]string{"Smith.trayitem": "theirs", "Smith.bpi": "b"})
	_, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10))
	if err == nil || !strings.Contains(err.Error(), "Smith.trayitem") {
		t.Fatalf("err = %v", err)
	}
	got := trayTree(t, tray)
	cur, _ := e.Get(folderGame, p.ID)
	if len(got) != 1 || got["Smith.trayitem"] != "mine" || len(cur.Entries) != 0 {
		t.Fatalf("tray = %v, entries = %v", got, keysOf(cur))
	}
}

func trayEnv(t *testing.T) (env, string) {
	t.Helper()
	e := newEnv(t)
	tray := filepath.Join(t.TempDir(), "Tray")
	e.TrayFolder = func(string) (string, error) { return tray, nil }
	if err := os.MkdirAll(tray, 0o750); err != nil {
		t.Fatal(err)
	}
	return e, tray
}

func TestFolderGameDeletingAProfileReleasesItsTrayFilesAndRestoreReplacesThem(t *testing.T) {
	t.Parallel()
	e, tray := trayEnv(t)
	if err := os.WriteFile(filepath.Join(tray, "mine.trayitem"), []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	p1, _ := e.Create(folderGame, "A")
	p2, _ := e.Create(folderGame, "B")
	zip := zipOf(t, "h.zip", map[string]string{"Smith.trayitem": "t", "x.package": "p"})
	for _, p := range []Profile{p1, p2} {
		if _, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10)); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.Delete(folderGame, p1.ID); err != nil {
		t.Fatal(err)
	}
	if got := trayTree(t, tray); got["Smith.trayitem"] != "t" {
		t.Fatalf("another profile still holds the file: %v", got)
	}
	if err := e.Delete(folderGame, p2.ID); err != nil {
		t.Fatal(err)
	}
	if got := trayTree(t, tray); len(got) != 1 || got["mine.trayitem"] != "mine" {
		t.Fatalf("tray after both deletes = %v", got)
	}
	if _, err := e.Restore(folderGame, p2.ID); err != nil {
		t.Fatal(err)
	}
	if got := trayTree(t, tray); got["Smith.trayitem"] != "t" || got["mine.trayitem"] != "mine" {
		t.Fatalf("tray after restore = %v", got)
	}
	if err := e.Delete(folderGame, p2.ID); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tray, "Smith.trayitem"), []byte("someone else's"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := e.Restore(folderGame, p2.ID)
	var tre *TrayRestoreError
	if !errors.As(err, &tre) || !slices.Equal(tre.Files, []string{"Smith.trayitem"}) {
		t.Fatalf("restore err = %v", err)
	}
	if got := trayTree(t, tray); got["Smith.trayitem"] != "someone else's" {
		t.Fatalf("a file of the player's own was replaced: %v", got)
	}
}

func TestFolderGameTrayFollowsAnUpdateAndItsRollback(t *testing.T) {
	t.Parallel()
	e, tray := trayEnv(t)
	p, _ := e.Create(folderGame, "S")
	v1 := zipOf(t, "a.zip", map[string]string{"A.trayitem": "1", "m.package": "m"})
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, v1, cfSource(10)); err != nil {
		t.Fatal(err)
	}
	v2 := zipOf(t, "b.zip", map[string]string{"A.trayitem": "2", "B.bpi": "b", "m.package": "m2"})
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, v2, cfSource(11).WithReplacing(10))
	if err != nil {
		t.Fatal(err)
	}
	if got := trayTree(t, tray); len(got) != 2 || got["A.trayitem"] != "2" || got["B.bpi"] != "b" {
		t.Fatalf("after update: %v", got)
	}
	back, err := e.RollBack(folderGame, p.ID, res.Profile.Entries[0].Key)
	if err != nil {
		t.Fatal(err)
	}
	if got := trayTree(t, tray); len(got) != 1 || got["A.trayitem"] != "1" {
		t.Fatalf("after rollback: %v", got)
	}
	if _, err := e.RollBack(folderGame, p.ID, back.Entries[0].Key); err != nil {
		t.Fatal(err)
	}
	if got := trayTree(t, tray); len(got) != 2 || got["A.trayitem"] != "2" || got["B.bpi"] != "b" {
		t.Fatalf("after rolling forward: %v", got)
	}
}

func TestFolderGameFailedUpdateLeavesTheOldTrayFilesAndEntries(t *testing.T) {
	t.Parallel()
	e, tray := trayEnv(t)
	p, _ := e.Create(folderGame, "S")
	v1 := zipOf(t, "a.zip", map[string]string{"A.trayitem": "1", "m.package": "m"})
	first, err := e.InstallSource(t.Context(), folderGame, p.ID, v1, cfSource(10))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tray, "C.trayitem"), []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	v2 := zipOf(t, "b.zip", map[string]string{"A.trayitem": "2", "C.trayitem": "theirs"})
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, v2, cfSource(11).WithReplacing(10)); err == nil {
		t.Fatal("a collision with the player's file must refuse the update")
	}
	cur, _ := e.Get(folderGame, p.ID)
	if got := trayTree(t, tray); len(got) != 2 || got["A.trayitem"] != "1" || got["C.trayitem"] != "mine" {
		t.Fatalf("tray = %v", got)
	}
	if !slices.Equal(keysOf(cur), keysOf(first.Profile)) {
		t.Fatalf("entries = %v, want %v", keysOf(cur), keysOf(first.Profile))
	}
}
