package itch

import (
	"testing"

	"github.com/Rethunk-Tech/go-itchio"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestPageURLRebuildsTheCanonicalAddress(t *testing.T) {
	t.Parallel()
	if got := (Driver{}).ModPageURL("", "someone/cool-mod"); got != "https://someone.itch.io/cool-mod" {
		t.Fatalf("got %q", got)
	}
	if itchio.PageURL("someone/../x") != "" || itchio.PageURL("nogame") != "" {
		t.Fatal("an invalid page id built an address")
	}
}

func TestSearchWithoutAKeyIsEmptyAndNeedsNoAPIKey(t *testing.T) {
	t.Parallel()
	page, err := (Driver{Key: func() (string, error) { t.Fatal("the key was read"); return "", nil }}).Search(t.Context(), source.Query{Game: "g"})
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("%+v %v", page, err)
	}
}
