package thunderpack

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestLayout(t *testing.T) {
	northstar := []components.LoaderRoute{{Dir: "R2Northstar/mods"}}
	umm := []components.LoaderRoute{
		{Dir: "UMM/Mods", Extensions: []string{".dll"}, Track: components.TrackSubdir, Default: true},
		{Dir: "UMM/BroMaker_Storage", Track: components.TrackSubdir},
	}
	shim := []components.LoaderRoute{
		{Dir: "shimloader/mod", Track: components.TrackSubdir, Default: true},
		{Dir: "shimloader/pak", Extensions: []string{".pak"}, Track: components.TrackSubdir},
		{Dir: "shimloader/cfg"},
		{Dir: "shimloader/overlay", Track: components.TrackSubdir},
	}
	cases := []struct {
		loader string
		files  []string
		routes []components.LoaderRoute
		want   map[string]string
	}{
		{"returnofmodding", []string{"manifest.json", "README.md", "main.lua", "custom/file.txt", "plugins/b.lua", "plugins_data/cache/d.dat", "config/c.cfg", "../x"}, nil, map[string]string{
			"main.lua":                 "ReturnOfModding/plugins/Ns-Mod/main.lua",
			"custom/file.txt":          "ReturnOfModding/plugins/Ns-Mod/custom/file.txt",
			"plugins/b.lua":            "ReturnOfModding/plugins/Ns-Mod/b.lua",
			"plugins_data/cache/d.dat": "ReturnOfModding/plugins_data/Ns-Mod/cache/d.dat",
			"config/c.cfg":             "ReturnOfModding/config/Ns-Mod/c.cfg",
		}},
		// The package's own mod folder is the shallowest one holding a manifest; the top manifest is Thunderstore's.
		{"gdweave", []string{"manifest.json", "GDWeave/mods/Fish/manifest.json", "GDWeave/mods/Fish/Fish.dll", "GDWeave/mods/Fish/sub/a.json", "other/readme.txt"}, nil, map[string]string{
			"GDWeave/mods/Fish/manifest.json": "GDWeave/mods/Ns-Mod/manifest.json",
			"GDWeave/mods/Fish/Fish.dll":      "GDWeave/mods/Ns-Mod/Fish.dll",
			"GDWeave/mods/Fish/sub/a.json":    "GDWeave/mods/Ns-Mod/sub/a.json",
		}},
		{"gdweave", []string{"manifest.json", "a.txt"}, nil, map[string]string{}},
		{"lovely", []string{"icon.png", "lovely.toml", "src/main.lua"}, nil, map[string]string{
			"lovely.toml":  "mods/Ns-Mod/lovely.toml",
			"src/main.lua": "mods/Ns-Mod/src/main.lua",
		}},
		{"rivet", []string{"manifest.json", "Mod.dll", "data/a.txt"}, nil, map[string]string{
			"Mod.dll":    "Rivet/Mods/Ns-Mod/Mod.dll",
			"data/a.txt": "Rivet/Mods/Ns-Mod/data/a.txt",
		}},
		{"umm", []string{"manifest.json", "Mod.dll", "Info.json", "BroMaker_Storage/x/y.json"}, umm, map[string]string{
			"Mod.dll":                   "UMM/Mods/Ns-Mod/Mod.dll",
			"Info.json":                 "UMM/Mods/Ns-Mod/Info.json",
			"BroMaker_Storage/x/y.json": "UMM/BroMaker_Storage/Ns-Mod/x/y.json",
		}},
		{"shimloader", []string{"manifest.json", "pak/a.pak", "mod/scripts/main.lua", "cfg/p.cfg", "loose.pak", "other/x.lua", "overlay/o.txt"}, shim, map[string]string{
			"pak/a.pak":            "shimloader/pak/Ns-Mod/a.pak",
			"mod/scripts/main.lua": "shimloader/mod/Ns-Mod/scripts/main.lua",
			"cfg/p.cfg":            "shimloader/cfg/p.cfg",
			"loose.pak":            "shimloader/pak/Ns-Mod/loose.pak",
			"other/x.lua":          "shimloader/mod/Ns-Mod/x.lua",
			"overlay/o.txt":        "shimloader/overlay/Ns-Mod/o.txt",
		}},
		{"northstar", []string{"manifest.json", "mods/Skin/mod.json", "mods/Skin/a/b.txt", "readme.txt"}, northstar, map[string]string{
			"mods/Skin/mod.json": "R2Northstar/mods/Skin/mod.json",
			"mods/Skin/a/b.txt":  "R2Northstar/mods/Skin/a/b.txt",
		}},
	}
	for _, c := range cases {
		got := Layout(c.loader, c.files, "Ns-Mod", c.routes)
		if len(got) != len(c.want) {
			t.Errorf("%s %v: got %v, want %v", c.loader, c.files, got, c.want)
			continue
		}
		for src, dest := range c.want {
			if got[src] != dest {
				t.Errorf("%s %s: got %q, want %q", c.loader, src, got[src], dest)
			}
		}
	}
}

