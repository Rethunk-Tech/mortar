package profile

import (
	"path/filepath"
	"slices"
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
