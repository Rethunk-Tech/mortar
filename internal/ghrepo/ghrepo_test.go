// Package ghrepo validates a GitHub "owner/repo" name before it reaches an API URL.
package ghrepo

import "testing"

func TestValidRefusesWhatMovesTheURL(t *testing.T) {
	for _, repo := range []string{"me/..", "./mod", "../..", "me/.", "o/r?x", "o/r#x", "o/r/s", "o /r"} {
		if Valid(repo) {
			t.Errorf("Valid(%q) = true", repo)
		}
	}
	if !Valid("me/mod.cfg") {
		t.Error("a dotted repo name was refused")
	}
}
