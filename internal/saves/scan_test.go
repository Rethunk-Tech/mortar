package saves

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

var index = map[string][]meta.Ref{
	"sonozuki.moregrass":     {{Site: "Nexus", ID: 1}},
	"author.mod_with_under":  {{Site: "Nexus", ID: 2}},
	"author.mod":             {{Site: "Nexus", ID: 3}},
	"pathoschild.smapi":      {{Site: "Nexus", ID: 4}},
	"spacechase0.jsonassets": {{Site: "Nexus", ID: 5}},
}

func item(key string) string {
	return "<item><key><string>" + key + "</string></key><value><string>1</string></value></item>"
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestUniqueIDBoundaries(t *testing.T) {
	cases := map[string]string{
		"Sonozuki.MoreGrass/GrassOffsetX0":          "sonozuki.moregrass",
		"sonozuki.moregrass_thing":                  "sonozuki.moregrass",
		"Author.Mod_With_Under/setting":             "author.mod_with_under",
		"Author.Mod_Other":                          "author.mod",
		"Author.Mod_With_Under_Extra.Deep":          "author.mod_with_under",
		"smapi/mod-data/spacechase0.jsonassets/ids": "spacechase0.jsonassets",
		"smapi/mod-data/unknown.mod/x":              "",
		"Sonozuki.MoreGrassX/GrassOffset":           "",
		"Sonozuki.MoreGrass":                        "",
		"RSVDailiesDone":                            "",
		"Pathoschild.SMAPI/x":                       "pathoschild.smapi",
	}
	for key, want := range cases {
		got, ok := uniqueID(key, index)
		if got != want || ok != (want != "") {
			t.Errorf("uniqueID(%q) = %q, %v; want %q", key, got, ok, want)
		}
	}
}

func fixture(t *testing.T) (dir string) {
	t.Helper()
	dir = t.TempDir()
	save := filepath.Join(dir, "Farm_1")
	write(t, filepath.Join(save, "Farm_1"), "\xEF\xBB\xBF<SaveGame><modData>"+item("Sonozuki.MoreGrass/A")+item("Sonozuki.MoreGrass/B")+
		item("Author.Mod_With_Under/x")+item("smapi/mod-data/spacechase0.jsonassets/ids")+item("Wood")+
		"<item><key><string /></key></item></modData><whichFarm>1</whichFarm></SaveGame>")
	write(t, filepath.Join(save, "SaveGameInfo"), "\xEF\xBB\xBF<Farmer><name>Ann</name><farmName>Sunny</farmName><items><Item><name>Axe</name></Item></items>"+
		"<dayOfMonthForSaveGame>5</dayOfMonthForSaveGame><seasonForSaveGame>2</seasonForSaveGame><yearForSaveGame>3</yearForSaveGame>"+
		"<money>125300</money><millisecondsPlayed>151200000</millisecondsPlayed></Farmer>")
	write(t, filepath.Join(dir, "NotASave", "other.txt"), "x")
	return dir
}

func TestScanReadsSaveAndCachesByMtime(t *testing.T) {
	dir := fixture(t)
	s := &Scanner{Dir: dir, CacheDir: t.TempDir()}
	got, err := s.Scan(index)
	if err != nil {
		t.Fatal(err)
	}
	want := []Info{{
		Folder: "Farm_1", Farm: "Sunny", Farmer: "Ann", Season: 2, Day: 5, Year: 3, Played: got[0].Played,
		WhichFarm: 1, MillisecondsPlayed: 151200000, Money: 125300,
		Used: []mod.ID{"smapi:author.mod_with_under", "smapi:sonozuki.moregrass", "smapi:spacechase0.jsonassets"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scan = %+v, want %+v", got, want)
	}

	// A different index size rescans; an unchanged one is served from the cache.
	smaller := map[string][]meta.Ref{"sonozuki.moregrass": index["sonozuki.moregrass"]}
	if again, _ := s.Scan(smaller); len(again[0].Used) != 1 {
		t.Fatalf("index change did not rescan: %+v", again[0].Used)
	}
	if again, _ := s.Scan(smaller); len(again[0].Used) != 1 {
		t.Fatal("cache lost")
	}
	// A changed main file is rescanned.
	write(t, filepath.Join(dir, "Farm_1", "Farm_1"), "<SaveGame>"+item("Author.Mod/x")+"</SaveGame>")
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "Farm_1", "Farm_1"), later, later); err != nil {
		t.Fatal(err)
	}
	got, err = s.Scan(index)
	if err != nil || !reflect.DeepEqual(got[0].Used, []mod.ID{"smapi:author.mod"}) {
		t.Fatalf("rescan = %+v, %v", got, err)
	}
}

func TestNewestReadsOnlyTheNewestSave(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "Old_1")
	write(t, filepath.Join(old, "Old_1"), "not a save")
	newest := filepath.Join(dir, "New_1")
	write(t, filepath.Join(newest, "New_1"), "<SaveGame><modData>"+item("Author.Mod/x")+"</modData></SaveGame>")
	now := time.Now()
	if err := os.Chtimes(filepath.Join(old, "Old_1"), now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(newest, "New_1"), now, now); err != nil {
		t.Fatal(err)
	}
	got, err := (&Scanner{Dir: dir}).Newest(index)
	if err != nil || got.Folder != "New_1" || !reflect.DeepEqual(got.Used, []mod.ID{"smapi:author.mod"}) {
		t.Fatalf("newest = %+v, %v", got, err)
	}
}

