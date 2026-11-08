package profile

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestInstallArchiveAsksForFomodThenInstallsChoices(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "A")
	xml, err := safefs.ReadFile(filepath.Join("..", "installer", "testdata", "fomod", "choose-one.xml"))
	if err != nil {
		t.Fatal(err)
	}
	z := buildZip(t, "fomod.zip", map[string]string{
		"fomod/ModuleConfig.xml": string(xml),
		"alpha/manifest.json":    manifestJSON("A.Alpha"),
		"beta/manifest.json":     manifestJSON("A.Beta"),
	})
	res, err := e.InstallArchive(t.Context(), "stardew", p.ID, z)
	if err != nil {
		t.Fatal(err)
	}
	if res.Fomod == nil || res.Fomod.ModuleName != "Choose One" {
		t.Fatalf("want fomod ask, got %+v", res)
	}
	if len(res.Profile.Entries) != 0 {
		t.Fatalf("asked but already installed: %+v", res.Profile.Entries)
	}
	res, err = e.InstallFomod("stardew", p.ID, res.Fomod.Key, res.Fomod.Source.WithDisabled([]mod.ID{"smapi:A.Alpha"}), map[string]map[string][]string{
		"Options": {"Pack": {"Alpha"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Fomod != nil || len(res.Added) != 1 || res.Added[0] != "A.Alpha" {
		t.Fatalf("install: %+v", res)
	}
	got, err := os.ReadFile(filepath.Join(e.mods(p.ID), "."+res.Profile.Entries[0].Key, "manifest.json"))
	if err != nil || string(got) != manifestJSON("A.Alpha") {
		t.Fatalf("layout %q %v", got, err)
	}
	if res.Profile.Entries[0].Fomod["Options"]["Pack"][0] != "Alpha" {
		t.Fatalf("stored choices: %+v", res.Profile.Entries[0].Fomod)
	}
	if len(res.Profile.Entries[0].Disabled) != 1 || res.Profile.Entries[0].Disabled[0] != "smapi:A.Alpha" {
		t.Fatalf("disabled mods: %+v", res.Profile.Entries[0].Disabled)
	}
}

func TestFomodReplayMismatchAsksAgain(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "A")
	xml, err := safefs.ReadFile(filepath.Join("..", "installer", "testdata", "fomod", "choose-one.xml"))
	if err != nil {
		t.Fatal(err)
	}
	e.item(t, "local-fomod", map[string]string{
		"fomod/ModuleConfig.xml": string(xml),
		"alpha/manifest.json":    manifestJSON("A.Alpha"),
		"beta/manifest.json":     manifestJSON("A.Beta"),
	})
	res, err := e.InstallFomod("stardew", p.ID, "local-fomod", Source{Kind: KindLocal, Name: "x"}, map[string]map[string][]string{
		"Options": {"Pack": {"Gone"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Fomod == nil {
		t.Fatal("vanished choice should ask again")
	}
}

func TestFomodUpdateReplaysOnlyUnchangedOptions(t *testing.T) {
	t.Parallel()
	raw, err := safefs.ReadFile(filepath.Join("..", "installer", "testdata", "fomod", "choose-one.xml"))
	if err != nil {
		t.Fatal(err)
	}
	base := string(raw)
	extra := `<group name="Extras" type="SelectAny"><plugins><plugin name="Gamma"><description/><files/>` +
		`<typeDescriptor><type name="Optional"/></typeDescriptor></plugin></plugins></group></optionalFileGroups>`
	alpha := map[string]map[string][]string{"Options": {"Pack": {"Alpha"}}}
	for _, tc := range []struct {
		name, xml string
		want      map[string]map[string][]string
	}{
		{"unchanged", base, nil},
		{"new group", strings.Replace(base, "</optionalFileGroups>", extra, 1), alpha},
		{"plugin gone", strings.Replace(base, `name="Alpha"`, `name="Alpha2"`, 1), map[string]map[string][]string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			p, err := e.Create("stardew", "A")
			if err != nil {
				t.Fatal(err)
			}
			files := func(xml string) map[string]string {
				return map[string]string{
					"fomod/ModuleConfig.xml": xml,
					"alpha/manifest.json":    manifestJSON("A.Alpha"),
					"beta/manifest.json":     manifestJSON("A.Beta"),
				}
			}
			e.item(t, "v1", files(base))
			e.item(t, "v2", files(tc.xml))
			if _, err := e.InstallFomod("stardew", p.ID, "v1", Source{Kind: KindLocal, Name: "x"}, alpha); err != nil {
				t.Fatal(err)
			}
			src, ask, need, err := e.installAsk("stardew", p.ID, "v2", Source{Kind: KindLocal, Name: "x"})
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == nil {
				if need || !reflect.DeepEqual(src.fomodMap(), alpha) {
					t.Fatalf("install should replay silently: need=%v choices=%v", need, src.fomodMap())
				}
			} else if !need || !ask.Changed || ask.OldKey != "v1" || !reflect.DeepEqual(ask.Choices, tc.want) {
				t.Fatalf("install should re-offer: need=%v ask=%+v", need, ask)
			}
			_, err = e.UpdateEntry("stardew", p.ID, "v1", "v2")
			need2, asked := errors.AsType[*NeedChoicesError](err)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("update should replay silently: %v", err)
				}
			} else if !asked || !need2.Ask.Changed || !reflect.DeepEqual(need2.Ask.Choices, tc.want) {
				t.Fatalf("update should re-offer: %v", err)
			}
		})
	}
}

func TestFomodImageRefusesAFileOverTheCap(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-img", map[string]string{
		"small.png": "x",
		"big.png":   strings.Repeat("x", maxFomodImage+1),
	})
	if b, err := e.FomodImage("stardew", "local-img", "small.png"); err != nil || string(b) != "x" {
		t.Fatalf("small image = %q, %v", b, err)
	}
	if _, err := e.FomodImage("stardew", "local-img", "big.png"); err == nil {
		t.Fatal("an image over the cap was read")
	}
}

const requiredAndOptionalFomod = `<config><moduleName>Pack</moduleName>` +
	`<requiredInstallFiles><file source="required/manifest.json" destination="required/manifest.json"/></requiredInstallFiles>` +
	`<installSteps><installStep name="Setup"><optionalFileGroups>` +
	`<group name="Extras" type="SelectAny"><plugins><plugin name="Extra"><description/><files>` +
	`<file source="extra/manifest.json" destination="extra/manifest.json"/></files>` +
	`<typeDescriptor><type name="Optional"/></typeDescriptor></plugin></plugins></group>` +
	`</optionalFileGroups></installStep></installSteps></config>`

func TestInstallFomodLaysOutRequiredAndChosenFiles(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "A")
	e.item(t, "local-fomod", map[string]string{
		"fomod/ModuleConfig.xml": requiredAndOptionalFomod,
		"required/manifest.json": manifestJSON("A.Required"),
		"extra/manifest.json":    manifestJSON("A.Extra"),
	})
	ask, err := e.FomodPreview("stardew", p.ID, "local-fomod", nil)
	if err != nil || ask.ModuleName != "Pack" || len(ask.Steps) != 1 || ask.Steps[0].Name != "Setup" ||
		len(ask.Steps[0].Groups) != 1 || ask.Steps[0].Groups[0].Name != "Extras" ||
		ask.Steps[0].Groups[0].Type != "SelectAny" || len(ask.Steps[0].Groups[0].Plugins) != 1 ||
		ask.Steps[0].Groups[0].Plugins[0].Name != "Extra" {
		t.Fatalf("preview = %+v, %v", ask, err)
	}
	choices := map[string]map[string][]string{"Setup": {"Extras": {"Extra"}}}
	res, err := e.InstallFomod("stardew", p.ID, "local-fomod", Source{Kind: KindLocal, Name: "x"}, choices)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(e.mods(p.ID), res.Profile.Entries[0].Key)
	if got := testfs.Read(t, filepath.Join(root, "required", "manifest.json")); got != manifestJSON("A.Required") {
		t.Fatalf("required layout %q", got)
	}
	if got := testfs.Read(t, filepath.Join(root, "extra", "manifest.json")); got != manifestJSON("A.Extra") {
		t.Fatalf("extra layout %q", got)
	}
	got := res.Profile.Entries[0].Fomod
	if got["Setup"]["Extras"][0] != "Extra" {
		t.Fatalf("stored choices: %+v", got)
	}
	ask, err = e.FomodPreview("stardew", p.ID, "local-fomod", nil)
	if err != nil || ask.Choices["Setup"]["Extras"][0] != "Extra" {
		t.Fatalf("preview after install = %+v, %v", ask, err)
	}
	res, err = e.InstallFomod("stardew", p.ID, "local-fomod", Source{Kind: KindLocal, Name: "x"}, map[string]map[string][]string{
		"Setup": {"Extras": {}},
	})
	if err != nil {
		t.Fatal(err)
	}
	root = filepath.Join(e.mods(p.ID), res.Profile.Entries[0].Key)
	if _, err := os.Stat(filepath.Join(root, "required", "manifest.json")); err != nil {
		t.Fatalf("required file dropped: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "extra", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("extra file still laid out: %v", err)
	}
	if extras := res.Profile.Entries[0].Fomod["Setup"]["Extras"]; len(extras) != 0 {
		t.Fatalf("stored choices after reinstall: %+v", res.Profile.Entries[0].Fomod)
	}
}

func TestInstalledGameVersionWithAndWithoutAVersionFile(t *testing.T) {
	e := newEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	// Where SMAPI writes its log: {appData} on Windows, {xdgConfig} elsewhere.
	xdg := filepath.Join(home, ".config")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if runtime.GOOS == "windows" {
		xdg = filepath.Join(home, "AppData", "Roaming")
		t.Setenv("APPDATA", xdg)
	}
	install := t.TempDir()
	writeFile(t, install, "Stardew Valley.dll", "x")
	// Status only reads SMAPI-latest.txt when the loader looks installed or broken: the .dll marks it on Linux and
	// macOS, the .exe on Windows.
	writeFile(t, install, "StardewModdingAPI.dll", "x")
	writeFile(t, install, "StardewModdingAPI.exe", "x")
	set, err := settings.OpenIn(e.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders["stardew"] = install }); err != nil {
		t.Fatal(err)
	}
	e.settings = set
	if got := e.installedGameVersion("stardew"); got != "" {
		t.Fatalf("without version file: %q", got)
	}
	writeFile(t, xdg, "StardewValley/ErrorLogs/SMAPI-latest.txt",
		"[14:00:00 INFO SMAPI] SMAPI 4.1.10 with Stardew Valley 1.6.15\n")
	if got := e.installedGameVersion("stardew"); got != "1.6.15" {
		t.Fatalf("with version file: %q", got)
	}
}
