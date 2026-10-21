package installer

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

func extracted(t *testing.T, key string, files map[string]string) Archive {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return Open(dir, key)
}

func render(l Layout) string {
	var out []string
	for _, f := range l.Files {
		out = append(out, f.Target+":"+f.Rel+"<-"+f.Src)
	}
	slices.Sort(out)
	return strings.Join(out, " ")
}

func TestLayoutGoldens(t *testing.T) {
	choose, err := os.ReadFile("../fomod/testdata/choose-one.xml")
	if err != nil {
		t.Fatal(err)
	}
	bep := Game{Loaders: []string{"bepinex5"}, Targets: []Target{{ID: TargetProfile}, {ID: TargetMods}}}
	smapi := Game{Loaders: []string{"smapi"}, Targets: []Target{{ID: TargetMods}}}
	mel := Game{Loaders: []string{"melonloader"}, Targets: []Target{{ID: TargetProfile}}}
	melClassic := mel
	melClassic.Routes = []components.LoaderRoute{
		{Dir: "Mods", Extensions: []string{".dll"}, Default: true},
		{Dir: "Plugins", Extensions: []string{".plugin.dll"}},
	}
	pack := func(id string) Game { return Game{Loaders: []string{id}, Targets: []Target{{ID: TargetProfile}}} }
	northstar := pack("northstar")
	northstar.Routes = []components.LoaderRoute{{Dir: "R2Northstar/mods"}}
	cases := []struct {
		name    string
		game    Game
		key     string
		files   map[string]string
		choices Choices
		driver  string
		want    string
	}{
		{
			"SMAPI mod zip", smapi, "nexus-1-2",
			map[string]string{"ModA/manifest.json": "{}", "ModA/ModA.dll": "x"},
			nil, "plain",
			"mods:nexus-1-2/ModA/ModA.dll<-ModA/ModA.dll mods:nexus-1-2/ModA/manifest.json<-ModA/manifest.json",
		},
		{
			"multi-mod Nexus zip with junk", smapi, "nexus-3-4",
			map[string]string{
				"One/manifest.json": "{}", "Two/manifest.json": "{}", "readme.txt": "r", "__MACOSX/One/._manifest.json": "j", "Two/Thumbs.db": "t", ".DS_Store": "d",
			},
			nil, "plain",
			"mods:nexus-3-4/One/manifest.json<-One/manifest.json mods:nexus-3-4/Two/manifest.json<-Two/manifest.json mods:nexus-3-4/readme.txt<-readme.txt",
		},
		{
			"FOMOD sample", smapi, "nexus-5-6",
			map[string]string{
				"fomod/ModuleConfig.xml": string(choose), "alpha/manifest.json": "a", "beta/manifest.json": "b",
			},
			Choices{"Options": {"Pack": {"Beta"}}},
			"fomod", "mods:nexus-5-6/manifest.json<-beta/manifest.json",
		},
		{
			"FOMOD destinations differing only in case", smapi, "nexus-7-8",
			map[string]string{
				"fomod/ModuleConfig.xml": `<config><moduleName>Case</moduleName><requiredInstallFiles>` +
					`<file source="one.txt" destination="Data.txt"/><file source="two.txt" destination="data.txt"/>` +
					`</requiredInstallFiles></config>`,
				"one.txt": "1", "two.txt": "2",
			},
			nil, "fomod", "mods:nexus-7-8/data.txt<-two.txt",
		},
		{
			"Thunderstore plugin", bep, "Ns-Mod",
			map[string]string{
				"manifest.json": `{"name":"Mod","version_number":"1.0.0"}`, "icon.png": "i", "README.md": "r", "plugins/Mod.dll": "d", "Assets/x.bundle": "b",
			},
			nil, "thunderstore-rules",
			"profile:BepInEx/plugins/Ns-Mod/Mod.dll<-plugins/Mod.dll profile:BepInEx/plugins/Ns-Mod/README.md<-README.md profile:BepInEx/plugins/Ns-Mod/icon.png<-icon.png profile:BepInEx/plugins/Ns-Mod/manifest.json<-manifest.json profile:BepInEx/plugins/Ns-Mod/x.bundle<-Assets/x.bundle",
		},
		{
			// MirageCore's shape: libraries in folders of their own, loaded from beside the plugin, each with a LICENSE.
			"Thunderstore package flattened by r2modman's rules", bep, "Ns-Core",
			map[string]string{
				"manifest.json": `{"name":"Core","version_number":"1.0.0"}`, "Core.dll": "c", "LICENSE": "root", "mod.dll": "m",
				"FSharp.Core/FSharp.Core.dll": "f", "FSharp.Core/LICENSE": "fs", "SileroVAD/LICENSE": "vad", "Sub/MOD.DLL": "M",
			},
			nil, "thunderstore-rules",
			"profile:BepInEx/plugins/Ns-Core/Core.dll<-Core.dll profile:BepInEx/plugins/Ns-Core/FSharp.Core.dll<-FSharp.Core/FSharp.Core.dll " +
				"profile:BepInEx/plugins/Ns-Core/LICENSE<-SileroVAD/LICENSE profile:BepInEx/plugins/Ns-Core/MOD.DLL<-Sub/MOD.DLL profile:BepInEx/plugins/Ns-Core/manifest.json<-manifest.json",
		},
		{
			"config-only Thunderstore package", bep, "Ns-Cfg",
			map[string]string{
				"manifest.json": `{"name":"Cfg","version_number":"1.0.0"}`, "config/ns.cfg.cfg": "c",
			},
			nil, "thunderstore-rules",
			"profile:BepInEx/config/ns.cfg.cfg<-config/ns.cfg.cfg profile:BepInEx/plugins/Ns-Cfg/manifest.json<-manifest.json",
		},
		{
			// The BOM is what Evaisa-HookGenPatcher's manifest carries; Thunderstore accepts it.
			"patchers-only Thunderstore package with a BOM manifest", bep, "Evaisa-HookGenPatcher",
			map[string]string{
				"manifest.json": "\ufeff" + `{"name":"HookGenPatcher","version_number":"0.0.5"}`, "icon.png": "i", "README.md": "r",
				"patchers/BepInEx.MonoMod.HookGenPatcher/HookGen.dll": "d",
			},
			nil, "thunderstore-rules",
			"profile:BepInEx/patchers/Evaisa-HookGenPatcher/BepInEx.MonoMod.HookGenPatcher/HookGen.dll<-patchers/BepInEx.MonoMod.HookGenPatcher/HookGen.dll profile:BepInEx/plugins/Evaisa-HookGenPatcher/README.md<-README.md profile:BepInEx/plugins/Evaisa-HookGenPatcher/icon.png<-icon.png profile:BepInEx/plugins/Evaisa-HookGenPatcher/manifest.json<-manifest.json",
		},
		{
			"config and patchers Thunderstore package", bep, "Ns-Gen",
			map[string]string{
				"manifest.json": "\ufeff" + `{"name":"Gen","version_number":"1.0.0"}`, "icon.png": "i",
				"config/Gen.cfg": "c", "patchers/Gen/Gen.dll": "d", "BepInEx/core/Core.dll": "k", "Loose.dll": "l",
			},
			nil, "thunderstore-rules",
			"profile:BepInEx/config/Gen.cfg<-config/Gen.cfg profile:BepInEx/core/Ns-Gen/Core.dll<-BepInEx/core/Core.dll " +
				"profile:BepInEx/patchers/Ns-Gen/Gen/Gen.dll<-patchers/Gen/Gen.dll profile:BepInEx/plugins/Ns-Gen/Loose.dll<-Loose.dll profile:BepInEx/plugins/Ns-Gen/icon.png<-icon.png profile:BepInEx/plugins/Ns-Gen/manifest.json<-manifest.json",
		},
		{
			"BepInEx plugin from another site, without a manifest", bep, "nexus-17-1",
			map[string]string{"lbtokg.dll": "d", "readme.txt": "r"},
			nil, "thunderstore-rules",
			"profile:BepInEx/plugins/nexus-17-1/lbtokg.dll<-lbtokg.dll profile:BepInEx/plugins/nexus-17-1/readme.txt<-readme.txt",
		},
		{
			"MelonLoader package under the recursive layout", mel, "Ns-Mel",
			map[string]string{
				"manifest.json": `{"name":"Mel","version_number":"1.0.0"}`, "icon.png": "i", "Mods/Mel.dll": "m", "Plugins/Lib.dll": "p", "UserData/Mel/cfg.toml": "c",
			},
			nil, "thunderstore-rules",
			"profile:Mods/Ns-Mel/Mods/Mel.dll<-Mods/Mel.dll profile:Mods/Ns-Mel/Plugins/Lib.dll<-Plugins/Lib.dll profile:UserData/Ns-Mel/Mel/cfg.toml<-UserData/Mel/cfg.toml",
		},
		{
			"MelonLoader package under the classic layout", melClassic, "Ns-Old",
			map[string]string{"manifest.json": `{"name":"Old","version_number":"1.0.0"}`, "Old.dll": "m", "Helper.plugin.dll": "p", "notes.txt": "n"},
			nil, "thunderstore-rules",
			"profile:Mods/Old.dll<-Old.dll profile:Mods/notes.txt<-notes.txt profile:Plugins/Helper.plugin.dll<-Helper.plugin.dll",
		},
		{
			"ReturnOfModding package", pack("returnofmodding"), "Ns-Rom",
			map[string]string{"manifest.json": `{"name":"Rom","version_number":"1.0.0"}`, "main.lua": "m", "config/a.cfg": "c"},
			nil, "thunderstore-rules",
			"profile:ReturnOfModding/config/Ns-Rom/a.cfg<-config/a.cfg profile:ReturnOfModding/plugins/Ns-Rom/main.lua<-main.lua",
		},
		{
			"GDWeave package", pack("gdweave"), "Ns-Fish",
			map[string]string{"manifest.json": `{"name":"Fish","version_number":"1.0.0"}`, "GDWeave/mods/Fish/manifest.json": "{}", "GDWeave/mods/Fish/Fish.dll": "d"},
			nil, "thunderstore-rules",
			"profile:GDWeave/mods/Ns-Fish/Fish.dll<-GDWeave/mods/Fish/Fish.dll profile:GDWeave/mods/Ns-Fish/manifest.json<-GDWeave/mods/Fish/manifest.json",
		},
		{
			"Lovely package", pack("lovely"), "Ns-Smods",
			map[string]string{"manifest.json": `{"name":"Smods","version_number":"1.0.0"}`, "lovely.toml": "t", "src/a.lua": "l"},
			nil, "thunderstore-rules",
			"profile:mods/Ns-Smods/lovely.toml<-lovely.toml profile:mods/Ns-Smods/src/a.lua<-src/a.lua",
		},
		{
			"Northstar package", northstar, "Ns-Skin",
			map[string]string{"manifest.json": `{"name":"Skin","version_number":"1.0.0"}`, "mods/Skin/mod.json": "{}", "notes.txt": "n"},
			nil, "thunderstore-rules",
			"profile:R2Northstar/mods/Skin/mod.json<-mods/Skin/mod.json",
		},
		{
			"a Lovely game takes only a Thunderstore package",
			Game{Loaders: []string{"lovely"}, Targets: []Target{{ID: TargetProfile}, {ID: TargetMods}}},
			"k",
			map[string]string{"mod.dll": "m"},
			nil, "plain",
			"mods:k/mod.dll<-mod.dll",
		},
		{
			"no manifest and no BepInEx content stays plain", bep, "k",
			map[string]string{"notes.txt": "n"},
			nil, "plain",
			"mods:k/notes.txt<-notes.txt",
		},
		{
			"Thunderstore manifest in a SMAPI game is plain", smapi, "k",
			map[string]string{"manifest.json": `{"name":"M","version_number":"1.0.0"}`},
			nil, "plain",
			"mods:k/manifest.json<-manifest.json",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := extracted(t, c.key, c.files)
			inst, ok := Pick(a, c.game)
			if !ok || inst.ID() != c.driver {
				t.Fatalf("picked %v, want %s", inst, c.driver)
			}
			l, err := inst.Layout(a, c.game, c.choices)
			if err != nil {
				t.Fatal(err)
			}
			if got := render(l); got != c.want {
				t.Fatalf("layout\n got %s\nwant %s", got, c.want)
			}
		})
	}
}

