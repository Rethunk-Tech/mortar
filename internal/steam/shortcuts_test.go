package steam

import (
	"bufio"
	"bytes"
	"testing"
)

func TestAddShortcutAppendsOnceAndKeepsExistingEntries(t *testing.T) {
	first := Shortcut{Name: "Other game", Exe: "/usr/bin/other", StartDir: "/usr/bin"}
	body, added, err := addShortcut(nil, first)
	if err != nil || !added {
		t.Fatalf("first add: %v %v", added, err)
	}
	play := Shortcut{Name: "Main (Stardew Valley)", Exe: "/opt/mortar", StartDir: "/opt", LaunchOptions: "--play=stardew/p1"}
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
	if entry.Key != "1" || field(entry, "Exe") != `"/opt/mortar"` || field(entry, "LaunchOptions") != "--play=stardew/p1" {
		t.Fatalf("entry %+v", entry)
	}
	if entry.Child[0].Key != "appid" || entry.Child[0].Int&0x80000000 == 0 {
		t.Fatal("a non-Steam shortcut's appid has its top bit set")
	}
}
