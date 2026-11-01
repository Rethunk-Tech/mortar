package vsmoddb

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestOrderBy(t *testing.T) {
	t.Parallel()
	for sort, want := range map[string]string{
		source.SortDownloads:    "downloads",
		source.SortEndorsements: "follows",
		source.SortUpdated:      "lastreleased",
		source.SortNewest:       "asset.created",
		"name":                  "",
		"":                      "",
	} {
		if got := orderBy(sort); got != want {
			t.Errorf("orderBy(%q) = %q, want %q", sort, got, want)
		}
	}
}

func TestNewestPrefersStableThenAnyWithAFile(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		releases []release
		want     string
		ok       bool
	}{
		{"test build is the fallback", []release{{ModVersion: "1.0.0"}, {ModVersion: "1.1.0-rc1", MainFile: "f.zip"}}, "1.1.0-rc1", true},
		{"stable wins over a newer test build", []release{{ModVersion: "2.0.0-pre.1", MainFile: "b.zip"}, {ModVersion: "1.0.0", MainFile: "a.zip"}}, "1.0.0", true},
		{"no file anywhere", []release{{ModVersion: "1.0.0"}, {ModVersion: "1.1.0-rc1"}}, "", false},
		{"no releases", nil, "", false},
	}
	for _, c := range cases {
		got, ok := modDetail{Releases: c.releases}.newest()
		if ok != c.ok || got.ModVersion != c.want {
			t.Errorf("%s: newest = %q, %v; want %q, %v", c.name, got.ModVersion, ok, c.want, c.ok)
		}
	}
}
