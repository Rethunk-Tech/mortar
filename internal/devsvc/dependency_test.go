package devsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/settings"
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
		if err := setUserDependency(dir, "AinaVT-LethalConfig", "/store/l;c/"); err != nil {
			t.Fatal(err)
		}
	}
	if err := setUserDependency(dir, "Evaisa-LethalLib", "/store/ll/"); err != nil {
		t.Fatal(err)
	}
	body := read(t, filepath.Join(dir, "Config.Build.user.props"))
	if strings.Count(body, `<MortarDependencyFolder Include="AinaVT-LethalConfig" />`) != 1 ||
		strings.Count(body, `<MortarDependencyFolder Include="Evaisa-LethalLib" />`) != 1 ||
		!strings.Contains(body, "<MortarDependency_AinaVT_LethalConfig>/store/l%3Bc/</MortarDependency_AinaVT_LethalConfig>") ||
		strings.Count(body, "<ItemGroup>") != 1 {
		t.Errorf("props = %s", body)
	}
}

func TestWireFoldsOnlyItsOwnChangesIntoTheStartCommit(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	svc.Run = nil
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := svc.run(t.Context(), dir, "git", args...)
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q")
	write("Plugin.cs", "a")
	git("add", "-A")
	git("commit", "-q", "-m", "Start Mod")
	write("Mine.cs", "player's own")
	wired := func() error { write("Compat.cs", "c"); write("Plugin.cs", "b"); return nil }
	if err := svc.wire(t.Context(), dir, wired); err != nil {
		t.Fatal(err)
	}
	if n, status := git("rev-list", "--count", "HEAD"), git("status", "--porcelain"); n != "1" || status != "?? Mine.cs" {
		t.Fatalf("after wiring: %s commits, status %q", n, status)
	}
	// Once the player commits, wiring stays a change for them to commit.
	git("add", "Mine.cs")
	git("commit", "-q", "-m", "Mine")
	if err := svc.wire(t.Context(), dir, func() error { write("Plugin.cs", "d"); return nil }); err != nil {
		t.Fatal(err)
	}
	if status := git("status", "--porcelain"); status != "M Plugin.cs" {
		t.Fatalf("after the player's commit, status %q", status)
	}
}

func TestSyncReferencesADependencyAnotherProjectBuilds(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	prof, err := svc.Profiles.Create("lethal-company", "Dev")
	if err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	lib, mod := t.TempDir(), t.TempDir()
	write(filepath.Join(lib, propsFile), "<Project><PropertyGroup><ThunderstoreTeam>Mortar</ThunderstoreTeam></PropertyGroup></Project>")
	write(filepath.Join(lib, "src", "ConfigSync", "ConfigSync.csproj"), "<Project><PropertyGroup><ThunderstoreName>ConfigSync</ThunderstoreName></PropertyGroup></Project>")
	csproj := filepath.Join(mod, "src", "Mod", "Mod.csproj")
	write(csproj, "<Project>\n  <ItemGroup>\n    <ThunderstoreDependency Include=\"Mortar-ConfigSync\" Version=\"0.1.0\" />\n  </ItemGroup>\n</Project>\n")
	output := filepath.Join(lib, "artifacts", "bin")
	for _, p := range []settings.DevProject{{Dir: lib, Name: "ConfigSync", Output: output}, {Dir: mod, Name: "Mod", Profile: prof.ID}} {
		if err := svc.Settings.AddDevProject("lethal-company", p); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.syncDependencies("lethal-company", settings.DevProject{Dir: mod, Name: "Mod", Profile: prof.ID}); err != nil {
		t.Fatal(err)
	}
	project, _ := fsx.ReadFile(csproj)
	props, _ := fsx.ReadFile(filepath.Join(mod, "Config.Build.user.props"))
	if !strings.Contains(string(project), "$(MortarDependency_Mortar_ConfigSync)**/*.dll") ||
		!strings.Contains(string(props), "<MortarDependency_Mortar_ConfigSync>"+output+string(filepath.Separator)+"<") {
		t.Fatalf("project:\n%s\nprops:\n%s", project, props)
	}
}