func TestKeysSpanChunks(t *testing.T) {
	pad := make([]byte, 1<<20-5)
	for i := range pad {
		pad[i] = 'x'
	}
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Big_2", "Big_2"), string(pad)+item("Author.Mod/x")+string(pad)+item("Sonozuki.MoreGrass/y"))
	got, err := (&Scanner{Dir: dir}).Scan(index)
	if err != nil || !reflect.DeepEqual(got[0].Used, []mod.ID{"smapi:author.mod", "smapi:sonozuki.moregrass"}) {
		t.Fatalf("scan = %+v, %v", got, err)
	}
}

func TestScanDoesNotWriteSaves(t *testing.T) {
	dir := fixture(t)
	main := filepath.Join(dir, "Farm_1", "Farm_1")
	info := filepath.Join(dir, "Farm_1", "SaveGameInfo")
	stMain, err := os.Stat(main)
	if err != nil {
		t.Fatal(err)
	}
	stInfo, err := os.Stat(info)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&Scanner{Dir: dir, CacheDir: t.TempDir()}).Scan(index); err != nil {
		t.Fatal(err)
	}
	afterStMain, err := os.Stat(main)
	if err != nil {
		t.Fatal(err)
	}
	afterStInfo, err := os.Stat(info)
	if err != nil {
		t.Fatal(err)
	}
	if afterStMain.Size() != stMain.Size() || afterStInfo.Size() != stInfo.Size() {
		t.Fatal("scan changed save size")
	}
	if !afterStMain.ModTime().Equal(stMain.ModTime()) || !afterStInfo.ModTime().Equal(stInfo.ModTime()) {
		t.Fatal("scan touched save mtime")
	}
}

func TestWhichFarmSpansChunks(t *testing.T) {
	pad := make([]byte, 1<<20-6)
	for i := range pad {
		pad[i] = 'x'
	}
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Big_3", "Big_3"), string(pad)+"<whichFarm>7</whichFarm>"+item("Author.Mod/x"))
	got, err := (&Scanner{Dir: dir}).Scan(index)
	if err != nil || got[0].WhichFarm != 7 {
		t.Fatalf("whichFarm = %+v, %v", got, err)
	}
}

func TestWhichModFarmIsTheCustomFarmID(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Plain_1", "Plain_1"), `<whichFarm>7</whichFarm><whichModFarm>Author.Frontier_Frontier</whichModFarm>`+item("Author.Mod/x"))
	write(t, filepath.Join(dir, "Typed_2", "Typed_2"), `<whichFarm>7</whichFarm><whichModFarm><Id>Grandpa.Farm</Id><MapName>Farm_Grandpa</MapName></whichModFarm>`)
	write(t, filepath.Join(dir, "Vanilla_3", "Vanilla_3"), `<whichFarm>2</whichFarm><whichModFarm xsi:nil="true" />`)
	got, err := (&Scanner{Dir: dir}).Scan(index)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, in := range got {
		ids[in.Folder] = in.WhichModFarm
	}
	if ids["Plain_1"] != "Author.Frontier_Frontier" || ids["Typed_2"] != "Grandpa.Farm" || ids["Vanilla_3"] != "" {
		t.Fatalf("whichModFarm = %v", ids)
	}
}

func TestLacking(t *testing.T) {
	have := map[string]bool{"smapi:a.on": true, "smapi:b.off": false}
	got := Lacking([]mod.ID{"smapi:a.on", "smapi:b.off", "smapi:c.gone", "smapi:d.dismissed"}, have, []mod.ID{"smapi:D.Dismissed"})
	want := []Lack{{ID: "smapi:b.off", Disabled: true}, {ID: "smapi:c.gone"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lacking = %+v", got)
	}
}

// lethalCompany lays out a Lethal Company save folder the way the game leaves it: three slots and the challenge file
// beside its settings, its log and a mod's config folder, which are not saves.
func lethalCompany(t *testing.T) Layout {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "Lethal Company")
	for _, name := range []string{"LCSaveFile1", "LCSaveFile3", "LCChallengeFile", "LCGeneralSaveData", "Player.log", "InputUtils/binds.json"} {
		write(t, filepath.Join(dir, name), "ES3 "+name)
	}
	return Layout{Dir: dir, Files: []string{"LCSaveFile*", "LCChallengeFile"}}
}

func TestFileSavesAreFoundWithAnUnknownFit(t *testing.T) {
	l := lethalCompany(t)
	names, err := l.Names()
	if want := []string{"LCChallengeFile", "LCSaveFile1", "LCSaveFile3"}; err != nil || !reflect.DeepEqual(names, want) {
		t.Fatalf("Names = %v, %v; want %v", names, err, want)
	}
	s := &Scanner{Dir: l.Dir, Files: l.Files, CacheDir: t.TempDir()}
	infos, err := s.Scan(index)
	if err != nil || len(infos) != 3 {
		t.Fatalf("Scan = %+v, %v", infos, err)
	}
	for _, in := range infos {
		if !in.Unrecorded || in.Played == 0 || len(in.Used) != 0 {
			t.Errorf("%s = %+v, want an unrecorded save with a played time", in.Folder, in)
		}
	}
	newest, err := s.Newest(index)
	if err != nil || !l.IsSave(newest.Folder) {
		t.Fatalf("Newest = %+v, %v", newest, err)
	}
	for _, name := range []string{"LCGeneralSaveData", "Player.log", "InputUtils", "../LCSaveFile1", "LCSaveFile9"} {
		if l.IsSave(name) {
			t.Errorf("IsSave(%q) = true", name)
		}
	}
}

func TestFolderSavesStayStardewShaped(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Farm_1", "Farm_1"), "<SaveGame/>")
	write(t, filepath.Join(dir, "Notes", "readme.txt"), "x")
	write(t, filepath.Join(dir, "LCSaveFile1"), "x")
	if names, err := (Layout{Dir: dir}).Names(); err != nil || !reflect.DeepEqual(names, []string{"Farm_1"}) {
		t.Fatalf("Names = %v, %v", names, err)
	}
}
