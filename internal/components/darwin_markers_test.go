package components

import "testing"

// Each value is the macOS launch executable Steam's app info gives the game (api.steamcmd.net/v1/info/<appId>,
// config.launch entry with oslist macos), below the install folder: [app id, darwinMarker].
var steamMacExecutables = map[string][2]string{
	"stardew":                           {"413150", "Contents/MacOS/StardewValley"},
	"muck":                              {"1625450", "Muck.app/Contents/MacOS/Muck"},
	"skul-the-hero-slayer":              {"1147560", "Skul.app/Contents/MacOS/Skul"},
	"totally-accurate-battle-simulator": {"508440", "TotallyAccurateBattleSimulator.app/Contents/MacOS/Totally Accurate Battle Simulator"},
	"project-gorgon":                    {"342940", "MacPlayer.app/Contents/MacOS/Project Gorgon"},
	"cuphead":                           {"268910", "Cuphead.app/Contents/MacOS/Cuphead"},
	"my-dream-setup":                    {"2200780", "MDS.app/Contents/MacOS/MDS"},
	"necropolis":                        {"384490", "Necropolis.app/Contents/MacOS/Necropolis"},
	"paralives":                         {"1118520", "Paralives.app/Contents/MacOS/Paralives"},
	"pikuniku":                          {"572890", "Pikuniku.app/Contents/MacOS/Pikuniku"},
	"ravenfield":                        {"636480", "ravenfield.app/Contents/MacOS/RavenfieldSteam"},
	"west-of-loathing":                  {"597220", "West of Loathing.app/Contents/MacOS/West of Loathing"},
	"stick-fight-the-game":              {"674940", "StickFight.app/Contents/MacOS/StickFight"},
	"timberborn":                        {"1062090", "Timberborn.app/Contents/MacOS/Timberborn"},
	"crab-game":                         {"1782210", "Crab Game.app/Contents/MacOS/Crab Game"},
	"magicite":                          {"268750", "Magicite.app/Contents/MacOS/Magicite"},
	"shadows-over-loathing":             {"1939160", "Shadows Over Loathing.app/Contents/MacOS/Shadows Over Loathing"},
	"grime":                             {"1123050", "GRIME.app/Contents/MacOS/GRIME"},
	"disco-elysium":                     {"632470", "disco.app/Contents/MacOS/Disco Elysium"},
	"fate-reawakened":                   {"3030720", "fate.app/Contents/MacOS/FATE Reawakened"},
	"papers-please":                     {"239030", "PapersPlease.app/Contents/MacOS/PapersPlease"},
}

func TestDarwinMarkersAreTheExecutablesSteamLaunchesOnMacOS(t *testing.T) {
	m, err := BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]GameInfo{}
	for _, g := range m.Games {
		got[g.ID] = g
	}
	for id, want := range steamMacExecutables {
		g, ok := got[id]
		if !ok {
			t.Errorf("%s is not in the catalog", id)
			continue
		}
		if g.DarwinMarker != want[1] {
			t.Errorf("%s: darwinMarker %q, want %q", id, g.DarwinMarker, want[1])
		}
		if g.SteamAppID() != want[0] {
			t.Errorf("%s: Steam app %q, want %q", id, g.SteamAppID(), want[0])
		}
	}
}

func TestLinuxMarkerIsASlashPathBelowTheInstall(t *testing.T) {
	g := GameInfo{ID: "x", Name: "x", Marker: "x.exe", LinuxMarker: "_linux/game.bin", Loaders: []GameLoader{{ID: "l"}}, Deploy: DeployProfile}
	if err := g.Validate(); err != nil {
		t.Fatalf("valid linuxMarker rejected: %v", err)
	}
	for _, bad := range []string{"../x", "/abs"} {
		ng := g
		ng.LinuxMarker = bad
		if ng.Validate() == nil {
			t.Fatalf("linuxMarker %q validated", bad)
		}
	}
	m, err := BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	found, ok := GameInfo{}, false
	for _, game := range m.Games {
		if game.ID == "darkest-dungeon" {
			found, ok = game, true
			break
		}
	}
	if !ok {
		t.Fatal("darkest-dungeon is not in the catalog")
	}
	if found.LinuxMarker != "_linux/darkest.bin.x86_64" {
		t.Errorf("linuxMarker %q", found.LinuxMarker)
	}
	if found.Paths["mods"].Linux != "{install}/mods" {
		t.Errorf("linux mods path %q", found.Paths["mods"].Linux)
	}
}
