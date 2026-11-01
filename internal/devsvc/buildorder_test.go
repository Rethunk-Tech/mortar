package devsvc

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// ConfigSync builds first; the mod that depends on it and an unrelated one build together after it, and a project
// depending on a package none of them publishes waits for nothing.
func TestBuildOrderPutsADependencyInAnEarlierWave(t *testing.T) {
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	project := func(name, deps string) string {
		dir := t.TempDir()
		write(filepath.Join(dir, propsFile), "<Project><PropertyGroup><ThunderstoreTeam>Mortar</ThunderstoreTeam></PropertyGroup></Project>")
		write(filepath.Join(dir, "src", name, name+".csproj"), "<Project><ItemGroup>"+deps+"</ItemGroup></Project>")
		return dir
	}
	lib := project("ConfigSync", "")
	mod := project("Dropzone", `<ThunderstoreDependency Include="Mortar-ConfigSync" Version="0.1.1" />`)
	other := project("Posters", `<ThunderstoreDependency Include="BepInEx-BepInExPack" Version="5.4.2305" />`)
	got := (&Service{}).BuildOrder([]string{mod, other, lib})
	want := [][]string{{other, lib}, {mod}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("waves = %v, want %v", got, want)
	}
}
