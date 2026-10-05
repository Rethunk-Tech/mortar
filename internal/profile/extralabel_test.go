package profile

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

func TestExtraFileLabelUsesNexusFileTitle(t *testing.T) {
	e := Entry{
		Mods: []Component{{Name: "Main", ID: "smapi:A.Main", Folder: "."}},
	}
	files := []nexus.File{{FileID: 2, FileName: "Optional pack", Version: "1.2"}}
	got := ExtraFileLabel(e, "nexus-7-2", files)
	if got != "Optional pack (1.2)" {
		t.Fatalf("got %q", got)
	}
}

func TestExtraFileLabelUsesModFolders(t *testing.T) {
	e := Entry{
		Mods: []Component{
			{Name: "Part B", Version: "2.0", Folder: "extra-folder/PartB"},
		},
	}
	got := ExtraFileLabel(e, "extra-folder", nil)
	if got != "Part B (2.0)" {
		t.Fatalf("got %q", got)
	}
}
