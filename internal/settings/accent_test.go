package settings

import (
	"os"
	"regexp"
	"testing"
)

func TestAccentColorsMatchTheFrontend(t *testing.T) {
	b, err := os.ReadFile("../../frontend/src/theme/accents.ts")
	if err != nil {
		t.Fatal(err)
	}
	ts := map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^\s+(\w+): '(#[0-9A-Fa-f]{6})',$`).FindAllStringSubmatch(string(b), -1) {
		ts[m[1]] = m[2]
	}
	if len(ts) != len(accentColors) || len(accents) != len(accentColors) {
		t.Fatalf("accents.ts has %d, accent.go %d, settings accents %d", len(ts), len(accentColors), len(accents))
	}
	for _, name := range accents {
		if accentColors[name] == "" || ts[name] != accentColors[name] {
			t.Errorf("%s: accent.go %q, accents.ts %q", name, accentColors[name], ts[name])
		}
	}
}
