package patreon

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestParsePostURL(t *testing.T) {
	good := map[string]string{
		"https://www.patreon.com/posts/cool-mod-v2-12345678":  "12345678",
		"https://patreon.com/posts/12345678":                  "12345678",
		"  https://www.patreon.com/posts/a-b-c-9/  ":          "9",
		"https://www.patreon.com/posts/x-1?utm_source=a#frag": "1",
	}
	for in, want := range good {
		if got, ok := ParsePostURL(in); !ok || got != want {
			t.Errorf("%q = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{
		"", "12345678", "http://www.patreon.com/posts/x-1", "https://evil.test/posts/x-1",
		"https://www.patreon.com.evil.test/posts/x-1", "https://user@www.patreon.com/posts/x-1",
		"https://www.patreon.com:444/posts/x-1", "https://www.patreon.com/creator/posts",
		"https://www.patreon.com/posts/x-1/extra", "https://www.patreon.com/posts/-1",
		"https://www.patreon.com/posts/x-abc", "https://www.patreon.com/posts/x-1234567890123",
		"https://www.patreon.com/posts/x y-1", "javascript:alert(1)", "https://www.patreon.com/posts/",
	} {
		if got, ok := ParsePostURL(in); ok {
			t.Errorf("%q must be refused, got %q", in, got)
		}
	}
}

func TestPostURLIsRebuiltFromTheIDAlone(t *testing.T) {
	if got := PostURL("123"); got != "https://www.patreon.com/posts/123" {
		t.Fatalf("got %q", got)
	}
	if PostURL("1/../x") != "" || PostURL("") != "" {
		t.Fatal("a non-id must not build an address")
	}
	if (Driver{}).ModPageURL("k", "77") != "https://www.patreon.com/posts/77" {
		t.Fatal("the mod page is the post")
	}
}

func TestPatreonIsAHandoffSourceThatBrowseNeverSearches(t *testing.T) {
	e, ok := source.Get(ID)
	if !ok {
		t.Fatal("not registered")
	}
	if _, searchable := e.Source.(source.Searcher); searchable {
		t.Fatal("Patreon must not be searchable")
	}
	if name, ok := source.NameOfHost("www.patreon.com"); !ok || name != "Patreon" {
		t.Fatalf("host lookup = %q, %v", name, ok)
	}
}
