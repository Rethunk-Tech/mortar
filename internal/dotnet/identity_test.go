package dotnet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

// The MVIDs are what System.Reflection.Metadata's ModuleVersionId.ToString("N") prints for each fixture, the form a
// Mono stack frame without symbols shows. A Windows PDB beside a DLL is told apart by its magic.
func TestIdentityReadsTheMVIDAndWhichSymbolsShip(t *testing.T) {
	dll, err := safefs.ReadFile(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	windows := filepath.Join(t.TempDir(), "Mod.dll")
	if err := os.WriteFile(windows, dll, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(windows), "Mod.pdb"), []byte("Microsoft C/C++ MSF 7.00\r\n\x1aDS\x00\x00\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ path, mvid, symbols string }{
		{filepath.Join("testdata", "mod.dll"), "5fbed16890e94601b0c1d6bd5787551f", ""},
		{filepath.Join("testdata", "embedded.dll"), "bae0cdbd54c4427189879800db2c4c0e", SymbolsPortable},
		{windows, "5fbed16890e94601b0c1d6bd5787551f", SymbolsWindows},
	} {
		if mvid, symbols := Identity(c.path); mvid != c.mvid || symbols != c.symbols {
			t.Errorf("%s: %q %q, want %q %q", c.path, mvid, symbols, c.mvid, c.symbols)
		}
	}
}
