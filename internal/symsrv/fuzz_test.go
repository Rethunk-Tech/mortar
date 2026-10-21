package symsrv

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// FuzzWindowsPDB feeds a PDB as a symbol server, or anyone on the path of its plain-http address, could answer, and a
// CodeView record as a crash dump carries it.
func FuzzWindowsPDB(f *testing.F) {
	pdb, err := os.ReadFile(filepath.Join("testdata", fixturePDB))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(pdb, uint64(0x101c))
	f.Add(pdb[:len(pdb)/3], uint64(0x1050))
	f.Add([]byte("RSDS0123456789abcdef\x01\x00\x00\x00C:\\x\\ntdll.pdb\x00"), uint64(0))
	f.Fuzz(func(t *testing.T, data []byte, rva uint64) {
		_, _, _ = Lookup(bytes.NewReader(data), int64(len(data)), rva)
		if cv, ok := ParseCodeView(data); ok && !pdbName.MatchString(cv.PDB) {
			t.Fatalf("PDB name %q would leave the cache folder", cv.PDB)
		}
	})
}
