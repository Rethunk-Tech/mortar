package saves

import "testing"

func TestDisplayNameMatchesTheApp(t *testing.T) {
	for _, c := range []struct{ farm, folder, want string }{
		{"Sunny", "Farm_1", "Sunny"},
		{"", "Farm_1", "Farm_1"},
		{"", "LCSaveFile2", "Save file 2"},
		{"", "LCChallengeFile", "Challenge moon"},
		{"", "characters_local/Ragnar.fch", "Ragnar"},
		{"", "worlds_local/Midgard.fwl", "Midgard"},
	} {
		if got := DisplayName(c.farm, c.folder); got != c.want {
			t.Errorf("DisplayName(%q, %q) = %q, want %q", c.farm, c.folder, got, c.want)
		}
	}
}
