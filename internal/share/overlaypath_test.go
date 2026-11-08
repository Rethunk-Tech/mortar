package share

import "testing"

func TestValidOverlayPath(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"", true},
		{"a", true},
		{"a/b.json", true},
		{".", false},
		{"..", false},
		{"../a", false},
		{"a/../b", false},
		{"a/..", false},
		{"/a", false},
		{"a//b", false},
		{"a/./b", false},
		{"a/", false},
		{`a\b`, false},
		{"c:/x", false},
		{"a\x00b", false},
	} {
		if got := validOverlayPath(tc.path); got != tc.want {
			t.Errorf("validOverlayPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
