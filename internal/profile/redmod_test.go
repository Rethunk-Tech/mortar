package profile

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/installer"
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
	dir, err := e.ProfileDir("cyberpunk-2077", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if notes := installer.ReadNotes(dir); len(notes) != 1 || notes[0].Folder != unknown.Key || notes[0].Code != installer.NoteUnrecognised {
		t.Fatalf("notes = %+v", notes)
	}
	mode = "refuse"
	zip = zipOf(t, "odd2.zip", map[string]string{"weird2/thing.dat": "x"})
	if _, err = e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zip, cfSource(41)); !errors.As(err, &unknown) || unknown.Overridable {
		t.Fatalf("refuse offers no override: %v", err)
	}
	protected := zipOf(t, "cfg.zip", map[string]string{"engine/config/giweights.json": "{}"})
	if _, err = e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, protected, cfSource(42)); !errors.As(err, &unknown) || unknown.Reason != ReasonProtectedPaths || !unknown.Overridable {
		t.Fatalf("a mod that writes protected config is asked about even where refuse stops unknown layouts: %v", err)
	}
	mode = "install"
	zip = zipOf(t, "odd3.zip", map[string]string{"weird3/thing.dat": "x"})
	if _, err = e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zip, cfSource(50)); err != nil {
		t.Fatalf("install places it as packed: %v", err)
	}
}

func TestAnArchiveModConvertsToAREDmodAndBack(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("cyberpunk-2077", "S")
	if err != nil {
		t.Fatal(err)
	}
	install := func(name string, files map[string]string, n int) string {
		t.Helper()
		res, err := e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zipOf(t, name+".zip", files), cfSource(60+n))
		if err != nil || len(res.Profile.Entries) == 0 {
			t.Fatal(err)
		}
		return res.Profile.Entries[len(res.Profile.Entries)-1].Key
	}
	pure := install("Pure Mod", map[string]string{"archive/pc/mod/pure.archive": "p"}, 0)
	mixed := install("Mixed Mod", map[string]string{"archive/pc/mod/m.archive": "m", "r6/scripts/m.reds": "r"}, 1)
	if got, err := e.REDmodChoice("cyberpunk-2077", p.ID, pure); err != nil || got != "convert" {
		t.Fatalf("choice = %q, %v", got, err)
	}
	if got, err := e.REDmodChoice("cyberpunk-2077", p.ID, mixed); err != nil || got != "" {
		t.Fatalf("a mod with more than archives is offered nothing: %q, %v", got, err)
	}
	res, err := e.ConvertToREDmod("cyberpunk-2077", p.ID, []string{pure, mixed})
	if err != nil || len(res.Skipped) != 1 || res.Skipped[0].Key != mixed {
		t.Fatalf("convert = %+v, %v", res, err)
	}
	folders, err := e.REDmodFolders("cyberpunk-2077", p.ID)
	if err != nil || len(folders) != 1 {
		t.Fatalf("folders = %v, %v", folders, err)
	}
	owners, err := e.PackageFileOwners("cyberpunk-2077", p.ID)
	if err != nil || owners["Mods/mods/"+folders[0]+"/archives/pure.archive"] != pure {
		t.Fatalf("owners = %v, %v", owners, err)
	}
	if order, _ := e.ArchiveOrder("cyberpunk-2077", p.ID); slices.Contains(order, "pure.archive") {
		t.Fatalf("a REDmod's archive is in the archive modlist: %v", order)
	}
	if got, err := e.REDmodChoice("cyberpunk-2077", p.ID, pure); err != nil || got != "revert" {
		t.Fatalf("choice after = %q, %v", got, err)
	}
	if _, err := e.RevertREDmod("cyberpunk-2077", p.ID, []string{pure}); err != nil {
		t.Fatal(err)
	}
	if folders, _ := e.REDmodFolders("cyberpunk-2077", p.ID); len(folders) != 0 {
		t.Fatalf("folders after revert = %v", folders)
	}
}