// packOf is a stand-in release zip in the shape of the real Thunderstore package of each loader.
func packOf(id string) (zip map[string]string, version string) {
	switch id {
	case "returnofmodding":
		return map[string]string{"manifest.json": "{}", "icon.png": "i", "ReturnOfModdingPack/d3d12.dll": "rom"}, "1.0.115"
	case "gdweave":
		return map[string]string{"manifest.json": "{}", "winmm.dll": "w", "GDWeave/core/GDWeave.dll": "core", "GDWeave/core/Iced.dll": "iced"}, "2.0.14"
	case "lovely":
		return map[string]string{"manifest.json": "{}", "winmm.dll": "l", "lovely/config.toml": "default"}, "0.10.0"
	case "rivet":
		return map[string]string{"manifest.json": "{}", "RivetPack/version.dll": "v", "RivetPack/Rivet.ini": "ini", "RivetPack/Rivet/Loader.dll": "loader"}, "0.1.17"
	case "shimloader":
		return map[string]string{"manifest.json": "{}", "dwmapi.dll": "proxy", "UE4SS/dwmapi.dll": "unused", "UE4SS/LICENSE": "l", "UE4SS/UE4SS.dll": "ue4ss", "UE4SS/UE4SS-settings.ini": "ini", "UE4SS/Mods/shared/UEHelpers/UEHelpers.lua": "lua"}, "1.1.7"
	case "umm":
		return map[string]string{"manifest.json": "{}", "winhttp.dll": "w", "doorstop_config.ini": "c", ".doorstop_version": "4.4.0", "doorstop_libs/libdoorstop_x64.so": "so", "run_umm.sh": "sh", "UMM/Core/UnityModManager.dll": "umm"}, "1.1.0"
	}
	return map[string]string{
		"manifest.json": "{}", "Northstar/NorthstarLauncher.exe": "exe", "Northstar/NorthstarLauncher.pdb": "pdb",
		"Northstar/R2Northstar/Northstar.dll": "dll", "Northstar/bin/x64_retail/wsock32.dll": "ws",
		"Northstar/R2Northstar/mods/Northstar.Client/mod.json": "client",
	}, "1.31.14"
}

