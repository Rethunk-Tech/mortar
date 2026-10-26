package pack

import (
	"maps"
	"slices"
	"testing"

	_ "github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/thunderpack"
)

func TestPackFilesLandUnderBepInEx(t *testing.T) {
	got, _ := ProfileFiles(Draft{
		Loose:   []File{{Path: "BepInEx/config/a.txt", Data: []byte("a")}, {Path: "BepInEx/plugins/a.txt", Data: []byte("a")}, {Path: "BepInEx/plugins/b.dll", Data: []byte("b")}},
		Configs: []File{{Path: "config/x.cfg", Data: []byte("x")}},
	}, ImportRoots("lethal-company"))
	if string(got["BepInEx/config/x.cfg"]) != "x" || string(got["BepInEx/config/a.txt"]) != "a" || len(got) != 2 {
		t.Fatalf("files = %v", got)
	}
}

func TestPackFilesLeaveOutTheBridgeState(t *testing.T) {
	got, _ := ProfileFiles(Draft{
		Loose:   []File{{Path: "BepInEx/config/Mortar-BepInEx-Bridge-2.json", Data: []byte("{}")}, {Path: "BepInEx/config/MORTAR-bepinex-bridge.json", Data: []byte("{}")}},
		Configs: []File{{Path: "config/mortar-bepinex-bridge.json", Data: []byte("{}")}},
	}, ImportRoots("lethal-company"))
	if len(got) != 1 {
		t.Fatalf("files = %v", slices.Collect(maps.Keys(got)))
	}
}

func TestPackFilesJudgeNamesAsWindowsStoresThem(t *testing.T) {
	got, _ := ProfileFiles(Draft{Loose: []File{
		{Path: "BepInEx/plugins/evil.dll.", Data: []byte("MZ")},
		{Path: "BepInEx/plugins/evil2.dll ", Data: []byte("MZ")},
		{Path: "BepInEx/config/. ./x.txt", Data: []byte("x")},
		{Path: "BepInEx/plugins/CON", Data: []byte("x")},
		{Path: "BepInEx/config/notes.txt. ", Data: []byte("n")},
	}}, ImportRoots("lethal-company"))
	if len(got) != 1 || string(got["BepInEx/config/notes.txt"]) != "n" {
		t.Fatalf("files = %v", slices.Collect(maps.Keys(got)))
	}
}

func TestPackFilesNameOnlyTheProgramsTheyLeaveOut(t *testing.T) {
	got, programs := ProfileFiles(Draft{Loose: []File{
		{Path: "doorstop_config.ini", Data: []byte("[General]")},
		{Path: "profile.json", Data: []byte("{}")},
		{Path: "BepInEx/plugins/run.dll", Data: []byte("MZ")},
		{Path: "BepInEx/plugins/off.DLL.old", Data: []byte("MZ")},
		{Path: "BepInEx/config/readme.txt", Data: []byte("r")},
	}}, ImportRoots("lethal-company"))
	if len(got) != 1 || !slices.Equal(programs, []string{"BepInEx/plugins/off.DLL.old", "BepInEx/plugins/run.dll"}) {
		t.Fatalf("files = %v, programs = %v", slices.Collect(maps.Keys(got)), programs)
	}
}

func TestPackFilesTakeNoScriptsFromAModTree(t *testing.T) {
	for _, game := range []string{"balatro", "hades-ii"} {
		got, _ := ProfileFiles(Draft{Loose: []File{
			{Path: "mods/Evil/main.lua", Data: []byte("os.execute()")},
			{Path: "ReturnOfModding/plugins/Evil/main.lua", Data: []byte("x")},
		}}, ImportRoots(game))
		if len(got) != 0 {
			t.Fatalf("%s files = %v", game, slices.Collect(maps.Keys(got)))
		}
	}
}
