package stardew

import (
	"slices"
	"strings"
	"testing"
)

func TestParseLaunchOptions(t *testing.T) {
	t.Parallel()
	got, err := ParseLaunchOptions(`  --developer-mode  "path with space"  'also here'  `)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--developer-mode", "path with space", "also here"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParseLaunchOptionsDenied(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"--mods-path /tmp/x", `--mods-path="/tmp/x"`, "--no-terminal", "--skip-terminal"} {
		_, err := ParseLaunchOptions(in)
		if err == nil {
			t.Fatalf("%q: want denylist error", in)
		}
		if !strings.Contains(err.Error(), "is set by Mortar") {
			t.Fatalf("%q: got %v", in, err)
		}
	}
}

func TestParseLaunchOptionsUnclosedQuote(t *testing.T) {
	t.Parallel()
	if _, err := ParseLaunchOptions(`"oops`); err == nil {
		t.Fatal("want unclosed quote")
	}
}
