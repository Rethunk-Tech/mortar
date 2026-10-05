package launchplan

import (
	"slices"
	"strings"
	"testing"
)

func TestSplitLaunchOptions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		windows bool
		in      string
		want    []string
	}{
		{"shell quotes", false, `  --developer-mode  "path with space"  'also here'  `, []string{"--developer-mode", "path with space", "also here"}},
		{"shell backslash escapes", false, `a\ b "c\"d"`, []string{"a b", `c"d`}},
		{"windows path with spaces", true, `--log "C:\Program Files (x86)\Mortar Logs\smapi.txt"`, []string{"--log", `C:\Program Files (x86)\Mortar Logs\smapi.txt`}},
		{"windows bare path", true, `--dir C:\Games\Stardew\Mods`, []string{"--dir", `C:\Games\Stardew\Mods`}},
		{"windows apostrophe in a path", true, `--dir "C:\Users\O'Brien\Mods"`, []string{"--dir", `C:\Users\O'Brien\Mods`}},
		{"windows escaped quote", true, `--title "say \"hi\""`, []string{"--title", `say "hi"`}},
		{"windows trailing backslashes close the quote", true, `--dir "C:\a b\\" --x`, []string{"--dir", `C:\a b\`, "--x"}},
		{"windows odd run keeps the quote", true, `a\\\"b`, []string{`a\"b`}},
	} {
		got, err := splitShellWords(tc.in, tc.windows)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %q, %v; want %q", tc.name, got, err, tc.want)
		}
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
