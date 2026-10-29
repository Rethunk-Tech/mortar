package dotnet

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

// The expected lines are what System.Reflection.Metadata's sequence points give for testdata/src/Mod/Mod.cs: the
// last visible point at or before the offset, a lambda in a compiler-generated nested type, and none before the first
// visible point or in a method without points. A PDB whose id is not the DLL's is refused.
func TestSourceLineReadsPortableSymbols(t *testing.T) {
	stale := filepath.Join(t.TempDir(), "Mod.dll")
	dll, err := safefs.ReadFile(filepath.Join("testdata", "portable", "Mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	pdb, err := safefs.ReadFile(filepath.Join("testdata", "portable", "Mod.pdb"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := open(dll)
	if err != nil {
		t.Fatal(err)
	}
	id := m.debugEntry(debugCodeView)[4:20]
	at := bytes.Index(pdb, id)
	if at < 0 {
		t.Fatal("the PDB does not carry the DLL's id")
	}
	pdb = bytes.Clone(pdb)
	pdb[at] ^= 0xff
	if err := os.WriteFile(stale, dll, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(stale), "Mod.pdb"), pdb, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		dll, method string
		params, il  int
		want        string
	}{
		{"embedded.dll", "Fixture.Mod::SwitchTool", 2, 0x1c, "Mod.cs:18"},
		{"embedded.dll", "Fixture.Mod::SwitchTool", -1, 0x3d, "Mod.cs:24"},
		{"embedded.dll", "Fixture.Mod::SwitchTool", 1, 0x1c, ""},
		{"embedded.dll", "Fixture.Mod::SwitchTool", 2, 0x5, ""},
		{"portable/Mod.dll", "Fixture.Hooks.Hooks+<>c::<Apply>b__0_1", 1, 0, "Mod.cs:135"},
		{"portable/Mod.dll", "Fixture.Patches.FarmerPatches::SkipUpdate", 0, 0, "Mod.cs:85"},
		{"portable/Mod.dll", "Fixture.Plugins.Plugin::.ctor", 0, 0, ""},
		{"mod.dll", "Fixture.Mod::SwitchTool", 2, 0x1c, ""},
		{stale, "Fixture.Patches.FarmerPatches::SkipUpdate", 0, 0, ""},
	} {
		path := c.dll
		if !filepath.IsAbs(path) {
			path = filepath.Join("testdata", filepath.FromSlash(path))
		}
		if got := SourceLine(path, c.method, c.params, c.il); got != c.want {
			t.Errorf("%s %s @%#x = %q, want %q", c.dll, c.method, c.il, got, c.want)
		}
	}
}
