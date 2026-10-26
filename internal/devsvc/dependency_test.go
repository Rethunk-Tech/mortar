package devsvc

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestWithDependency(t *testing.T) {
	listed := "<Project>\n  <ItemGroup>\n    <ThunderstoreDependency Include=\"BepInEx-BepInExPack\" Version=\"5.4.2305\" />\n  </ItemGroup>\n</Project>\n"
	hard := withDependency(listed, "Evaisa-LethalLib", "1.1.1", false)
	want := "<Project>\n  <ItemGroup>\n    <ThunderstoreDependency Include=\"BepInEx-BepInExPack\" Version=\"5.4.2305\" />\n" +
		"    <ThunderstoreDependency Include=\"Evaisa-LethalLib\" Version=\"1.1.1\" />\n" +
		"    <Reference Include=\"$(MortarDependency_Evaisa_LethalLib)**/*.dll\" Condition=\"'$(MortarDependency_Evaisa_LethalLib)' != ''\" Private=\"false\" />\n  </ItemGroup>\n</Project>\n"
	if hard != want {
		t.Fatalf("hard:\n%s", hard)
	}
	// Made soft, it leaves the manifest's items but keeps its one reference.
	soft := withDependency(hard, "Evaisa-LethalLib", "1.2.0", true)
	if !strings.Contains(soft, `<ThunderstoreSoftDependency Include="Evaisa-LethalLib" Version="1.2.0" />`) ||
		strings.Contains(soft, `<ThunderstoreDependency Include="Evaisa-LethalLib"`) || strings.Count(soft, "<Reference") != 1 {
		t.Fatalf("soft:\n%s", soft)
	}
	if deps := projectDependencies(soft); !deps["Evaisa-LethalLib"] || deps["BepInEx-BepInExPack"] {
		t.Fatalf("dependencies = %v", deps)
	}
	if fresh := withDependency("<Project>\n</Project>\n", "A-B", "1.0.0", false); !strings.Contains(fresh, "  <ItemGroup>\n    <ThunderstoreDependency Include=\"A-B\" Version=\"1.0.0\" />\n    <Reference") {
		t.Fatalf("fresh:\n%s", fresh)
	}
}

// A soft dependency is declared with the soft flag beside [BepInPlugin] and gets its compatibility class, once.
func TestDeclareSoftDependency(t *testing.T) {
	src := t.TempDir()
	plugin := filepath.Join(src, "Plugin.cs")
	if err := fsx.WriteFile(plugin, []byte("namespace Shop;\n\n[BepInPlugin(PluginInfo.Id, PluginInfo.Name, PluginInfo.Version)]\npublic class Plugin {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := declareDependency(src, "evaisa.lethallib", false); err != nil {
		t.Fatal(err)
	}
	if err := declareDependency(src, "evaisa.lethallib", true); err != nil {
		t.Fatal(err)
	}
	got, _ := fsx.ReadFile(plugin)
	if want := "namespace Shop;\n\n[BepInDependency(\"evaisa.lethallib\", BepInDependency.DependencyFlags.SoftDependency)]\n[BepInPlugin("; !strings.HasPrefix(string(got), want) {
		t.Fatalf("plugin:\n%s", got)
	}
	if err := writeCompat(src, "Shop", "Evaisa-LethalLib", "evaisa.lethallib"); err != nil {
		t.Fatal(err)
	}
	compat := filepath.Join(src, "Compat", "LethalLibCompat.cs")
	body, err := fsx.ReadFile(compat)
	if err != nil || !strings.Contains(string(body), `Chainloader.PluginInfos.ContainsKey("evaisa.lethallib")`) || !strings.Contains(string(body), "MethodImplOptions.NoInlining") || !strings.Contains(string(body), "namespace Shop.Compat;\n") {
		t.Fatalf("compat %v:\n%s", err, body)
	}
	if err := fsx.WriteFile(compat, []byte("edited"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeCompat(src, "Shop", "Evaisa-LethalLib", "evaisa.lethallib"); err != nil {
		t.Fatal(err)
	}
	if body, _ := fsx.ReadFile(compat); string(body) != "edited" {
		t.Fatal("the player's compatibility class was overwritten")
	}
}

func TestSetUserDependencyListsEachPackageOnce(t *testing.T) {
	dir := t.TempDir()
	for range 2 {
		if err := setUserDependency(dir, "AinaVT-LethalConfig", "/store/lc/"); err != nil {
			t.Fatal(err)
		}
	}
	if err := setUserDependency(dir, "Evaisa-LethalLib", "/store/ll/"); err != nil {
		t.Fatal(err)
	}
	body := read(t, filepath.Join(dir, "Config.Build.user.props"))
	if strings.Count(body, `<MortarDependencyFolder Include="AinaVT-LethalConfig" />`) != 1 ||
		strings.Count(body, `<MortarDependencyFolder Include="Evaisa-LethalLib" />`) != 1 ||
		!strings.Contains(body, "<MortarDependency_AinaVT_LethalConfig>/store/lc/</MortarDependency_AinaVT_LethalConfig>") ||
		strings.Count(body, "<ItemGroup>") != 1 {
		t.Errorf("props = %s", body)
	}
}
