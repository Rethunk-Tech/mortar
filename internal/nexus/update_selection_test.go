package nexus

import (
	"testing"
)

func TestChooseFileUpdateStaysInTheInstalledFileGroup(t *testing.T) {
	files := []File{
		{FileID: 9545, FileName: "EvenMoreSecretWoods - Content Patcher Version-2364-1-2.zip", Version: "1.2", Category: "MAIN"},
		{FileID: 9546, FileName: "EvenMoreSecretWoods - raw woods.xnb file-2364-1-2.zip", Version: "1.2", Category: "MAIN"},
	}

	got, ok := ChooseFile(files, "1.2", 9545)
	if !ok || got.FileID != 9545 {
		t.Fatalf("got file %#v, %v; want file 9545", got, ok)
	}
}

func TestNewestUpdateChoosesTheNewestSameStemFile(t *testing.T) {
	files := []File{
		{FileID: 9544, FileName: "Example.Mod-2364-1-1.zip", Version: "1.1", Category: "OLD_VERSION", ReplacedBy: 9545},
		{FileID: 9545, FileName: "Example.Mod-2364-1-2.zip", Version: "1.2", Category: "MAIN"},
		{FileID: 9546, FileName: "Example.Mod raw woods-2364-1-2.zip", Version: "1.2", Category: "MAIN"},
	}

	got := NewestUpdate(files, files[0])
	if got.FileID != 9545 {
		t.Fatalf("got file %#v; want file 9545", got)
	}
}

// Nexus now names archives "<name> <mod id> <version> <upload time> <token>.zip", so stems differ across versions.
func TestChooseFileFollowsTheChainForTimestampedArchiveNames(t *testing.T) {
	files := []File{
		{FileID: 171060, Name: "Quest Helper", FileName: "Quest Helper 41150 0.3.3 2026-06-13T09-35Z OtbsrfvQn.zip", Version: "0.3.3", Category: "OLD_VERSION", ReplacedBy: 185334},
		{FileID: 185334, Name: "Quest Helper", FileName: "Quest Helper 41150 0.3.4 2026-10-03T06-09Z Pct02696C.zip", Version: "0.3.4", Category: "MAIN", IsPrimary: true},
	}
	got, ok := ChooseFile(files, "0.3.4", 171060)
	if !ok || got.FileID != 185334 {
		t.Fatalf("got file %#v, %v; want file 185334", got, ok)
	}
}

func TestSameFileGroupUsesDisplayNameAndChain(t *testing.T) {
	a := File{FileID: 1, Name: "Machine Control Panel", FileName: "Machine Control Panel 28261 2.4.1 2026-08-17T21-05Z cYC7hVFrv.zip"}
	b := File{FileID: 2, Name: "Machine Control Panel", FileName: "Machine Control Panel 28261 2.5.0 2026-10-03T12-28Z KAEi8abtd.zip"}
	if !sameFileGroup(a, b) {
		t.Fatal("same display name should be one group")
	}
	c := File{FileID: 3, Name: "Optional textures", FileName: "x.zip"}
	if sameFileGroup(a, c) {
		t.Fatal("different display names should be different groups")
	}
	a.ReplacedBy = 3
	if !sameFileGroup(a, c) {
		t.Fatal("the author's file_updates chain should join groups")
	}
}

func TestChooseFileForAFreshInstall(t *testing.T) {
	files := []File{
		{FileID: 1, Name: "Example", Version: "1.0", Category: "MAIN", IsPrimary: true},
		{FileID: 2, Name: "Example extras", Version: "2.0", Category: "OPTIONAL"},
		{FileID: 3, Name: "Example", Version: "2.0.0", Category: "MAIN"},
	}
	for version, want := range map[string]int{"2.0": 3, "": 1, "9.0": 1} {
		if got, ok := ChooseFile(files, version, 0); !ok || got.FileID != want {
			t.Errorf("version %q: got %d, %v, want %d", version, got.FileID, ok, want)
		}
	}
	if _, ok := ChooseFile([]File{{FileID: 9, Category: "OPTIONAL"}}, "1.0", 0); ok {
		t.Error("an optional file was chosen for a profile that has none")
	}
}

