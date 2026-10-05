package sharesvc

import (
	"context"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

func TestPlanReplaceKeepsLocalAndListsRemovals(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Entries: []profile.Entry{
		{Key: "smapi-1", Source: profile.Source{Kind: profile.SourceSMAPI}, Mods: []profile.Component{{Name: "SMAPI"}}},
		{Key: "n-100-1", Source: profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 1}, Mods: []profile.Component{{Name: "Keep", ID: "smapi:A.Keep"}}},
		{Key: "n-200-2", Source: profile.Source{Kind: profile.KindNexus, ModID: 200, FileID: 2}, Mods: []profile.Component{{Name: "Drop", ID: "smapi:A.Drop"}}},
		{Key: "n-100-9", Source: profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 9}, Mods: []profile.Component{{Name: "Old file", ID: "smapi:A.Keep"}}},
		{Key: "local-x", Source: profile.Source{Kind: profile.KindLocal, Name: "mine.zip"}, Mods: []profile.Component{{Name: "Mine", ID: "smapi:A.Mine"}}},
	}}
	mods := []Mod{
		{Key: "a", Site: SiteNexus, ModID: 100, FileID: 1, IDs: []mod.ID{"smapi:A.Keep"}},
	}
	got := PlanReplace(p, mods)
	if !slices.Equal(got.KeepLocal, []string{"Mine"}) {
		t.Fatalf("keep = %+v", got)
	}
	if !slices.Contains(got.Remove, "Drop") || !slices.Contains(got.Remove, "Old file") || slices.Contains(got.Remove, "Keep") {
		t.Fatalf("remove = %+v", got)
	}
}

func TestReplaceRemovesExtrasAndQueuesTheShare(t *testing.T) {
	s, rec := newService(t, true)
	prof, err := s.d.Profiles.Create("stardew", "Mine")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.d.Profiles.InstallNexus("stardew", prof.ID, modZip(t, "A.Keep"), profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.d.Profiles.InstallNexus("stardew", prof.ID, modZip(t, "A.Drop"), profile.Source{Kind: profile.KindNexus, ModID: 600, FileID: 6}); err != nil {
		t.Fatal(err)
	}
	text := link(t, "Cozy", share.Ref{ModID: 100, FileID: 1})
	pv, err := s.PreviewLink(context.Background(), "stardew", text, prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(pv.Replace.Remove, "A.Drop") && !slices.ContainsFunc(pv.Replace.Remove, func(n string) bool { return n != "" }) {
		t.Fatalf("plan = %+v", pv.Replace)
	}
	if _, err := s.Replace(context.Background(), "stardew", pv.Session, prof.ID, nil); err != nil {
		t.Fatal(err)
	}
	got, err := s.find("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(got.Entries, func(e profile.Entry) bool {
		return e.Source.Kind == profile.KindNexus && e.Source.ModID == 600
	}) {
		t.Fatalf("drop still present: %+v", got.Entries)
	}
	_ = rec
}

func TestReplaceNeedsAProfile(t *testing.T) {
	s, _ := newService(t, true)
	_, err := s.Replace(context.Background(), "stardew", "nope", "", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
