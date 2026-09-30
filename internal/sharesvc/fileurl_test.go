package sharesvc

import (
	"path/filepath"
	"testing"
)

func TestFileURLPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in, want string
	}{
		{"linux", "file:///tmp/pack.mortar", filepath.FromSlash("/tmp/pack.mortar")},
		{"windows", "file:///C:/Users/x.mortar", filepath.FromSlash("C:/Users/x.mortar")},
		{"uncoded space", "file:///tmp/a%20b.mortar", filepath.FromSlash("/tmp/a b.mortar")},
		{"unc", "file://server/share/pack.mortar", filepath.FromSlash(`\\server/share/pack.mortar`)},
		{"plain", `/tmp/plain.mortar`, `/tmp/plain.mortar`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := fileURLPath(tc.in); got != tc.want {
				t.Fatalf("fileURLPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