// alchemistry is shaped like Nexus mod 22743: a main file and an optional bundle pack on one page, each with its
// own file_updates chain, and archived legacy packs. The mod page's version (2.0.2) is the main file's; the
// bundle pack's newest file is 2.0.1.
func alchemistry() []File {
	return []File{
		{FileID: 172590, Name: "(LEGACY) Alchemistry - CC Bundles (Easy)", Version: "2.0.0", Category: "ARCHIVED"},
		{FileID: 175151, Name: "Alchemistry", Version: "2.0.1", Category: "OLD_VERSION", ReplacedBy: 175656},
		{FileID: 175656, Name: "Alchemistry", Version: "2.0.2", Category: "MAIN", IsPrimary: true},
		{FileID: 175660, Name: "Alchemistry CC Bundles", Version: "2.0.0", Category: "OLD_VERSION", ReplacedBy: 178711},
		{FileID: 178711, Name: "Alchemistry CC Bundles", Version: "2.0.1", Category: "OPTIONAL"},
	}
}

func TestChooseFileNeverUpdatesABundlePackToTheMainFile(t *testing.T) {
	if got, ok := ChooseFile(alchemistry(), "2.0.2", 178711); ok {
		t.Fatalf("got file %d; want none, the bundle pack has no 2.0.2 file", got.FileID)
	}
	if got, ok := ChooseFile(alchemistry(), "2.0.2", 175151); !ok || got.FileID != 175656 {
		t.Fatalf("main file: got %d, %v; want 175656", got.FileID, ok)
	}
}

func TestSupersedesFindsTheBundlePacksOwnNewerFile(t *testing.T) {
	files := append(alchemistry(),
		File{FileID: 181000, Name: "Alchemistry CC Bundles", Version: "2.0.2", Category: "OPTIONAL"},
		File{FileID: 181001, Name: "Alchemistry", Version: "2.0.3", Category: "MAIN"},
	)
	files[2].Category = "OLD_VERSION"
	// The GraphQL listing carries no file_updates chain, so the name has to find it.
	for i := range files {
		files[i].ReplacedBy = 0
	}
	got, ok := Supersedes(files, FileByID(files, 178711), "2.0.2", "")
	if !ok || got.FileID != 181000 {
		t.Fatalf("got file %d, %v; want the bundle pack's 181000", got.FileID, ok)
	}
}

func TestSupersedesFallsBackToCategoryThenTheOneMainFile(t *testing.T) {
	files := []File{
		{FileID: 10, Name: "Old Pack Name", Version: "1.0", Category: "OLD_VERSION"},
		{FileID: 11, Name: "Old Main Name", Version: "1.0", Category: "OLD_VERSION"},
		{FileID: 20, Name: "New Pack Name", Version: "1.1", Category: "OPTIONAL"},
		{FileID: 21, Name: "New Main Name", Version: "1.2", Category: "MAIN"},
	}
	if got, ok := Supersedes(files, files[0], "1.1", "OPTIONAL"); !ok || got.FileID != 20 {
		t.Errorf("optional at version: got %d, %v; want 20", got.FileID, ok)
	}
	if got, ok := Supersedes(files, files[1], "1.1", "MAIN"); !ok || got.FileID != 21 {
		t.Errorf("one main file: got %d, %v; want 21", got.FileID, ok)
	}
	if got, ok := Supersedes(files, files[0], "1.2", "OPTIONAL"); ok {
		t.Errorf("an optional entry got file %d; want none", got.FileID)
	}
	files = append(files, File{FileID: 22, Name: "Other Main", Version: "1.2", Category: "MAIN"})
	if got, ok := Supersedes(files, files[1], "1.3", "MAIN"); ok {
		t.Errorf("two main files: got %d; want none", got.FileID)
	}
}