func TestLayoutRefusals(t *testing.T) {
	shallow := Game{Targets: []Target{{ID: TargetMods, MaxDepth: map[string]int{"ts4script": 2}}}}
	cases := map[string]map[string]string{
		"deeper than the target allows": {"a/b/c.ts4script": "x"}, // key + a + b = 3 folders
		"a name Windows cannot keep":    {"a/con.txt": "x"},
	}
	for name, files := range cases {
		a := extracted(t, "k", files)
		if _, err := (plain{}).Layout(a, shallow, nil); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	bad := Layout{Files: []File{{Src: "x", Target: TargetMods, Rel: "../escape"}, {Src: "y", Target: "nowhere", Rel: "y"}}}
	if validate(Layout{Files: bad.Files[:1]}, shallow) == nil || validate(Layout{Files: bad.Files[1:]}, shallow) == nil {
		t.Error("escape or unknown target accepted")
	}
}

func TestFolderGameSplitsUnlessAKeepWholeFileIsLaidOut(t *testing.T) {
	t.Parallel()
	g := Game{Loaders: []string{"folder"}, Targets: []Target{{ID: TargetMods, KeepWhole: []string{"ts4script"}}}}
	pkg := Layout{Files: []File{{Target: TargetMods, Rel: "a.package"}, {Target: TargetMods, Rel: "b.PACKAGE"}}}
	script := Layout{Files: append(slices.Clone(pkg.Files), File{Target: TargetMods, Rel: "m.Ts4Script"})}
	if !g.Splits(pkg) || g.Splits(script) {
		t.Fatalf("splits: packages %v, with a script %v", g.Splits(pkg), g.Splits(script))
	}
	if (Game{Loaders: []string{"smapi"}, Targets: g.Targets}).Splits(pkg) {
		t.Fatal("a game with another loader never splits")
	}
}

func TestFolderGameLaysAModRootFolderOutWholeAsOneEntry(t *testing.T) {
	t.Parallel()
	g := Game{Loaders: []string{"folder"}, Targets: []Target{{ID: TargetMods, ModRoot: "modinfo.json"}}}
	l, err := (plain{}).Layout(extracted(t, "k", map[string]string{
		"CarryOn/modinfo.json": "{}", "CarryOn/assets/a.json": "{}", "CarryOn/x/modinfo.json": "{}", "readme.txt": "x",
	}), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range l.Files {
		got[f.Rel] = true
	}
	// The shallowest folder holding the marker is the mod; a file beside it is not the mod's.
	if len(got) != 3 || !got["k/modinfo.json"] || !got["k/assets/a.json"] || !got["k/x/modinfo.json"] || g.Splits(l) {
		t.Fatalf("layout = %v, splits %v", got, g.Splits(l))
	}
	loose, err := (plain{}).Layout(extracted(t, "k", map[string]string{"Mod.cs": "x"}), g, nil)
	if err != nil || len(loose.Files) != 1 || loose.Files[0].Rel != "Mod.cs" {
		t.Fatalf("loose = %+v, %v", loose, err)
	}
}
