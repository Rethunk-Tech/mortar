package installer

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const wrapCfg = `<config><moduleName>M</moduleName><requiredInstallFiles><folder source="Data" destination=""/></requiredInstallFiles></config>`

func TestFomodSourcesResolveBesideTheConfigAndIgnoreCase(t *testing.T) {
	for name, tc := range map[string]struct{ cfg, data string }{
		"wrapper": {"Wrap/fomod/ModuleConfig.xml", "Wrap/Data/a.txt"},
		"case":    {"fomod/ModuleConfig.xml", "data/a.txt"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			testfs.WriteAt(t, filepath.Join(dir, filepath.FromSlash(tc.cfg)), []byte(wrapCfg))
			testfs.WriteAt(t, filepath.Join(dir, filepath.FromSlash(tc.data)), []byte("x"))
			g := Game{Loaders: []string{"smapi"}, Targets: []Target{{ID: TargetMods}}}
			a := Open(dir, "k")
			i, ok := Pick(a, g)
			if !ok {
				t.Fatal("no installer")
			}
			l, err := i.Layout(a, g, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(l.Files) != 1 || l.Files[0].Src != tc.data || l.Files[0].Rel != "k/a.txt" {
				t.Fatalf("layout %+v", l.Files)
			}
		})
	}
}

// A package packed on Linux can hold Data/ and data/ side by side; a source the exact folder lacks is found in the other.
func TestFomodSourceBacktracksAcrossCaseVariantFolders(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Data/ and data/ cannot sit side by side on a case-insensitive Windows folder")
	}
	dir := t.TempDir()
	cfg := `<config><moduleName>M</moduleName><requiredInstallFiles><file source="Data/y.txt" destination="y.txt"/></requiredInstallFiles></config>`
	testfs.WriteAt(t, filepath.Join(dir, "fomod", "ModuleConfig.xml"), []byte(cfg))
	testfs.WriteAt(t, filepath.Join(dir, "Data", "x.txt"), []byte("x"))
	testfs.WriteAt(t, filepath.Join(dir, "data", "y.txt"), []byte("y"))
	g := Game{Loaders: []string{"smapi"}, Targets: []Target{{ID: TargetMods}}}
	a := Open(dir, "k")
	i, ok := Pick(a, g)
	if !ok {
		t.Fatal("no installer")
	}
	l, err := i.Layout(a, g, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Files) != 1 || l.Files[0].Src != "data/y.txt" || l.Files[0].Rel != "k/y.txt" {
		t.Fatalf("layout %+v", l.Files)
	}
}

func TestFomodRefusesSourceOrDestinationOutsideTheMod(t *testing.T) {
	for name, op := range map[string]string{
		"source parent":      `<file source="../x" destination="a"/>`,
		"source absolute":    `<file source="/x" destination="a"/>`,
		"source backslash":   `<file source="..\x" destination="a"/>`,
		"dest cleaned out":   `<file source="a.txt" destination="a/../../b"/>`,
		"dest backslash out": `<file source="a.txt" destination="..\x"/>`,
		"dest parent":        `<file source="a.txt" destination=".."/>`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := `<config><moduleName>M</moduleName><requiredInstallFiles>` + op + `</requiredInstallFiles></config>`
			testfs.WriteAt(t, filepath.Join(dir, "fomod", "ModuleConfig.xml"), []byte(cfg))
			testfs.WriteAt(t, filepath.Join(dir, "a.txt"), []byte("x"))
			g := Game{Loaders: []string{"smapi"}, Targets: []Target{{ID: TargetMods}}}
			a := Open(dir, "k")
			i, ok := Pick(a, g)
			if !ok {
				t.Fatal("no installer")
			}
			if _, err := i.Layout(a, g, nil); !errors.Is(err, ErrUnsafe) {
				t.Fatalf("err = %v, want ErrUnsafe", err)
			}
		})
	}
}