func TestInstallAndLaunch(t *testing.T) {
	cases := []struct {
		loader, game string
		runtime      string
		wantArgs     func(profile string) []string
		wantDst      []string
		wantReqs     []string
	}{
		{"returnofmodding", "hades-ii", "proton", func(p string) []string { return []string{"--rom_modding_root_folder", `Z:` + backslash(p)} }, []string{"Ship/d3d12.dll"}, []string{"d3d12"}},
		{"gdweave", "webfishing", "proton", func(p string) []string {
			return []string{"--gdweave-folder-override=Z:" + backslash(filepath.Join(p, "GDWeave"))}
		}, []string{"winmm.dll"}, []string{"winmm"}},
		{"rivet", "scrap-mechanic", "proton", func(p string) []string {
			return []string{"-rivetEnable", "true", "-rivetTarget", "Z:" + backslash(filepath.Join(p, "Rivet", "Loader.dll")), "-rivetDirectory", "Z:" + backslash(filepath.Join(p, "Rivet", "Mods"))}
		}, []string{"Release/version.dll"}, []string{"version"}},
		{"umm", "broforce", "", func(p string) []string {
			target := filepath.Join(p, "UMM", "Core", "UnityModManager.dll")
			return []string{"--doorstop-enabled", "true", "--doorstop-target-assembly", target, "--doorstop-enable", "true", "--doorstop-target", target}
		}, []string{"doorstop_config.ini", "winhttp.dll"}, nil},
		{"shimloader", "palworld", "proton", func(p string) []string {
			var out []string
			for _, f := range []string{"mod", "pak", "cfg", "overlay"} {
				out = append(out, "--"+f+"-dir", "Z:"+backslash(filepath.Join(p, "shimloader", f)))
			}
			return out
		}, []string{"Pal/Binaries/Win64/UE4SS-settings.ini", "Pal/Binaries/Win64/dwmapi.dll", "Pal/Binaries/Win64/ue4ss.dll"}, []string{"dwmapi"}},
		{"lovely", "balatro", "", func(p string) []string { return []string{"--mod-dir", filepath.Join(p, "mods")} }, []string{"winmm.dll"}, nil},
		{
			"northstar", "titanfall-2", "mortar-proton", func(p string) []string {
				return []string{"-northstar", "-profile=Z:" + backslash(filepath.Join(p, "R2Northstar"))}
			},
			[]string{"NorthstarLauncher.exe", "R2Northstar/Northstar.dll", "bin/x64_retail/wsock32.dll"},
			[]string{"wsock32"},
		},
	}
	for _, c := range cases {
		t.Run(c.loader, func(t *testing.T) {
			l, ok := loader.Get(c.loader)
			if !ok {
				t.Fatal("not registered")
			}
			files, version := packOf(c.loader)
			zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "pack.zip"), files)
			dir := filepath.Join(t.TempDir(), "profile")
			got, err := l.Install(t.Context(), loader.Target{Game: c.game, ProfileDir: dir}, loader.Package{Archive: zip, Version: version}, nil)
			if err != nil || got != version {
				t.Fatalf("install = %q, %v", got, err)
			}
			if st, _ := l.Status(loader.Target{ProfileDir: dir}); !st.Installed || st.Version != version {
				t.Fatalf("status = %+v", st)
			}
			plan := launchplan.New(launchplan.ModeProfile)
			view := loader.ProfileView{Game: c.game, Dir: dir, InstallDir: filepath.Join(t.TempDir(), "install"), Platform: "windows", Runtime: c.runtime}
			if err := l.Contribute(t.Context(), plan, view); err != nil {
				t.Fatal(err)
			}
			if want := c.wantArgs(dir); !slices.Equal(plan.Args, want) {
				t.Errorf("args = %v, want %v", plan.Args, want)
			}
			var dst []string
			for _, f := range plan.Files {
				dst = append(dst, filepath.ToSlash(f.Dst))
			}
			slices.Sort(dst)
			if !slices.Equal(dst, c.wantDst) {
				t.Errorf("placed %v, want %v", dst, c.wantDst)
			}
			var reqs []string
			for _, r := range plan.RuntimeReqs {
				reqs = append(reqs, r.Key)
			}
			if !slices.Equal(reqs, c.wantReqs) {
				t.Errorf("overrides %v, want %v", reqs, c.wantReqs)
			}
		})
	}
}

