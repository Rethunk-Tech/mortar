package github

import "testing"

func TestIsArchive(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, ctype string
		want        bool
	}{
		{"m.tar.gz", "application/x-gtar", true},
		{"m.tgz", "application/x-gtar", true},
		{"m.lz4", "application/x-lz4", true},
		{"m.tar.xz", "application/x-xz-compressed-tar", true},
		{"m.tar.zst", "application/x-zstd-compressed-tar", true},
		{"m.zip", "", true},
		{"m.zip", "text/html", false},
		{"m.txt", "application/zip", false},
	} {
		if got := IsArchive(Asset{Name: c.name, ContentType: c.ctype}); got != c.want {
			t.Errorf("IsArchive(%q, %q) = %v, want %v", c.name, c.ctype, got, c.want)
		}
	}
}
