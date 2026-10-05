package profile

import "testing"

func TestNormalizeAuthorName(t *testing.T) {
	t.Parallel()
	if NormalizeAuthorName("  Pathoschild  ") != "pathoschild" {
		t.Fatal("trim and lower")
	}
	if NormalizeAuthorName("Space\tChase") != "space chase" {
		t.Fatal("collapse space")
	}
}

func TestSplitManifestAuthors(t *testing.T) {
	t.Parallel()
	got := SplitManifestAuthors("A, B & C")
	if len(got) != 3 || got[0] != "A" || got[1] != "B" || got[2] != "C" {
		t.Fatalf("split: %#v", got)
	}
	if SplitManifestAuthors("") != nil {
		t.Fatal("empty")
	}
}

func TestAuthorFieldIncludes(t *testing.T) {
	t.Parallel()
	if !AuthorFieldIncludes("Pathoschild, SpaceChase0", "spacechase0") {
		t.Fatal("should match second author")
	}
	if AuthorFieldIncludes("Only One", "Other") {
		t.Fatal("should not match")
	}
}
