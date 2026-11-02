package bannerlord

import (
	"bytes"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	bl "github.com/Rethunk-Tech/go-bannerlord"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/deploy"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const game = "mount-and-blade-2-bannerlord"

// subModule is a SubModule.xml; deps are "Id@version" entries.
func subModule(id string, official, def bool, deps ...string) []byte {
	var b strings.Builder
	b.WriteString(`<Module><Name value="` + id + `"/><Id value="` + id + `"/><Version value="v1.0.0"/>`)
	if official {
		b.WriteString(`<Official value="true"/>`)
	}
	if def {
		b.WriteString(`<DefaultModule value="true"/>`)
	}
	b.WriteString(`<DependedModules>`)
	for _, d := range deps {
		dep, ver, _ := strings.Cut(d, "@")
		b.WriteString(`<DependedModule Id="` + dep + `"`)
		if ver != "" {
			b.WriteString(` DependentVersion="` + ver + `"`)
		}
		b.WriteString(`/>`)
	}
	b.WriteString(`</DependedModules></Module>`)
	return []byte(b.String())
}

func put(t *testing.T, root, id string, xml []byte) {
	t.Helper()
	testfs.WriteAt(t, filepath.Join(root, id, "SubModule.xml"), xml)
}

func fixture(t *testing.T) (root string, view loader.ProfileView) {
	t.Helper()
	root = t.TempDir()
	install, profile := filepath.Join(root, "install"), filepath.Join(root, "profile")
	official := filepath.Join(install, "Modules")
	put(t, official, "Native", subModule("Native", true, true))
	put(t, official, "SandBoxCore", subModule("SandBoxCore", true, true, "Native"))
	put(t, official, "StoryMode", subModule("StoryMode", true, false, "SandBoxCore"))
	put(t, official, "MultiplayerModule", subModule("MultiplayerModule", true, false, "Native"))
	own := filepath.Join(profile, "Modules")
	put(t, own, "Zeta", subModule("Zeta", false, false, "Native", "Alpha"))
	put(t, own, "Alpha", subModule("Alpha", false, false, "StoryMode"))
	return root, loader.ProfileView{Game: game, Dir: profile, InstallDir: install}
}

func TestPrelaunchSelectsSortedModulesAndTheDeployerRestoresTheOwn(t *testing.T) {
	t.Parallel()
	root, view := fixture(t)
	if err := (extension{}).Prelaunch(view); err != nil {
		t.Fatal(err)
	}
	got := selectedModules(t, view)
	// The multiplayer module is official but neither a default nor needed; StoryMode is needed by Alpha.
	want := []string{"Native", "SandBoxCore", "StoryMode", "Alpha", "Zeta"}
	if !slices.Equal(got, want) {
		t.Fatalf("selected %v, want %v", got, want)
	}

	plan := launchplan.New(launchplan.ModeProfile)
	extension{}.Contribute(plan, view)
	if len(plan.Files) != 1 || plan.Files[0].Root != Role || plan.Files[0].Dst != "LauncherData.xml" {
		t.Fatalf("plan files %+v", plan.Files)
	}
	configs := filepath.Join(root, "Configs")
	own := filepath.Join(configs, "LauncherData.xml")
	testfs.WriteAt(t, own, []byte("the player's own"))
	d := deploy.Place{}
	dv := deploy.View{JournalDir: filepath.Join(root, "journal"), Roots: map[string]string{Role: configs}}
	dp, err := d.Plan(dv, view.InstallDir, plan.Files)
	if err != nil {
		t.Fatal(err)
	}
	m, err := d.Apply(t.Context(), dp)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(mustRead(t, own), mustRead(t, generatedPath(view.Dir))) {
		t.Fatal("the generated LauncherData.xml is not in place for the run")
	}
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, own)) != "the player's own" {
		t.Fatalf("the player's LauncherData.xml is not back: %q", mustRead(t, own))
	}
}

func TestFindingsNameTheProfilesModuleProblems(t *testing.T) {
	t.Parallel()
	_, view := fixture(t)
	own := filepath.Join(view.Dir, "Modules")
	put(t, own, "Lonely", subModule("Lonely", false, false, "Nowhere"))
	put(t, own, "Stale", subModule("Stale", false, false, "Native@v9.0.0"))
	put(t, own, "CycA", subModule("CycA", false, false, "CycB"))
	put(t, own, "CycB", subModule("CycB", false, false, "CycA"))
	// An official module with a missing dependency is the game's problem, not the profile's.
	put(t, filepath.Join(view.InstallDir, "Modules"), "Broken", subModule("Broken", true, false, "Gone"))

	byCode := map[string]loader.Finding{}
	for _, f := range (extension{}).Findings(view) {
		byCode[f.Code+":"+f.Plugin] = f
	}
	for _, key := range []string{CodeMissing + ":Lonely", CodeOutdated + ":Stale"} {
		if _, ok := byCode[key]; !ok {
			t.Fatalf("no %s in %v", key, byCode)
		}
	}
	if f := byCode[CodeMissing+":Lonely"]; f.Dependency != "Nowhere" || f.Kind != "missing-dependency" {
		t.Fatalf("missing finding %+v", f)
	}
	if len(byCode) != 3 {
		t.Fatalf("findings %v, want missing, outdated and one cycle only", byCode)
	}
	cyc := byCode[CodeCycle+":CycA"]
	if cyc.Code == "" || cyc.Params["list"] == "" {
		t.Fatalf("cycle finding %+v in %v", cyc, byCode)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := safefs.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPrelaunchSelectsTheOfficialModulesADependencyNeedsTransitively(t *testing.T) {
	t.Parallel()
	_, view := fixture(t)
	official := filepath.Join(view.InstallDir, "Modules")
	put(t, official, "DlcA", subModule("DlcA", true, false, "Native", "DlcB"))
	put(t, official, "DlcB", subModule("DlcB", true, false, "Native"))
	put(t, filepath.Join(view.Dir, "Modules"), "NeedsDlc", subModule("NeedsDlc", false, false, "DlcA"))
	if err := (extension{}).Prelaunch(view); err != nil {
		t.Fatal(err)
	}
	got := selectedModules(t, view)
	if !slices.Contains(got, "DlcA") || !slices.Contains(got, "DlcB") || slices.Contains(got, "MultiplayerModule") {
		t.Fatalf("selected %v", got)
	}
}

func selectedModules(t *testing.T, view loader.ProfileView) []string {
	t.Helper()
	l, err := bl.ParseLauncher(bytes.NewReader(mustRead(t, generatedPath(view.Dir))))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range l.Mods(bl.Singleplayer) {
		if m.Selected {
			got = append(got, m.ID)
		}
	}
	return got
}
