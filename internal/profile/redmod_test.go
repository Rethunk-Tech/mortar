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
		return map[string]string{dir + "/info.json": `{"name":"` + name + `"}`, dir + "/archives/a.archive": name}
	}
	for i, z := range []map[string]string{pack("First", "First"), pack("Second", "Second"), {"mods/Third/info.json": "{}", "mods/Third/archives/t.archive": "t"}} {
		if _, err := e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zipOf(t, "m.zip", z), cfSource(20+i)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := e.REDmodFolders("cyberpunk-2077", p.ID)
	if err != nil || !slices.Equal(got, []string{"Third", "Second", "First"}) {
		t.Fatalf("folders = %v, %v", got, err)
	}
	if none, err := e.REDmodFolders(folderGame, "x"); err != nil || none != nil {
		t.Fatalf("a game without REDmods: %v, %v", none, err)
	}
}
