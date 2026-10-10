//go:build !windows

package launchsvc

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/savesiso"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const optionsGame = "options-game"

// optionsEnv is a service for a catalog game whose options file lives under a fake Documents tree in the service's home.
func optionsEnv(t *testing.T, mode string) (svc *Service, profileID, target string) {
	t.Helper()
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	m.Games = append(slices.Clone(m.Games), components.GameInfo{
		ID: optionsGame, Name: "Options Game", Enabled: true, Marker: "G.dll", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "mods", Root: "{profile}/Mods", MaxDepth: map[string]int{"pkg": 1}}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "1"}},
		Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
		Paths: map[string]components.PathTemplate{
			"mods":    {Windows: "{documents}/G/Mods", Linux: "{documents}/G/Mods", Darwin: "{documents}/G/Mods"},
			"options": {Windows: "{documents}/G/Options.ini", Linux: "{documents}/G/Options.ini", Darwin: "{documents}/G/Options.ini"},
		},
		RequiredSettings: []components.RequiredSetting{
			{Path: "options", Key: "ModsDisabled", Value: "0", Message: "mods off"},
			{Path: "options", Key: "ScriptMods", Value: "1", Message: "scripts off"},
		},
	})
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	c := components.NewClient(nil)
	c.SetManifest(m)
	components.Use(c)
	t.Cleanup(func() { components.Use(nil) })

	data := t.TempDir()
	datadirtest.Use(t, data)
	t.Setenv("LOCALAPPDATA", data)
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "G.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) {
		v.GameFolders[optionsGame] = folder
		if mode != "" {
			gp := v.GamePrefs(optionsGame)
			gp.GameSettingsMode = mode
			if v.Games == nil {
				v.Games = map[string]*settings.GameSettings{}
			}
			v.Games[optionsGame] = &gp
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, optionsGame, "A")
	home := t.TempDir()
	svc = NewService(home, set, profiles)
	target = filepath.Join(home, "Documents", "G", "Options.ini")
	return svc, p.ID, target
}

func writeFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := fsx.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSwapOptionsSeedsTheProfileAndReturnsThePlayersFile(t *testing.T) {
	svc, pid, target := optionsEnv(t, settings.GameSettingsWarn)
	g := game.Find(optionsGame)
	_, selected := svc.installs(g)
	writeFile(t, target, "[options]\nModsDisabled = 1\n")
	dep := &deployment{}
	if err := svc.swapOptions(context.Background(), optionsGame, selected, pid, "", dep); err != nil {
		t.Fatal(err)
	}
	if dep.options == nil || dep.finish == nil {
		t.Fatalf("the swap is not tracked for undoing: %+v", dep)
	}
	own := filepath.Join(filepath.Dir(dep.options.Profile), "Options.ini")
	if readFile(t, own) != "[options]\nModsDisabled = 1\n" || readFile(t, target) != readFile(t, own) {
		t.Fatal("the first launch seeds the profile copy from the player's file")
	}
	writeFile(t, target, "[options]\nplayed = 1\n")
	dep.unwind(context.Background())
	if readFile(t, target) != "[options]\nModsDisabled = 1\n" {
		t.Fatalf("the player's file after the launch = %q", readFile(t, target))
	}
	if readFile(t, own) != "[options]\nplayed = 1\n" {
		t.Fatal("the profile keeps what the game wrote")
	}
}

func TestSwapOptionsAfterACrashIsRecoveredAtNextLaunchAndStart(t *testing.T) {
	svc, pid, target := optionsEnv(t, settings.GameSettingsWarn)
	g := game.Find(optionsGame)
	_, selected := svc.installs(g)
	writeFile(t, target, "player")
	if err := svc.swapOptions(context.Background(), optionsGame, selected, pid, "", &deployment{}); err != nil {
		t.Fatal(err)
	}
	dir, err := optionsJournal(selected.ID)
	if err != nil || !savesiso.HasFileJournal(dir) {
		t.Fatalf("no journal after the swap: %v", err)
	}
	if left := svc.LeftoverJournals(optionsGame); len(left) != 1 {
		t.Fatalf("leftover journals = %v", left)
	}
	if found, err := svc.RecoverGameDeploys(context.Background(), optionsGame); err != nil || !found {
		t.Fatalf("recover at start: %v %v", found, err)
	}
	if readFile(t, target) != "player" || savesiso.HasFileJournal(dir) {
		t.Fatal("recovery must return the player's file")
	}
}

