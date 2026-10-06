package dotnet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// FuzzAssembly feeds plugin DLLs, as a downloaded mod ships them, through both metadata readers: the BepInEx
// plugin scan every install runs and the member-write scan.
func FuzzAssembly(f *testing.F) {
	dll, err := fsx.ReadFile(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(dll)
	f.Add(dll[:len(dll)/2])
	f.Add([]byte("MZ"))
	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), "mod.dll")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		_, _ = Scan(path)
		_, _ = Writes(path, "StardewValley", "Netcode")
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
