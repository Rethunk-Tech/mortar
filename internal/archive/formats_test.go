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