// Loose redscript files are foldered by the mod's own name, so two mods with the same file names do not collide.
func TestCyberpunkLooseScriptsAreFolderedByTheirModsName(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("cyberpunk-2077", "S")
	if err != nil {
		t.Fatal(err)
	}
	var dsts []string
	for i, name := range []string{"Mod A", "Mod B"} {
		src := cfSource(50 + i)
		src.ModName = name
		res, err := e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zipOf(t, "m.zip", map[string]string{"utils.reds": name, "other.reds": name}), src)
		if err != nil {
			t.Fatal(err)
		}
		_, l, _, err := e.layoutOf("cyberpunk-2077", p.ID, res.Profile.Entries[len(res.Profile.Entries)-1].Key, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range l.Files {
			dsts = append(dsts, f.Rel)
		}
	}
	gh := Source{Kind: KindGitHub, Name: "c.zip", Version: "1", Repo: "Me/Mod C", Tag: "v1", Asset: "c.zip"}
	key, _, err := e.StageGitHub(t.Context(), "cyberpunk-2077", gh, zipOf(t, "c.zip", map[string]string{"utils.reds": "c"}))
	if err != nil {
		t.Fatal(err)
	}
	_, l, _, err := e.layoutOf("cyberpunk-2077", p.ID, key, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range l.Files {
		dsts = append(dsts, f.Rel)
	}
	if !slices.Contains(dsts, "r6/scripts/Mod A/utils.reds") || !slices.Contains(dsts, "r6/scripts/Mod B/utils.reds") || !slices.Contains(dsts, "r6/scripts/Mod C/utils.reds") {
		t.Fatalf("laid out to %q", dsts)
	}
}

func TestConvertingKeepsAModuleNameASwitchedOffREDmodHolds(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("cyberpunk-2077", "S")
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for i := range 2 {
		src := cfSource(70 + i)
		src.ModName = "Cool Mod"
		res, err := e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zipOf(t, "c.zip", map[string]string{"archive/pc/mod/c" + string(rune('a'+i)) + ".archive": "c"}), src)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, res.Profile.Entries[len(res.Profile.Entries)-1].Key)
	}
	if _, err := e.ConvertToREDmod("cyberpunk-2077", p.ID, keys[:1]); err != nil {
		t.Fatal(err)
	}
	cur, _ := e.Get("cyberpunk-2077", p.ID)
	first := cur.Entries[slices.IndexFunc(cur.Entries, func(en Entry) bool { return en.Key == keys[0] })]
	if _, err := e.SetModEnabled("cyberpunk-2077", p.ID, first.Key, first.Mods[0].ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ConvertToREDmod("cyberpunk-2077", p.ID, keys[1:]); err != nil {
		t.Fatal(err)
	}
	cur, _ = e.Get("cyberpunk-2077", p.ID)
	var modules []string
	for _, en := range cur.Entries {
		modules = append(modules, strings.ToLower(en.REDmod))
	}
	if len(modules) != 2 || modules[0] == "" || modules[0] == modules[1] {
		t.Fatalf("modules = %q, want two different folders", modules)
	}
}

// A download installed before the layout question existed is laid out again without asking it.
func TestRepairLaysOutAnUnrecognisedDownloadWithoutTheLayoutQuestion(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("cyberpunk-2077", "S")
	if err != nil {
		t.Fatal(err)
	}
	mode := "install"
	e.UnknownLayoutMode = func(string) string { return mode }
	res, err := e.InstallSource(t.Context(), "cyberpunk-2077", p.ID, zipOf(t, "odd.zip", map[string]string{"weird/thing.dat": "x", "weird/other.dat": "y"}), cfSource(40))
	if err != nil || len(res.Profile.Entries) != 2 || res.Profile.Entries[0].File == "" {
		t.Fatalf("install = %+v, %v", res.Profile.Entries, err)
	}
	want := res.Profile.Entries[0].File
	if _, err := e.update("cyberpunk-2077", p.ID, func(p *Profile, _ string) error {
		for i := range p.Entries {
			p.Entries[i].File = "Old/" + p.Entries[i].File
			p.Entries[i].Key = p.Entries[i].Item + fileKeySep + p.Entries[i].File
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	mode = "ask"
	if changed, err := e.RepairFolderEntries("cyberpunk-2077", p.ID); err != nil || !changed {
		t.Fatalf("repair = %v, %v", changed, err)
	}
	if got, _ := e.Get("cyberpunk-2077", p.ID); got.Entries[0].File != want {
		t.Fatalf("entries = %+v, want %s laid out again", got.Entries, want)
	}
	if e.layoutOK.take("cyberpunk-2077", p.ID, res.Profile.Entries[0].Item) {
		t.Fatal("the repair left the layout allowance behind")
	}
}
