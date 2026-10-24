package profile

import "testing"

func TestAnExtraFolderModMatchesItsRootWhateverTheCase(t *testing.T) {
	if self, child := placeIn(`C:\Mods\X`, `c:\mods`); self || !child {
		t.Fatalf("child: self=%v child=%v", self, child)
	}
	if self, _ := placeIn(`C:\Mods`, `c:\mods\`); !self {
		t.Fatal("the root itself")
	}
}
