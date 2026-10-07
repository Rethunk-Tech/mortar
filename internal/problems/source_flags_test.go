package problems

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

type fakeStandings struct {
	calls int
	flags map[string]source.Standing
	err   error
}

func (f *fakeStandings) Standings(_ context.Context, _ string, _ []string) (map[string]source.Standing, error) {
	f.calls++
	return f.flags, f.err
}

func flagService(t *testing.T) (*Service, *meta.Client) {
	t.Helper()
	home := t.TempDir()
	return &Service{home: home}, &meta.Client{CacheDir: filepath.Join(home, "cache")}
}

func TestNexusFlaggedReadsTheCachedPageStatus(t *testing.T) {
	t.Parallel()
	_, cl := flagService(t)
	for id, status := range map[int]string{1: "published", 2: "hidden", 3: "removed", 4: "under_moderation", 5: "not_published"} {
		page := nexus.Page{ModID: id, Status: status, Available: status == "published"}
		_, _ = meta.Cached(cl, nexussvc.DetailsName("stardewvalley", id), time.Hour, func() (nexussvc.Details, error) {
			return nexussvc.Details{Page: page}, nil
		})
	}
	mods := []framework.Mod{
		{Key: "nexus-1-10"}, {Key: "nexus-2-10"}, {Key: "nexus-3-10"}, {Key: "nexus-4-10"}, {Key: "nexus-5-10"}, {Key: "local"},
	}
	got := nexusFlagged(cl, "stardewvalley", mods)
	want := map[string]string{"nexus-2-10": "hidden", "nexus-3-10": "removed", "nexus-4-10": "moderated", "nexus-5-10": "unpublished"}
	if len(got) != len(want) {
		t.Fatalf("rows = %+v", got)
	}
	for _, b := range got {
		if b.Status != want[b.Key] || b.Source != "Nexus Mods" || b.Summary == "" {
			t.Errorf("%s = %+v", b.Key, b)
		}
	}
}

func siteMods(kind string) []framework.Mod {
	m := framework.Mod{Key: "k1", SourceKind: kind, SourceName: "proj-a"}
	if kind == profile.KindGitHub {
		m = framework.Mod{Key: "k1", SourceKind: kind, SourceRepo: "me/proj-a"}
	}
	other := m
	other.Key, other.SourceName, other.SourceRepo = "k2", "proj-b", "me/proj-b"
	return []framework.Mod{m, other}
}

// A flagged project of each batched site becomes one row naming the site, and the answer is cached so a second check
// asks nothing.
func TestSiteFlaggedRowsNameTheSiteAndAreCached(t *testing.T) {
	t.Parallel()
	for kind, site := range map[string]string{
		profile.KindModrinth: "Modrinth", profile.KindCurseForge: "CurseForge", profile.KindGitHub: "GitHub",
	} {
		s, cl := flagService(t)
		mods := siteMods(kind)
		id := mods[0].SourceName
		if kind == profile.KindGitHub {
			id = mods[0].SourceRepo
		}
		checker := &fakeStandings{flags: map[string]source.Standing{id: {State: "archived"}}}
		for range 2 {
			rows := s.siteFlagged(context.Background(), cl, kind, site, checker, mods)
			if len(rows) != 1 || rows[0].Key != "k1" || rows[0].Source != site || rows[0].Status != "archived" {
				t.Fatalf("%s rows = %+v", kind, rows)
			}
		}
		if checker.calls != 1 {
			t.Errorf("%s asked %d times, want 1", kind, checker.calls)
		}
	}
}

func TestSiteFlaggedSkipsWhenTheSiteCannotAnswer(t *testing.T) {
	t.Parallel()
	s, cl := flagService(t)
	rows := s.siteFlagged(context.Background(), cl, profile.KindGitHub, "GitHub", &fakeStandings{err: errors.New("busy")}, siteMods(profile.KindGitHub))
	if len(rows) != 0 {
		t.Fatalf("rows = %+v", rows)
	}
}
