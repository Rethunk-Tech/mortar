package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestPluginClashNamesTheGUIDAndKeepsTheNewestPlugin(t *testing.T) {
	seed := func(dir string, p ...dotnet.Plugin) {
		pluginMu.Lock()
		pluginCache[pluginKey{dir}] = dotnet.Declared{Plugins: p}
		pluginMu.Unlock()
	}
	seed("/a", dotnet.Plugin{GUID: "com.x.Cheats", Version: "1.2.0"}, dotnet.Plugin{GUID: "com.x.Solo", Version: "1.0.0"})
	seed("/b", dotnet.Plugin{GUID: "COM.X.cheats", Version: "1.10.0"})
	seed("/c", dotnet.Plugin{GUID: "com.y.Other", Version: "1.0.0"})
	got := pluginClashes([]profile.PackageRef{
		{Key: "a", Name: "Ann-Cheats", Version: "1.2.0", Dir: "/a"},
		{Key: "b", Name: "Bob-Cheats", Version: "1.10.0", Dir: "/b"},
		{Key: "c", Name: "Cy-Other", Version: "1.0.0", Dir: "/c"},
	}, nil)
	if len(got) != 1 || got[0].GUID != "com.x.Cheats" || len(got[0].Copies) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Keep != "b" {
		t.Fatalf("1.10.0 is newer than 1.2.0, keep %q", got[0].Keep)
	}
}

func TestPluginClashKeepsTheCopyThatIsNotDeprecatedBetweenEqualVersions(t *testing.T) {
	for _, dir := range []string{"/old", "/new"} {
		pluginMu.Lock()
		pluginCache[pluginKey{dir}] = dotnet.Declared{Plugins: []dotnet.Plugin{{GUID: "OpenDoors", Version: "1.0.1"}}}
		pluginMu.Unlock()
	}
	got := pluginClashes([]profile.PackageRef{
		{Key: "old", Name: "Ann-OpenDoorsOld", Version: "1.0.1", Dir: "/old"},
		{Key: "new", Name: "Ann-OpenDoors", Version: "1.0.1", Dir: "/new"},
	}, map[string]bool{"old": true})
	if len(got) != 1 || got[0].Keep != "new" {
		t.Fatalf("got %+v", got)
	}
}
