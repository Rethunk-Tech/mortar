package fsx

import (
	"strings"
	"testing"
)

func TestExtendedPathOnlyPrefixesLongDrivePaths(t *testing.T) {
	long := `C:\Users\u\AppData\Local\Mortar\store\blobs\` + strings.Repeat("a", 64) + `\` + strings.Repeat(`d\`, 80) + "f.dll"
	for _, tc := range []struct{ in, want string }{
		{`C:\short\path`, `C:\short\path`},
		{`\\server\share\` + strings.Repeat("x", 300), `\\server\share\` + strings.Repeat("x", 300)},
		{`\\?\C:\` + strings.Repeat("x", 300), `\\?\C:\` + strings.Repeat("x", 300)},
		{long, `\\?\` + long},
	} {
		if got := extendedPath(tc.in); got != tc.want {
			t.Errorf("extendedPath(%.40q) = %.40q, want %.40q", tc.in, got, tc.want)
		}
	}
}
