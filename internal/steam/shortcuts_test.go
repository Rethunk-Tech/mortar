package steam

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"testing"
)

func TestAddShortcutAppendsOnceAndKeepsExistingEntries(t *testing.T) {
	first := Shortcut{Name: "Other game", Exe: "/usr/bin/other", StartDir: "/usr/bin"}
	body, _, added, err := addShortcut(nil, first)
	if err != nil || added != Added {
		t.Fatalf("first add: %v %v", added, err)
	}
	play := Shortcut{Name: "Main (Stardew Valley)", Exe: "/opt/mortar", StartDir: "/opt", LaunchOptions: "--play=stardew/p1 --steam-session"}
	next, _, added, err := addShortcut(body, play)
	if err != nil || added != Added {
		t.Fatalf("second add: %v %v", added, err)
	}
	if !bytes.HasPrefix(next[:len(next)-2], body[:len(body)-2]) {
		t.Fatal("the existing entry must be kept byte for byte")
	}
	if _, _, again, err := addShortcut(next, play); err != nil || again != Unchanged {
		t.Fatalf("the same command must not be added twice: %v %v", again, err)
	}
	root, err := readVDFMap(bufio.NewReader(bytes.NewReader(next)))
	if err != nil || len(root) != 1 || len(root[0].Child) != 2 {
		t.Fatalf("parsed %+v %v", root, err)
	}
	entry := root[0].Child[1]
	if entry.Key != "1" || field(entry, "Exe") != `"/opt/mortar"` || field(entry, "LaunchOptions") != "--play=stardew/p1 --steam-session" {
		t.Fatalf("entry %+v", entry)
	}
	if entry.Child[0].Key != "appid" || entry.Child[0].Int&0x80000000 == 0 {
		t.Fatal("a non-Steam shortcut's appid has its top bit set")
	}
}

func TestAddShortcutUpdatesTheProfilesEntryInPlace(t *testing.T) {
	// An entry as an earlier Mortar wrote it, then played from Steam: older options and name, playtime of its own.
	old := Shortcut{Name: "Old name", Exe: "/opt/mortar", StartDir: "/opt", LaunchOptions: "--play=stardew/p1"}.node(1)
	for i, c := range old.Child {
		if c.Key == "LastPlayTime" {
			old.Child[i].Int = 1_700_000_000
		}
	}
	other := Shortcut{Name: "P10", Exe: "/opt/mortar", StartDir: "/opt", LaunchOptions: "--play=stardew/p10 --steam-session"}.node(0)
	var fixture bytes.Buffer
	writeVDFMap(&fixture, []vdfNode{{Kind: vdfMap, Key: "shortcuts", Child: []vdfNode{other, old}}})

	sc := Shortcut{
		Name: "Main (Stardew Valley)", Exe: "/opt/mortar", StartDir: "/opt",
		LaunchOptions: "--play=stardew/p1 --steam-session", Key: "--play=stardew/p1",
	}
	next, appID, changed, err := addShortcut(fixture.Bytes(), sc)
	if err != nil || changed != Updated {
		t.Fatalf("update: %v %v", changed, err)
	}
	root, err := readVDFMap(bufio.NewReader(bytes.NewReader(next)))
	if err != nil || len(root[0].Child) != 2 {
		t.Fatalf("an update adds no entry: %+v %v", root, err)
	}
	got := root[0].Child[1]
	if field(got, "AppName") != sc.Name || field(got, "LaunchOptions") != sc.LaunchOptions {
		t.Fatalf("entry not updated: %+v", got)
	}
	if appID != appIDOf(old) || appIDOf(got) != appIDOf(old) {
		t.Fatal("the entry keeps its appid, so Steam keeps its playtime and art")
	}
	for _, c := range got.Child {
		if c.Key == "LastPlayTime" && c.Int != 1_700_000_000 {
			t.Fatal("fields Mortar does not set are kept")
		}
	}
	if field(root[0].Child[0], "AppName") != "P10" {
		t.Fatal("another profile whose id starts the same is a different entry")
	}
	if _, _, again, err := addShortcut(next, sc); err != nil || again != Unchanged {
		t.Fatalf("an up-to-date entry is left alone: %v %v", again, err)
	}
}

func TestRemoveShortcutsDropsOnlyMortarEntriesAndRenumbers(t *testing.T) {
	var body []byte
	for _, sc := range []Shortcut{
		{Name: "Other", Exe: `C:\games\other.exe`, StartDir: `C:\games`},
		{Name: "A", Exe: `C:\Mortar\mortar.exe`, StartDir: `C:\Mortar`, LaunchOptions: "--play=stardew/a --steam-session"},
		{Name: "Mortar itself", Exe: `C:\Mortar\mortar.exe`, StartDir: `C:\Mortar`},
		{Name: "B", Exe: `c:\mortar\MORTAR.exe`, StartDir: `C:\Mortar`, LaunchOptions: "--play=stardew/b --steam-session"},
		{Name: "Elsewhere", Exe: `D:\other\mortar.exe`, StartDir: `D:\other`, LaunchOptions: "--play=stardew/c --steam-session"},
	} {
		var err error
		if body, _, _, err = addShortcut(body, sc); err != nil {
			t.Fatal(err)
		}
	}
	match := func(exe, opts string) bool {
		return strings.EqualFold(exe, quoted(`C:\Mortar\mortar.exe`)) && strings.HasPrefix(opts, "--play=")
	}
	next, n, err := removeShortcuts(body, match)
	if err != nil || n != 2 {
		t.Fatalf("removed %d, %v", n, err)
	}
	root, err := readVDFMap(bufio.NewReader(bytes.NewReader(next)))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for i, e := range root[0].Child {
		if e.Key != strconv.Itoa(i) {
			t.Errorf("entry %d keyed %q", i, e.Key)
		}
		names = append(names, field(e, "AppName"))
	}
	if strings.Join(names, ",") != "Other,Mortar itself,Elsewhere" {
		t.Fatalf("kept %v", names)
	}
	if same, n, err := removeShortcuts(next, match); err != nil || n != 0 || !bytes.Equal(same, next) {
		t.Fatalf("second pass changed the file: %d %v", n, err)
	}
	if _, _, err := removeShortcuts([]byte("junk"), match); err == nil {
		t.Fatal("a malformed file must be an error")
	}
}
