package itch

import (
	"github.com/Rethunk-Tech/mortar/internal/source"
	"strings"
	"testing"
)

func TestParsePageURL(t *testing.T) {
	t.Parallel()
	for text, want := range map[string]string{
		"https://someone.itch.io/cool-mod":              "someone/cool-mod",
		"  https://Someone.itch.io/Cool_Mod/?x=1#frag ": "someone/Cool_Mod",
		"https://a-b.itch.io/m":                         "a-b/m",
	} {
		if got, ok := ParsePageURL(text); !ok || got != want {
			t.Errorf("%q = %q, %v; want %q", text, got, ok, want)
		}
	}
	for _, text := range []string{
		"http://someone.itch.io/cool-mod",
		"https://itch.io/cool-mod",
		"https://www.itch.io/cool-mod",
		"https://someone.itch.io/",
		"https://someone.itch.io/cool/devlog",
		"https://user:pw@someone.itch.io/cool",
		"https://someone.itch.io:8443/cool",
		"https://someone.itch.io.evil.com/cool",
		"https://someone.itch.io/%2e%2e",
		"https://some one.itch.io/cool",
		"https://evil.com/someone.itch.io/cool",
		"",
	} {
		if got, ok := ParsePageURL(text); ok {
			t.Errorf("%q accepted as %q", text, got)
		}
	}
}

func TestPageURLRebuildsTheCanonicalAddress(t *testing.T) {
	t.Parallel()
	if got := (Driver{}).ModPageURL("", "someone/cool-mod"); got != "https://someone.itch.io/cool-mod" {
		t.Fatalf("got %q", got)
	}
	if PageURL("someone/../x") != "" || PageURL("nogame") != "" {
		t.Fatal("an invalid page id built an address")
	}
}

// FuzzParsePageURL holds that an accepted page name rebuilds an address on itch.io whatever the text.
func FuzzParsePageURL(f *testing.F) {
	f.Add("https://someone.itch.io/cool-mod")
	f.Add("https://itch.io/cool-mod")
	f.Add("https://someone.itch.io/%2e%2e")
	f.Add("https://user:pw@someone.itch.io/x")
	f.Add("https://someone.itch.io.evil.com/x")
	f.Fuzz(func(t *testing.T, text string) {
		id, ok := ParsePageURL(text)
		if !ok {
			return
		}
		if !ValidPage(id) || !strings.HasPrefix(PageURL(id), "https://") || !strings.Contains(PageURL(id), ".itch.io/") {
			t.Fatalf("%q gave %q", text, id)
		}
	})
}

func TestSearchWithoutAKeyIsEmptyAndNeedsNoAPIKey(t *testing.T) {
	t.Parallel()
	page, err := (Driver{Key: func() (string, error) { t.Fatal("the key was read"); return "", nil }}).Search(t.Context(), source.Query{Game: "g"})
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("%+v %v", page, err)
	}
}
