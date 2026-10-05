package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestPluginClashNamesTheGUIDAndKeepsTheNewestPlugin(t *testing.T) {
	seed := func(dir string, p ...dotnet.Plugin) {
		pluginMu.Lock()
		pluginCache[pluginKey{dir}] = p
		pluginMu.Unlock()
	}
	seed("/a", dotnet.Plugin{GUID: "com.x.Cheats", Version: "1.2.0"}, dotnet.Plugin{GUID: "com.x.Solo", Version: "1.0.0"})
	seed("/b", dotnet.Plugin{GUID: "COM.X.cheats", Version: "1.10.0"})
	seed("/c", dotnet.Plugin{GUID: "com.y.Other", Version: "1.0.0"})
	got := pluginClashes([]profile.PackageRef{
		{Key: "a", Name: "Ann-Cheats", Version: "1.2.0", Dir: "/a"},
		{Key: "b", Name: "Bob-Cheats", Version: "1.10.0", Dir: "/b"},
		{Key: "c", Name: "Cy-Other", Version: "1.0.0", Dir: "/c"},
	})
	if len(got) != 1 || got[0].GUID != "com.x.Cheats" || len(got[0].Copies) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Keep != "b" {
		t.Fatalf("1.10.0 is newer than 1.2.0, keep %q", got[0].Keep)
	}
}
