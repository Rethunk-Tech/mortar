package smapimanifest

import "testing"

func TestFirstNexusID(t *testing.T) {
	tests := []struct {
		keys []string
		want int
	}{
		{nil, 0},
		{[]string{"GitHub:me/repo"}, 0},
		{[]string{"Nexus:1", "Nexus:2"}, 1},
		{[]string{"GitHub:me/repo", "Nexus:9"}, 9},
		{[]string{"Nexus:0", "Nexus:-1", "Nexus:3"}, 3},
		{[]string{"ModDrop:5", "Nexus:4"}, 4},
	}
	for _, tt := range tests {
		if got := FirstNexusID(tt.keys); got != tt.want {
			t.Errorf("FirstNexusID(%q) = %d, want %d", tt.keys, got, tt.want)
		}
	}
}

func TestFirstGitHubRepo(t *testing.T) {
	tests := []struct {
		keys []string
		want string
	}{
		{nil, ""},
		{[]string{"Nexus:1"}, ""},
		{[]string{"GitHub:me/repo", "GitHub:other/x"}, "me/repo"},
		{[]string{"Nexus:1", "GitHub:me/repo"}, "me/repo"},
		{[]string{"GitHub:me", "GitHub:me/repo"}, "me/repo"},
	}
	for _, tt := range tests {
		if got := FirstGitHubRepo(tt.keys); got != tt.want {
			t.Errorf("FirstGitHubRepo(%q) = %q, want %q", tt.keys, got, tt.want)
		}
	}
}
