package dotnet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// craftedDLL copies testdata/mod.dll, lets edit rewrite its metadata, and returns the file's path.
func craftedDLL(t *testing.T, edit func(m *module)) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := open(data)
	if err != nil {
		t.Fatal(err)
	}
	edit(m)
	path := filepath.Join(t.TempDir(), "mod.dll")
	if err := fsx.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// typeRefCycle points every TypeRef's resolution scope at the next TypeRef.
func typeRefCycle(m *module) {
	n := m.rows[tTypeRef]
	for r := 1; r <= n; r++ {
		p := m.start[tTypeRef] + (r-1)*m.rowSize[tTypeRef]
		v := (r%n+1)<<2 | 3
		m.tables[p], m.tables[p+1] = byte(v&0xFF), byte(v>>8&0xFF)
	}
}

// typeSpecSelf makes every TypeSpec's signature GENERICINST CLASS <that TypeSpec>.
func typeSpecSelf(m *module) {
	for r := 1; r <= m.rows[tTypeSpec] && r < 32; r++ {
		i := m.cell(tTypeSpec, r, 0)
		if n, size := compressed(m.blobs[i:]); n >= 3 {
			b := m.blobs[int(i)+size:]
			b[0], b[1], b[2] = 0x15, 0x12, byte((r<<2|2)&0xFF)
		}
	}
}

func TestATypeRefCycleIsMalformedNotAStackOverflow(t *testing.T) {
	path := craftedDLL(t, typeRefCycle)
	if _, err := Scan(path); err == nil || !strings.Contains(err.Error(), "malformed assembly") {
		t.Fatalf("Scan err = %v, want a malformed assembly", err)
	}
}

func TestATypeSpecNamingItselfIsMalformedNotAStackOverflow(t *testing.T) {
	path := craftedDLL(t, typeSpecSelf)
	if _, err := Writes(path, "StardewValley", "Netcode"); err == nil || !strings.Contains(err.Error(), "malformed assembly") {
		t.Fatalf("Writes err = %v, want a malformed assembly", err)
	}
}

func TestAnAssemblyOverTheSizeCapIsRefusedUnread(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.dll")
	if err := fsx.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, maxAssembly+1); err != nil {
		t.Fatal(err)
	}
	if _, err := Scan(path); err == nil || !strings.Contains(err.Error(), "over the") {
		t.Fatalf("Scan err = %v, want the size limit", err)
	}
	if mvid, _ := Identity(path); mvid != "" {
		t.Fatalf("Identity read %q from an oversized file", mvid)
	}
}
