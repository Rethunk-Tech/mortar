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
	body, added, err := addShortcut(nil, first)
	if err != nil || !added {
		t.Fatalf("first add: %v %v", added, err)
	}
	play := Shortcut{Name: "Main (Stardew Valley)", Exe: "/opt/mortar", StartDir: "/opt", LaunchOptions: "--play=stardew/p1 --steam-session"}
	next, added, err := addShortcut(body, play)
	if err != nil || !added {
		t.Fatalf("second add: %v %v", added, err)
	}
	if !bytes.HasPrefix(next[:len(next)-2], body[:len(body)-2]) {
		t.Fatal("the existing entry must be kept byte for byte")
	}
	if _, again, err := addShortcut(next, play); err != nil || again {
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
		if body, _, err = addShortcut(body, sc); err != nil {
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
