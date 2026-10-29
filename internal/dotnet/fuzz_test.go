package dotnet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// FuzzAssembly feeds plugin DLLs, as a downloaded mod ships them, through each metadata reader: the BepInEx plugin
// scan every install runs, the member-write, member, usage, MelonLoader and Harmony patch scans, the identity a ran error's stack is matched by,
// and the source line read from an embedded PDB.
func FuzzAssembly(f *testing.F) {
	dll, err := fsx.ReadFile(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		f.Fatal(err)
	}
	embedded, err := fsx.ReadFile(filepath.Join("testdata", "embedded.dll"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(dll)
	f.Add(dll[:len(dll)/2])
	f.Add(embedded)
	f.Add([]byte("MZ"))
	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), "mod.dll")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		_, _ = Scan(path)
		_, _ = Writes(path, "StardewValley", "Netcode")
		_, _ = Patches(path)
		_, _ = Members(path)
		_, _ = Usages(path, "StardewValley.Farmer::currentToolIndex")
		_, _ = ScanMelon(path)
		_, _ = Identity(path)
		_ = SourceLine(path, "Fixture.Mod::SwitchTool", -1, 0x1c)
	})
}

// FuzzPluginArgs feeds the BepInPlugin attribute blob directly, which mutating a whole assembly rarely reaches.
func FuzzPluginArgs(f *testing.F) {
	f.Add([]byte{1, 0, 8, 'a', '.', 'b', '.', 'c', 'd', 'e', 'f', 4, 'N', 'a', 'm', 'e', 5, '1', '.', '0', '.', '0', 0, 0})
	f.Add([]byte{1, 0, 0xFF, 0xFF})
	f.Add([]byte{1, 0, 0xC0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) {
		p, ok := pluginArgs(b)
		if ok && len(p.GUID)+len(p.Name)+len(p.Version) > len(b) {
			t.Fatalf("plugin %+v is longer than its %d-byte blob", p, len(b))
		}
	})
}

// FuzzRelationArgs feeds the BepInDependency and BepInIncompatibility blobs directly.
func FuzzRelationArgs(f *testing.F) {
	f.Add([]byte{1, 0, 1, 'g', 1, 0, 0, 0, 0, 0}, false)
	f.Add([]byte{1, 0, 1, 'g', 3, '1', '.', '0', 0, 0}, false)
	f.Add([]byte{1, 0, 1, 'g', 0, 0}, true)
	f.Add([]byte{1, 0, 0xFF, 0xFF}, false)
	f.Fuzz(func(t *testing.T, b []byte, incompatible bool) {
		r, ok := relationArgs(b, incompatible)
		if ok && len(r.GUID)+len(r.MinVersion) > len(b) {
			t.Fatalf("relation %+v is longer than its %d-byte blob", r, len(b))
		}
	})
}

// FuzzPortablePDB feeds a PDB as a mod ships it beside its DLL to the sequence point reader, with the id the fixture
// DLL expects so mutations get past the stale check.
func FuzzPortablePDB(f *testing.F) {
	pdb, err := fsx.ReadFile(filepath.Join("testdata", "portable", "Mod.pdb"))
	if err != nil {
		f.Fatal(err)
	}
	dll, err := fsx.ReadFile(filepath.Join("testdata", "portable", "Mod.dll"))
	if err != nil {
		f.Fatal(err)
	}
	m, err := open(dll)
	if err != nil {
		f.Fatal(err)
	}
	id := m.debugEntry(debugCodeView)[4:20]
	f.Add(pdb, 1, 0x1c)
	f.Add(pdb[:len(pdb)/2], 3, 0)
	f.Fuzz(func(t *testing.T, data []byte, row, il int) {
		_, _ = func() (s string, err error) {
			defer malformed("Mod.pdb", &s, &err)
			return sequenceLine(data, id, row, il)
		}()
	})
}
