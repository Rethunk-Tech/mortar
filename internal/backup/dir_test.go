package backup

import (
	"path/filepath"
	"testing"
)

func TestLocationsReadsDefaultAfterCustomWriteDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	custom := filepath.Join(t.TempDir(), "elsewhere")
	write, reads, err := Locations(custom)
	if err != nil {
		t.Fatal(err)
	}
	if write != custom {
		t.Fatalf("write = %q", write)
	}
	if len(reads) != 2 || reads[0] != custom {
		t.Fatalf("reads = %q", reads)
	}
	def, _, err := Locations("")
	if err != nil {
		t.Fatal(err)
	}
	if reads[1] != def {
		t.Fatalf("old folder %q missing from %q", def, reads)
	}
}
