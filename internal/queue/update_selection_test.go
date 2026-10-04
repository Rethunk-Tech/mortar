package queue

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

func TestChooseFileUpdateStaysInTheInstalledFileGroup(t *testing.T) {
	files := []nexus.File{
		{FileID: 9545, FileName: "EvenMoreSecretWoods - Content Patcher Version-2364-1-2.zip", Version: "1.2", Category: "MAIN"},
		{FileID: 9546, FileName: "EvenMoreSecretWoods - raw woods.xnb file-2364-1-2.zip", Version: "1.2", Category: "MAIN"},
	}

	got, ok := ChooseFile(files, "1.2", 9545)
	if !ok || got.FileID != 9545 {
		t.Fatalf("got file %#v, %v; want file 9545", got, ok)
	}
}

func TestNewestUpdateChoosesTheNewestSameStemFile(t *testing.T) {
	files := []nexus.File{
		{FileID: 9544, FileName: "Example.Mod-2364-1-1.zip", Version: "1.1", Category: "OLD_VERSION", ReplacedBy: 9545},
		{FileID: 9545, FileName: "Example.Mod-2364-1-2.zip", Version: "1.2", Category: "MAIN"},
		{FileID: 9546, FileName: "Example.Mod raw woods-2364-1-2.zip", Version: "1.2", Category: "MAIN"},
	}

	got := newestUpdate(files, files[0])
	if got.FileID != 9545 {
		t.Fatalf("got file %#v; want file 9545", got)
	}
}

// Nexus now names archives "<name> <mod id> <version> <upload time> <token>.zip", so stems differ across versions.
func TestChooseFileFollowsTheChainForTimestampedArchiveNames(t *testing.T) {
	files := []nexus.File{
		{FileID: 171060, Name: "Quest Helper", FileName: "Quest Helper 41150 0.3.3 2026-06-13T09-35Z OtbsrfvQn.zip", Version: "0.3.3", Category: "OLD_VERSION", ReplacedBy: 185334},
		{FileID: 185334, Name: "Quest Helper", FileName: "Quest Helper 41150 0.3.4 2026-10-03T06-09Z Pct02696C.zip", Version: "0.3.4", Category: "MAIN", IsPrimary: true},
	}
	got, ok := ChooseFile(files, "0.3.4", 171060)
	if !ok || got.FileID != 185334 {
		t.Fatalf("got file %#v, %v; want file 185334", got, ok)
	}
}

func TestSameFileGroupUsesDisplayNameAndChain(t *testing.T) {
	a := nexus.File{FileID: 1, Name: "Machine Control Panel", FileName: "Machine Control Panel 28261 2.4.1 2026-08-17T21-05Z cYC7hVFrv.zip"}
	b := nexus.File{FileID: 2, Name: "Machine Control Panel", FileName: "Machine Control Panel 28261 2.5.0 2026-10-03T12-28Z KAEi8abtd.zip"}
	if !sameFileGroup(a, b) {
		t.Fatal("same display name should be one group")
	}
	c := nexus.File{FileID: 3, Name: "Optional textures", FileName: "x.zip"}
	if sameFileGroup(a, c) {
		t.Fatal("different display names should be different groups")
	}
	a.ReplacedBy = 3
	if !sameFileGroup(a, c) {
		t.Fatal("the author's file_updates chain should join groups")
	}
}
