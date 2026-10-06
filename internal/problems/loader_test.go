package problems

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestLoaderFailuresAttributeLogFindingsToPackages(t *testing.T) {
	log, err := fsx.ReadFile(filepath.Join("..", "loader", "bepinex5", "testdata", "logoutput-missing-dependency.log"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	testfs.WriteFile(t, dir, "BepInEx/LogOutput.log", string(log))
	boombox := inst("boombox", "YoutubeBoombox", "1.0", true)
	player := "NullReferenceException: boom\n  at Boombox.Plugin.Update () [0x00000] in <abc>:0\n"
	known := map[string]framework.Mod{"youtube boombox 1.5.0": boombox}
	for name, owners := range map[string]map[string]framework.Mod{"known": known, "unknown": {}} {
		got := loaderFailures(bepinex5.Loader{}, loader.ProfileView{Dir: dir}, player, func() map[string]framework.Mod { return owners })
		if len(got) != 3 || got[0].Plugin != "Youtube Boombox 1.5.0" || got[0].Kind != bepinex5.KindMissingDependency {
			t.Fatalf("%s: failures = %+v", name, got)
		}
		wantKey := ""
		if name == "known" {
			wantKey = "boombox"
		}
		if got[0].Key != wantKey || got[0].Name != boombox.Name && wantKey != "" {
			t.Fatalf("%s: owner = %q %q", name, got[0].Key, got[0].Name)
		}
	}
}

func TestGameVersionFailureNeedsAnOlderGameThanTheLoaderAccepts(t *testing.T) {
	l, _ := game.PrimaryLoader("stardew")
	for _, c := range []struct {
		version string
		bad     bool
	}{{"1.5.6", true}, {"1.6.14", false}, {"", false}} {
		_, bad := gameVersionFailure("stardew", l, Environment{GameVersion: c.version, VersionScheme: deps.SemverSMAPI})
		if bad != c.bad {
			t.Errorf("game %q flagged = %v", c.version, bad)
		}
	}
}

func TestPluginOwnersNameAPackageByItsPluginsNamespace(t *testing.T) {
	dll, err := fsx.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	pkg := func(key string) framework.Mod {
		m := inst(key, key, "1.0.0", true)
		m.Folder = t.TempDir()
		if err := fsx.WriteFile(filepath.Join(m.Folder, "Plugin.dll"), dll, 0o600); err != nil {
			t.Fatal(err)
		}
		return m
	}
	one := pkg("one")
	if got := pluginOwners([]framework.Mod{one})["fixture"]; got.Key != "one" {
		t.Fatalf("the plugin class's root namespace names its package: %+v", got)
	}
	if got, ok := pluginOwners([]framework.Mod{one, pkg("two")})["fixture"]; ok {
		t.Fatalf("a namespace two packages share names neither: %+v", got)
	}
}
