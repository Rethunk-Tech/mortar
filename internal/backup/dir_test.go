package backup

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/saves"
)

func TestLocationsReadsDefaultAfterCustomWriteDir(t *testing.T) {
	data := t.TempDir()
	custom := filepath.Join(t.TempDir(), "elsewhere")
	write, reads, err := Locations(data, custom, "stardew")
	if err != nil {
		t.Fatal(err)
	}
	if write != filepath.Join(custom, "stardew") {
		t.Fatalf("write = %q", write)
	}
	if len(reads) != 2 || reads[0] != write {
		t.Fatalf("reads = %q", reads)
	}
	def, _, err := Locations(data, "", "stardew")
	if err != nil {
		t.Fatal(err)
	}
	if reads[1] != def {
		t.Fatalf("old folder %q missing from %q", def, reads)
	}
}

func TestLocationsKeepEachGameApart(t *testing.T) {
	data := t.TempDir()
	for _, custom := range []string{"", t.TempDir()} {
		_, a, _ := Locations(data, custom, "stardew")
		_, b, _ := Locations(data, custom, "lethal-company")
		for _, x := range a {
			for _, y := range b {
				if x == y {
					t.Fatalf("custom %q: both games read %q", custom, x)
				}
			}
		}
	}
}

// layout is a Stardew-shaped saves folder.
func layout(dir string) saves.Layout { return saves.Layout{Dir: dir} }

// folderSaves lays saves out as Stardew Valley does, one folder per save; List needs only its shape.
var folderSaves = saves.Layout{}
