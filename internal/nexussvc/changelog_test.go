package nexussvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/github"
)

func TestReleaseChangelogsSkipsDraftsAndTrimsTags(t *testing.T) {
	got := releaseChangelogs([]github.Release{
		{Tag: "v2.0.0", Body: " big\n", Published: "2026-01-02T03:04:05Z"},
		{Tag: "v1.9.0", Draft: true},
		{Tag: "1.0.0"},
	})
	if len(got) != 2 || got[0].Version != "2.0.0" || got[0].Date != "2026-01-02" || got[0].Body != "big" || got[1].Version != "1.0.0" {
		t.Fatalf("got %+v", got)
	}
}
