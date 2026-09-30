package profile

import (
	"strings"
	"testing"
)

func TestUniqueName(t *testing.T) {
	long := strings.Repeat("x", maxName)
	cases := []struct {
		taken []string
		name  string
		want  string
	}{
		{nil, "Cozy", "Cozy"},
		{[]string{"Other"}, "Cozy", "Cozy"},
		{[]string{"Cozy"}, "Cozy", "Cozy (2)"},
		{[]string{"cozy", "Cozy (2)", "Cozy (4)"}, " Cozy ", "Cozy (3)"},
		{[]string{long}, long, long[:maxName-4] + " (2)"},
	}
	for _, c := range cases {
		if got := UniqueName(c.taken, c.name); got != c.want {
			t.Errorf("UniqueName(%q, %q) = %q, want %q", c.taken, c.name, got, c.want)
		}
	}
}
