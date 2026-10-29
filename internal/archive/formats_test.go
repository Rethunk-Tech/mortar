package archive

import "testing"

func TestLz4NamesAreArchives(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{"a.tar.lz4": "a", "B.LZ4": "B"} {
		if StripExtension(in) != want {
			t.Errorf("StripExtension(%q) = %q", in, StripExtension(in))
		}
	}
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
