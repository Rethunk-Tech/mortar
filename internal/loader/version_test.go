package loader

import "testing"

func TestVersionChanged(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		recorded, installed string
		want                bool
	}{{"", "1.6.15", false}, {"1.6.15", "", false}, {"1.6.15", "1.6.15", false}, {"1.6.14", "1.6.15", true}} {
		if got := VersionChanged(c.recorded, c.installed); got != c.want {
			t.Errorf("VersionChanged(%q, %q) = %v", c.recorded, c.installed, got)
		}
	}
}