func TestAnUpdateKeepsWhatThePlayerHasAndDropsWhatTheReleaseDid(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "profile")
	install := func(id string, files map[string]string, version string) {
		t.Helper()
		l, _ := loader.Get(id)
		zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "pack.zip"), files)
		if _, err := l.Install(t.Context(), loader.Target{ProfileDir: dir}, loader.Package{Archive: zip, Version: version}, nil); err != nil {
			t.Fatal(err)
		}
	}
	have := func(rel string) bool { return fsx.IsFile(filepath.Join(dir, filepath.FromSlash(rel))) }

	// Lovely 0.9 shipped version.dll; the player edited the config and added a mod.
	install("lovely", map[string]string{"version.dll": "old", "lovely/config.toml": "default"}, "0.9.0")
	testfs.WriteFile(t, dir, "mods/lovely/config.toml", "edited")
	testfs.WriteFile(t, dir, "mods/Mine/lovely.toml", "mod")
	install("lovely", map[string]string{"winmm.dll": "new", "lovely/config.toml": "default"}, "0.10.0")
	if have("version.dll") || !have("winmm.dll") || !have("mods/Mine/lovely.toml") {
		t.Fatal("the old proxy must go, the new one come, and the player's mod stay")
	}
	if b, _ := fsx.ReadFile(filepath.Join(dir, "mods", "lovely", "config.toml")); string(b) != "edited" {
		t.Fatalf("config = %q: the player's edit must survive an update", b)
	}

	// GDWeave's core is the release's; its mods folder is the player's.
	install("gdweave", map[string]string{"winmm.dll": "w", "GDWeave/core/Gone.dll": "x", "GDWeave/core/GDWeave.dll": "1"}, "2.0.13")
	testfs.WriteFile(t, dir, "GDWeave/mods/Fish/Fish.dll", "mod")
	install("gdweave", map[string]string{"winmm.dll": "w", "GDWeave/core/GDWeave.dll": "2"}, "2.0.14")
	if have("GDWeave/core/Gone.dll") || !have("GDWeave/core/GDWeave.dll") || !have("GDWeave/mods/Fish/Fish.dll") {
		t.Fatal("a core file the release dropped must go and a mod must stay")
	}
}

func TestShimloaderLaysUEOut(t *testing.T) {
	l, _ := loader.Get("shimloader")
	files, version := packOf("shimloader")
	dir := filepath.Join(t.TempDir(), "profile")
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "pack.zip"), files)
	if _, err := l.Install(t.Context(), loader.Target{ProfileDir: dir}, loader.Package{Archive: zip, Version: version}, nil); err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]string{"dwmapi.dll": "proxy", "ue4ss.dll": "ue4ss", "UE4SS-settings.ini": "ini", "shimloader/mod/shared/UEHelpers/UEHelpers.lua": "lua"} {
		if b, _ := fsx.ReadFile(filepath.Join(dir, filepath.FromSlash(rel))); string(b) != want {
			t.Errorf("%s = %q, want %q", rel, b, want)
		}
	}
	for _, rel := range []string{"UE4SS", "shimloader/pak", "shimloader/cfg", "shimloader/overlay"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); (rel == "UE4SS") == (err == nil) {
			t.Errorf("%s: stat error %v", rel, err)
		}
	}
}

func backslash(p string) string {
	out := []byte(p)
	for i, c := range out {
		if c == '/' {
			out[i] = '\\'
		}
	}
	return string(out)
}

func TestNorthstarIsNotStartedThroughSteamsProton(t *testing.T) {
	l, _ := loader.Get("northstar")
	files, version := packOf("northstar")
	dir := filepath.Join(t.TempDir(), "profile")
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "pack.zip"), files)
	if _, err := l.Install(t.Context(), loader.Target{ProfileDir: dir}, loader.Package{Archive: zip, Version: version}, nil); err != nil {
		t.Fatal(err)
	}
	view := loader.ProfileView{Game: "titanfall-2", Dir: dir, InstallDir: t.TempDir(), Platform: "windows", Runtime: "proton"}
	if err := l.Contribute(t.Context(), launchplan.New(launchplan.ModeProfile), view); err == nil {
		t.Fatal("Steam's Proton relay cannot start another executable, so the launch must be refused")
	}
}

func TestAMappedDestinationOutsideTheProfileIsRefused(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "profile")
	l := Loader{Name: "Escape", Map: func(rel string) string { return "../" + rel }}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "pack.zip"), map[string]string{"x": "1"})
	if _, err := l.Install(t.Context(), loader.Target{ProfileDir: dir}, loader.Package{Archive: zip, Version: "1"}, nil); err == nil {
		t.Fatal("a destination below ../ must be refused")
	} else if !strings.Contains(err.Error(), "outside the profile") {
		t.Fatalf("refused for another reason: %v", err)
	}
	if fsx.IsFile(filepath.Join(root, "x")) {
		t.Fatal("the file escaped the profile")
	}
}
