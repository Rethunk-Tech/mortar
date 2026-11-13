package components

import "testing"

// Each value is the macOS executable below the install folder: [app id, darwinMarker].
// Steam's config.launch macos entry is used when it names Contents/MacOS/<exe>
// (api.steamcmd.net/v1/info/<appId>, backslashes normalised). When that entry names
// only a .app, the marker is the bundle plus Contents/MacOS/<CFBundleExecutable> from
// the Mac depot file list (the file Info.plist names).
var steamMacExecutables = map[string][2]string{
	"20-minutes-till-dawn":              {"1966900", "20MinutesTillDawn.app/Contents/MacOS/MinutesTillDawn"},
	"7-days-to-die":                     {"251570", "7DaysToDie.app/Contents/MacOS/7 Days To Die"},
	"9-kings":                           {"2784470", "9Kings.app/Contents/MacOS/9Kings"},
	"a-dance-of-fire-and-ice":           {"977950", "ADanceOfFireAndIce.app/Contents/MacOS/ADanceOfFireAndIce"},
	"a-gentlemens-dispute":              {"2820700", "AGentlemensDispute.app/Contents/MacOS/A Gentlemen's Dispute"},
	"a-little-to-the-left":              {"1629520", "A Little To The Left.app/Contents/MacOS/A Little To The Left"},
	"a-webbing-journey":                 {"2073910", "A Webbing Journey.app/Contents/MacOS/A Webbing Journey"},
	"across-the-obelisk":                {"1385380", "Contents/MacOS/AcrossTheObelisk"},
	"aneurism-iv":                       {"2773280", "ANEURISM IV.app/Contents/MacOS/ANEURISM IV"},
	"arcus-chroma":                      {"1447350", "Arcus Chroma Mac.app/Contents/MacOS/Arcus Chroma"},
	"backpack-hero":                     {"1970580", "BackpackHero.app/Contents/MacOS/Backpack Hero"},
	"bad-north":                         {"688420", "BadNorth.app/Contents/MacOS/BadNorth"},
	"balatro":                           {"2379780", "Balatro.app/Contents/MacOS/love"},
	"beetleball":                        {"2776270", "Beetleball.app/Contents/MacOS/Beetleball"},
	"bendy-and-the-ink-machine":         {"622650", "BATIM.app/Contents/MacOS/BATIM"},
	"big-ambitions":                     {"1331550", "Big Ambitions.app/Contents/MacOS/Big Ambitions"},
	"big-walk":                          {"1478500", "Big Walk.app/Contents/MacOS/Big Walk"},
	"block-story":                       {"270110", "Contents/MacOS/Block Story"},
	"bloons-td-6":                       {"960090", "BloonsTD6.app/Contents/MacOS/BloonsTD6"},
	"blue-prince":                       {"1569580", "BLUE PRINCE.app/Contents/MacOS/BLUE PRINCE"},
	"book-of-travels":                   {"1152340", "BookOfTravels.app/Contents/MacOS/BookOfTravels"},
	"broforce":                          {"274190", "Broforce.app/Contents/MacOS/Broforce"},
	"brotato":                           {"1942280", "Brotato.app/Contents/MacOS/Brotato"},
	"cast-n-chill":                      {"3483740", "Cast n Chill.app/Contents/MacOS/Cast n Chill"},
	"castle-story":                      {"227860", "Castle Story.app/Contents/MacOS/Castle Story"},
	"cat-mail-co":                       {"4380490", "CatMailCo-OSX-Universal.app/Contents/MacOS/CatMailCo"},
	"cats-are-liquid":                   {"1188080", "CaL-ABP-macOS.app/Contents/MacOS/Cats are Liquid - A Better Place"},
	"cave-crawlers":                     {"2143030", "Contents/MacOS/Cave Crawlers"},
	"chill-with-you":                    {"3548580", "Chill With You.app/Contents/MacOS/Chill With You"},
	"crab-game":                         {"1782210", "Crab Game.app/Contents/MacOS/Crab Game"},
	"crawl":                             {"293780", "Crawl.app/Contents/MacOS/Crawl"},
	"cult-of-pin":                       {"3144340", "CultOfPin.app/Contents/MacOS/Cult of PIN"},
	"cult-of-the-lamb":                  {"1313140", "Cult Of The Lamb.app/Contents/MacOS/Cult Of The Lamb"},
	"cuphead":                           {"268910", "Cuphead.app/Contents/MacOS/Cuphead"},
	"depo":                              {"1091320", "Contents/MacOS/DEPO"},
	"disco-elysium":                     {"632470", "disco.app/Contents/MacOS/Disco Elysium"},
	"distance":                          {"233610", "Distance.app/Contents/MacOS/Distance"},
	"dome-keeper":                       {"1637320", "Contents/MacOS/Dome Keeper"},
	"dredge":                            {"1562430", "DREDGE.app/Contents/MacOS/DREDGE"},
	"ducks-can-drive":                   {"2472840", "Ducks Can Drive.app/Contents/MacOS/Ducks Can Drive"},
	"dusk":                              {"519860", "DUSK.app/Contents/MacOS/DUSK"},
	"electoral-carnage":                 {"3706810", "Electoral Carnage.app/Contents/MacOS/Electoral Carnage"},
	"ena-dream-bbq":                     {"2134320", "ENA-4-DreamBBQ.app/Contents/MacOS/ENA-4-DreamBBQ"},
	"enter-the-gungeon":                 {"311690", "EtG_OSX.app/Contents/MacOS/EtG_OSX"},
	"fate-reawakened":                   {"3030720", "fate.app/Contents/MacOS/FATE Reawakened"},
	"football-manager-26":               {"3551340", "fm.app/Contents/MacOS/Football Manager 26"},
	"grime":                             {"1123050", "GRIME.app/Contents/MacOS/GRIME"},
	"magicite":                          {"268750", "Magicite.app/Contents/MacOS/Magicite"},
	"minigunner":                        {"1778200", "Contents/MacOS/Minigunner"},
	"muck":                              {"1625450", "Muck.app/Contents/MacOS/Muck"},
	"my-dream-setup":                    {"2200780", "MDS.app/Contents/MacOS/MDS"},
	"necropolis":                        {"384490", "Necropolis.app/Contents/MacOS/Necropolis"},
	"papers-please":                     {"239030", "PapersPlease.app/Contents/MacOS/PapersPlease"},
	"paralives":                         {"1118520", "Paralives.app/Contents/MacOS/Paralives"},
	"pikuniku":                          {"572890", "Pikuniku.app/Contents/MacOS/Pikuniku"},
	"project-gorgon":                    {"342940", "MacPlayer.app/Contents/MacOS/Project Gorgon"},
	"ravenfield":                        {"636480", "ravenfield.app/Contents/MacOS/RavenfieldSteam"},
	"shadows-over-loathing":             {"1939160", "Shadows Over Loathing.app/Contents/MacOS/Shadows Over Loathing"},
	"skul-the-hero-slayer":              {"1147560", "Skul.app/Contents/MacOS/Skul"},
	"stardew":                           {"413150", "Contents/MacOS/StardewValley"},
	"stick-fight-the-game":              {"674940", "StickFight.app/Contents/MacOS/StickFight"},
	"sun-haven":                         {"1432860", "Contents/MacOS/Sun Haven"},
	"timberborn":                        {"1062090", "Timberborn.app/Contents/MacOS/Timberborn"},
	"totally-accurate-battle-simulator": {"508440", "TotallyAccurateBattleSimulator.app/Contents/MacOS/Totally Accurate Battle Simulator"},
	"valheim":                           {"892970", "valheim.app/Contents/MacOS/Valheim"},
	"very-very-valet":                   {"1448840", "macOS/Very Very Valet.app/Contents/MacOS/Very Very Valet"},
	"west-of-loathing":                  {"597220", "West of Loathing.app/Contents/MacOS/West of Loathing"},
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
