package archive

import (
	"strings"
	"testing"
)

func TestLz4StreamKeepsItsName(t *testing.T) {
	t.Parallel()
	if SavedName("id", "log.lz4") != "id~~log.lz4" {
		t.Errorf("SavedName = %q", SavedName("id", "log.lz4"))
	}
}

func TestSavedNameKeepsWholeEnding(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]string{"x.tar.gz": "id.tar.gz", "X.TBZ": "id.TBZ", "m.zip": "id.zip"} {
		got := SavedName("id", name)
		if got != want || SavedID(got) != "id" {
			t.Errorf("SavedName(%q) = %q, SavedID = %q", name, got, SavedID(got))
		}
	}
	if SavedID("id.gz") != "id" || SavedID("id.dll") != "id" {
		t.Error("an older release's saved name must still parse")
	}
}

// A bare stream's saved name is a file name alone: every character Windows refuses becomes "_", and a name too long
// for a folder is cut before its ending rather than losing the ending that names the file.
func TestSavedNameOfAStreamIsSafeAndShort(t *testing.T) {
	t.Parallel()
	if got := SavedName("id", `dir/na:me?.txt.gz`); got != "id~~na_me_.txt.gz" {
		t.Errorf("SavedName = %q", got)
	}
	long := strings.Repeat("x", maxSavedName+20) + ".gz"
	got := SavedName("id", long)
	if name := strings.TrimPrefix(got, "id~~"); len(name) != maxSavedName || !strings.HasSuffix(name, ".gz") {
		t.Errorf("a long name was saved as %d bytes %q", len(name), name)
	}
}

// SavedID reads the id from the separator, from an archive ending, or from a name that has neither.
func TestSavedIDFallsBackToTheNameWithoutItsExtension(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]string{"id~~notes.txt": "id", "id.zip": "id", "id.partial": "id", "plain": "plain"} {
		if got := SavedID(name); got != want {
			t.Errorf("SavedID(%q) = %q, want %q", name, got, want)
		}
	}
}
