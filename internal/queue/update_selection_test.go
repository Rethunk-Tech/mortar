package queue

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/nexus"
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
