package saves

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/meta"
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
		Used: []string{"author.mod_with_under", "sonozuki.moregrass", "spacechase0.jsonassets"},
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
	if err != nil || !reflect.DeepEqual(got[0].Used, []string{"author.mod"}) {
		t.Fatalf("rescan = %+v, %v", got, err)
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
	if err != nil || !reflect.DeepEqual(got[0].Used, []string{"author.mod", "sonozuki.moregrass"}) {
		t.Fatalf("scan = %+v, %v", got, err)
	}
}

func TestFarmNameMatchesWhichFarm(t *testing.T) {
	want := []string{"Standard", "Riverland", "Forest", "Hill-top", "Wilderness", "Four Corners", "Beach", "Meadowlands"}
	for id, name := range want {
		if got := farmName(id); got != name {
			t.Errorf("farmName(%d) = %q, want %q", id, got, name)
		}
	}
	if farmName(-1) != "" || farmName(8) != "" {
		t.Fatal("unknown whichFarm must be empty")
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
	if err != nil || got[0].WhichFarm != 7 || farmName(got[0].WhichFarm) != "Meadowlands" {
		t.Fatalf("whichFarm = %+v, %v", got, err)
	}
}

func TestLacking(t *testing.T) {
	have := map[string]bool{"a.on": true, "b.off": false}
	got := Lacking([]string{"a.on", "b.off", "c.gone", "d.dismissed"}, have, []string{"D.Dismissed"})
	want := []Lack{{UniqueID: "b.off", Disabled: true}, {UniqueID: "c.gone"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lacking = %+v", got)
	}
}
