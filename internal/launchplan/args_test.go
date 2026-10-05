package launchplan

import (
	"slices"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	t.Parallel()
	got, err := ParseArgs(`  --developer-mode  "path with space"  'also here'  `, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--developer-mode", "path with space", "also here"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParseArgsDenied(t *testing.T) {
	t.Parallel()
	denied := []string{"--mods-path", "--no-terminal", "--skip-terminal"}
	for _, in := range []string{"--mods-path /tmp/x", `--mods-path="/tmp/x"`, "--no-terminal", "--skip-terminal"} {
		_, err := ParseArgs(in, denied)
		if err == nil {
			t.Fatalf("%q: want denylist error", in)
		}
		if !strings.Contains(err.Error(), "is set by Mortar") {
			t.Fatalf("%q: got %v", in, err)
		}
	}
}

func TestParseArgsUnclosedQuote(t *testing.T) {
	t.Parallel()
	if _, err := ParseArgs(`"oops`, nil); err == nil {
		t.Fatal("want unclosed quote")
	}
}
