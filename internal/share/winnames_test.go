package share

import (
	"bytes"
	"path/filepath"
	"slices"
	"testing"
)

var winNames = []string{"CONIN$.ini", "CONOUT$.ini", "COM¹.ini", "lpt³.txt", "con .json"}

func TestWindowsRefusedNamesAreNotWritten(t *testing.T) {
	t.Parallel()
	for _, n := range winNames {
		if ValidConfigPath(n[:len(n)-len(filepath.Ext(n))] + ".json") {
			t.Errorf("ValidConfigPath accepted %q", n)
		}
		if validChangedPath("sims4", "Mods/x/"+n) {
			t.Errorf("validChangedPath accepted %q", n)
		}
		if validLoaderConfigPath("config/" + n[:len(n)-len(filepath.Ext(n))] + ".cfg") {
			t.Errorf("validLoaderConfigPath accepted %q", n)
		}
	}
	dir, p := changedProfile(t)
	writeIn(t, dir, "changed/Mods/x/CONIN$.ini", []byte("x"))
	var buf bytes.Buffer
	skipped, err := Write(&buf, "sims4", p, filepath.Join(dir, "mods"), OwnInclude())
	if err != nil || !slices.Contains(skipped, "changed/Mods/x/CONIN$.ini") {
		t.Fatalf("skipped %v, %v", skipped, err)
	}
	if pv, err := ReadBytes(buf.Bytes()); err != nil || len(pv.ChangedFiles) != 2 {
		t.Fatalf("%d changed files, %v", len(pv.ChangedFiles), err)
	}
}

func TestReadSkipsWindowsRefusedNames(t *testing.T) {
	t.Parallel()
	head := [2]string{"profile.json", `{"version":3,"game":"sims4","sourceKeys":{},"name":"x","notes":"","entries":[],"ids":["smapi:A.one"]}`}
	pv, err := Read(zipOf(t, head,
		[2]string{"changed/Mods/x/CONIN$.ini", "x"},
		[2]string{"configs/smapi/A.one/con .json", "{}"},
		[2]string{"configs/smapi/A.one/ok.json", "{}"},
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(pv.ChangedFiles) != 0 || len(pv.Configs) != 1 || pv.Configs[0].Path != "ok.json" {
		t.Errorf("changed %v, configs %v", pv.ChangedFiles, pv.Configs)
	}
}
