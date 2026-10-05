package support

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestClipKeepsWholeCharacters(t *testing.T) {
	s := strings.Repeat("a", maxRenderLog-1) + "é"
	got := clip(s, maxRenderLog)
	if !utf8.ValidString(got) || got != strings.Repeat("a", maxRenderLog-1) {
		t.Fatalf("clip cut inside a character: %q", got[len(got)-3:])
	}
	if clip("short", maxRenderLog) != "short" {
		t.Fatal("a short message was cut")
	}
}
