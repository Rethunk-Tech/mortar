package problems

import "testing"

func TestVersionChangeWarningUnchangedOmitsBroken(t *testing.T) {
	broken := []Broken{{UniqueID: "A.Mod", Name: "A"}}
	got := versionChangeWarning("1.6.15", "1.6.15", broken)
	if got.Changed {
		t.Fatalf("Changed = true, want false")
	}
	if len(got.Broken) != 0 {
		t.Fatalf("Broken = %v, want none", got.Broken)
	}
	if got.Recorded != "1.6.15" || got.Installed != "1.6.15" {
		t.Fatalf("versions = %#v", got)
	}
}

func TestVersionChangeWarningChangedReturnsBroken(t *testing.T) {
	broken := []Broken{{UniqueID: "A.Mod", Name: "A"}}
	got := versionChangeWarning("1.6.8", "1.6.15", broken)
	if !got.Changed {
		t.Fatalf("Changed = false, want true")
	}
	if len(got.Broken) != 1 || got.Broken[0].UniqueID != "A.Mod" {
		t.Fatalf("Broken = %v", got.Broken)
	}
	if got.Recorded != "1.6.8" || got.Installed != "1.6.15" {
		t.Fatalf("versions = %#v", got)
	}
}
