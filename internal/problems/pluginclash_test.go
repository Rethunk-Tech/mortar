package problems

import (
	"os"
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
	}, nil, nil)
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
	}, map[string]bool{"old": true}, nil)
	if len(got) != 1 || got[0].Keep != "new" {
		t.Fatalf("got %+v", got)
	}
	// The copy the player builds is the one kept, even when the other is newer.
	pluginMu.Lock()
	pluginCache[pluginKey{"/new"}] = dotnet.Declared{Plugins: []dotnet.Plugin{{GUID: "OpenDoors", Version: "2.0.0"}}}
	pluginMu.Unlock()
	got = pluginClashes([]profile.PackageRef{
		{Key: "old", Name: "Me-OpenDoors", Dir: "/old"},
		{Key: "new", Name: "Ann-OpenDoors", Dir: "/new"},
	}, nil, map[string]bool{"old": true})
	if len(got) != 1 || got[0].Keep != "old" {
		t.Fatalf("developed copy not kept: %+v", got)
	}
}

func TestThePluginScanOfARemovedStoreFolderIsDropped(t *testing.T) {
	gone, kept := t.TempDir(), t.TempDir()
	declaredIn(gone)
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	declaredIn(kept)
	pluginMu.Lock()
	_, stale := pluginCache[pluginKey{gone}]
	_, fresh := pluginCache[pluginKey{kept}]
	pluginMu.Unlock()
	if stale || !fresh {
		t.Fatalf("removed folder kept %v, live folder kept %v", stale, fresh)
	}
}
