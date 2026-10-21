package pack

import (
	"maps"
	"slices"
	"testing"

	_ "github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
)

func TestPackFilesLandUnderBepInEx(t *testing.T) {
	got, _ := ProfileFiles(Draft{
		Loose:   []File{{Path: "BepInEx/plugins/a.txt", Data: []byte("a")}, {Path: "BepInEx/plugins/b.dll", Data: []byte("b")}},
		Configs: []File{{Path: "config/x.cfg", Data: []byte("x")}},
	}, ImportRoots("lethal-company"))
	if string(got["BepInEx/config/x.cfg"]) != "x" || string(got["BepInEx/plugins/a.txt"]) != "a" || len(got) != 2 {
		t.Fatalf("files = %v", got)
	}
}

func TestPackFilesJudgeNamesAsWindowsStoresThem(t *testing.T) {
	got, _ := ProfileFiles(Draft{Loose: []File{
		{Path: "BepInEx/plugins/evil.dll.", Data: []byte("MZ")},
		{Path: "BepInEx/plugins/evil2.dll ", Data: []byte("MZ")},
		{Path: "BepInEx/plugins/. ./x.txt", Data: []byte("x")},
		{Path: "BepInEx/plugins/CON", Data: []byte("x")},
		{Path: "BepInEx/plugins/notes.txt. ", Data: []byte("n")},
	}}, ImportRoots("lethal-company"))
	if len(got) != 1 || string(got["BepInEx/plugins/notes.txt"]) != "n" {
		t.Fatalf("files = %v", slices.Collect(maps.Keys(got)))
	}
}

func TestPackFilesNameOnlyTheProgramsTheyLeaveOut(t *testing.T) {
	got, programs := ProfileFiles(Draft{Loose: []File{
		{Path: "doorstop_config.ini", Data: []byte("[General]")},
		{Path: "profile.json", Data: []byte("{}")},
		{Path: "BepInEx/plugins/run.dll", Data: []byte("MZ")},
		{Path: "BepInEx/plugins/readme.txt", Data: []byte("r")},
	}}, ImportRoots("lethal-company"))
	if len(got) != 1 || !slices.Equal(programs, []string{"BepInEx/plugins/run.dll"}) {
		t.Fatalf("files = %v, programs = %v", slices.Collect(maps.Keys(got)), programs)
	}
}
