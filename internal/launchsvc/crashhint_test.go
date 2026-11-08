package launchsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestPickCulpritBlamesTheOwnerOfTheFirstNamedPlugin(t *testing.T) {
	owners := map[string]profile.Installed{"fixture plugin": {Key: "b"}, "com.fixture.hard": {Key: "c"}}
	found := []loader.Finding{
		{Plugin: "Stranger", Message: "no installed mod owns it"},
		{Plugin: "Fixture Plugin", Message: "load failed"},
		{Plugin: "com.fixture.hard", Message: "later"},
	}
	m, f, ok := pickCulprit(found, owners)
	if !ok || m.Key != "b" || f.Message != "load failed" {
		t.Fatalf("culprit = %+v, %+v, %v", m, f, ok)
	}
	if _, _, ok := pickCulprit([]loader.Finding{{Plugin: "BepInEx"}}, owners); ok {
		t.Fatal("a plugin that is no installed mod blames nothing")
	}
}

// A log names a plugin exactly; a name that only reads alike is a different mod, and the Problems tab does not blame it.
func TestPickCulpritNeedsTheOwnerLookupsExactName(t *testing.T) {
	owners := map[string]profile.Installed{"lethallib": {Key: "a"}}
	for _, plugin := range []string{"Lethal Lib", "evaisa.lethallib", "lethallib.extended"} {
		if m, _, ok := pickCulprit([]loader.Finding{{Plugin: plugin}}, owners); ok {
			t.Fatalf("plugin %q blamed %q", plugin, m.Key)
		}
	}
	if m, _, ok := pickCulprit([]loader.Finding{{Plugin: "LethalLib"}}, owners); !ok || m.Key != "a" {
		t.Fatalf("culprit = %+v, %v", m, ok)
	}
}

// The owners are the ones Problems uses: the GUIDs and names a package's own DLLs declare, never a name the log's
// plugin merely resembles, so a log line names the same mod on both screens.
func TestPickCulpritUsesThePluginsOwnersLookup(t *testing.T) {
	dll, err := safefs.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Plugin.dll"), dll, 0o600); err != nil {
		t.Fatal(err)
	}
	packages := []profile.Installed{{Key: "fixture", Folder: dir, Enabled: true}}
	owners := loader.PluginOwners(packages,
		func(m profile.Installed) string { return m.Folder },
		func(m profile.Installed) bool { return m.Enabled })
	m, f, ok := pickCulprit([]loader.Finding{{Plugin: "Fixture Plugin 1.2.3", Message: "load failed"}, {Plugin: "com.fixture.plugin"}}, owners)
	if !ok || m.Key != "fixture" || f.Message != "load failed" {
		t.Fatalf("culprit = %+v, %+v, %v", m, f, ok)
	}
	if _, _, ok := pickCulprit([]loader.Finding{{Plugin: "com.fixture.hard"}}, owners); ok {
		t.Fatal("a plugin no package's DLL declares blames nothing")
	}
}
