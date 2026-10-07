package source

import "testing"

type gatedFake struct {
	Source
	reason string
}

func (g gatedFake) Unavailable() string { return g.reason }

func TestUnavailableReadsAsASentence(t *testing.T) {
	for reason, want := range map[string]string{
		"this build has no CurseForge API key": "This build has no CurseForge API key",
		"needs an itch.io API key":             "Needs an itch.io API key",
		"":                                     "",
	} {
		if got := Unavailable(gatedFake{reason: reason}); got != want {
			t.Errorf("Unavailable(%q) = %q, want %q", reason, got, want)
		}
	}
}
