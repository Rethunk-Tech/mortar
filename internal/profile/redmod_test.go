package profile

import (
	"errors"
	"slices"
	"testing"
)

func TestREDmodFoldersLeadWithTheEntryThatWins(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("cyberpunk-2077", "S")
	if err != nil {
		t.Fatal(err)
	}
	pack := func(name, dir string) map[string]string {
		return map[string]string{dir + "/info.json": `{"name":"` + name + `","version":"1"}`, dir + "/archives/a.archive": name}
	}
	for i, z := range []map[string]string{pack("First", "First"), pack("Second", "Second"), {"mods/Third/info.json": `{"name":"Third","version":"1"}`, "mods/Third/archives/t.archive": "t"}} {
		if _, err := e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zipOf(t, "m.zip", z), cfSource(20+i)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := e.REDmodFolders("cyberpunk-2077", p.ID)
	if err != nil || !slices.Equal(got, []string{"Third", "Second", "First"}) {
		t.Fatalf("folders = %v, %v", got, err)
	}
	if _, err := e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zipOf(t, "a.zip", map[string]string{"archive/pc/mod/z.archive": "z", "archive/pc/mod/m.archive": "m"}), cfSource(30)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zipOf(t, "b.zip", map[string]string{"archive/pc/mod/a.archive": "a"}), cfSource(31)); err != nil {
		t.Fatal(err)
	}
	if order, err := e.ArchiveOrder("cyberpunk-2077", p.ID); err != nil || !slices.Equal(order, []string{"m.archive", "z.archive", "a.archive"}) {
		t.Fatalf("archive order = %v, %v", order, err)
	}
	if none, err := e.REDmodFolders(folderGame, "x"); err != nil || none != nil {
		t.Fatalf("a game without REDmods: %v, %v", none, err)
	}
}

func TestAnUnrecognisedLayoutFollowsTheUnknownLayoutSetting(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("cyberpunk-2077", "S")
	if err != nil {
		t.Fatal(err)
	}
	mode := "ask"
	e.UnknownLayoutMode = func(string) string { return mode }
	zip, src := zipOf(t, "odd.zip", map[string]string{"weird/thing.dat": "x", "weird/other.dat": "y"}), cfSource(40)
	_, err = e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zip, src)
	var unknown *UnknownLayoutError
	if !errors.As(err, &unknown) || !unknown.Overridable || len(unknown.Files) != 2 {
		t.Fatalf("ask stops with the files: %v", err)
	}
	e.AllowUnknownLayout("cyberpunk-2077", p.ID, unknown.Key)
	if _, err := e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zip, src); err != nil {
		t.Fatalf("an allowed install places it as packed: %v", err)
	}
	mode = "refuse"
	zip = zipOf(t, "odd2.zip", map[string]string{"weird2/thing.dat": "x"})
	if _, err = e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zip, cfSource(41)); !errors.As(err, &unknown) || unknown.Overridable {
		t.Fatalf("refuse offers no override: %v", err)
	}
	mode = "install"
	zip = zipOf(t, "odd3.zip", map[string]string{"weird3/thing.dat": "x"})
	if _, err = e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zip, cfSource(50)); err != nil {
		t.Fatalf("install places it as packed: %v", err)
	}
}