func TestSwapOptionsSkipsAGameWithoutTheRole(t *testing.T) {
	svc, p := startEnv(t)
	g := game.Find("stardew")
	_, selected := svc.installs(g)
	dep := &deployment{}
	if err := svc.swapOptions(context.Background(), "stardew", selected, p.ID, "", dep); err != nil || dep.options != nil {
		t.Fatalf("a game without an options role was swapped: %v %+v", err, dep)
	}
}

func TestSwapOptionsEditModeChangesOnlyTheProfilesCopy(t *testing.T) {
	player := "[options]\r\nModsDisabled = 1\r\nOther = 5\r\n"
	svc, pid, target := optionsEnv(t, "")
	g := game.Find(optionsGame)
	_, selected := svc.installs(g)
	writeFile(t, target, player)
	dep := &deployment{}
	if err := svc.swapOptions(context.Background(), optionsGame, selected, pid, "", dep); err != nil {
		t.Fatal(err)
	}
	want := "[options]\r\nModsDisabled = 0\r\nOther = 5\r\nScriptMods = 1\r\n"
	if got := readFile(t, target); got != want {
		t.Fatalf("the game sees %q, want %q", got, want)
	}
	dep.unwind(context.Background())
	if got := readFile(t, target); got != player {
		t.Fatalf("the player's file changed: %q", got)
	}
	own := dep.options.Profile
	if got := readFile(t, own); got != want {
		t.Fatalf("the profile copy = %q", got)
	}
}

func TestSwapOptionsWarnModeLeavesTheBytesAlone(t *testing.T) {
	player := "[options]\nModsDisabled = 1\n"
	svc, pid, target := optionsEnv(t, settings.GameSettingsWarn)
	g := game.Find(optionsGame)
	_, selected := svc.installs(g)
	writeFile(t, target, player)
	dep := &deployment{}
	if err := svc.swapOptions(context.Background(), optionsGame, selected, pid, "", dep); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, target); got != player {
		t.Fatalf("warn mode edited the file: %q", got)
	}
	dep.unwind(context.Background())
	if got := readFile(t, dep.options.Profile); got != player {
		t.Fatalf("profile copy = %q", got)
	}
}

func TestPlanProfileListsPackagesThatOnlyTheLayoutPutsInTheProfile(t *testing.T) {
	svc, pid, _ := optionsEnv(t, "")
	g := game.Find(optionsGame)
	_, selected := svc.installs(g)
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "m.zip"), map[string]string{"R/x.pkg": "x"})
	if _, err := svc.profiles.InstallArchive(context.Background(), optionsGame, pid, zip); err != nil {
		t.Fatal(err)
	}
	plan, err := svc.planProfile(context.Background(), g, selected, pid, launchplan.ModeProfile, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 1 || plan.Files[0].Root != "mods" {
		t.Fatalf("the plan must carry the installed mod for the mods folder: %+v", plan.Files)
	}
}

func TestSwapOptionsLeavesAFileMortarCannotEditAndStillLaunches(t *testing.T) {
	utf16 := "\xff\xfe[\x00o\x00]\x00\nM\x00o\x00d\x00s\x00D\x00i\x00s\x00a\x00b\x00l\x00e\x00d\x00=\x001\x00"
	svc, pid, target := optionsEnv(t, "")
	g := game.Find(optionsGame)
	_, selected := svc.installs(g)
	writeFile(t, target, utf16)
	dep := &deployment{}
	if err := svc.swapOptions(context.Background(), optionsGame, selected, pid, "", dep); err != nil {
		t.Fatalf("a file Mortar cannot edit stopped the launch: %v", err)
	}
	if got := readFile(t, target); got != utf16 {
		t.Fatalf("the game's file was changed: %q", got)
	}
	dep.unwind(context.Background())
	if got := readFile(t, target); got != utf16 || readFile(t, dep.options.Profile) != utf16 {
		t.Fatal("the player's file or the profile copy changed")
	}
}
