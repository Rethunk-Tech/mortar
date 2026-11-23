package profile

import (
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
