package devsvc

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// ConfigSync builds first; the mod that depends on it and an unrelated one build together after it, and a project
// depending on a package none of them publishes waits for nothing.
func TestBuildOrderPutsADependencyInAnEarlierWave(t *testing.T) {
	project := func(name, deps string) string {
		dir := t.TempDir()
		testfs.WriteAt(t, filepath.Join(dir, propsFile), []byte("<Project><PropertyGroup><ThunderstoreTeam>Mortar</ThunderstoreTeam></PropertyGroup></Project>"))
		testfs.WriteAt(t, filepath.Join(dir, "src", name, name+".csproj"), []byte("<Project><ItemGroup>"+deps+"</ItemGroup></Project>"))
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
