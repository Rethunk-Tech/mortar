package cli

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestSentence(t *testing.T) {
	if Sentence(usererr.Busy) != "The game is already running." {
		t.Fatalf("%q", Sentence(usererr.Busy))
	}
	if Sentence(usererr.Unknown) != "Something went wrong." {
		t.Fatalf("%q", Sentence(usererr.Unknown))
	}
}
