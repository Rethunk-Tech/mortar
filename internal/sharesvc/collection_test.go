package sharesvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/store"
)

func TestParseCollectionURL(t *testing.T) {
	tests := []struct {
		in       string
		domain   string
		slug     string
		revision int
		ok       bool
	}{
		{in: "https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm", domain: "stardewvalley", slug: "cozy-farm", ok: true},
		{in: "https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm/revisions/3", domain: "stardewvalley", slug: "cozy-farm", revision: 3, ok: true},
		{in: "https://next.nexusmods.com/games/stardewvalley/collections/cozy-farm", domain: "stardewvalley", slug: "cozy-farm", ok: true},
		{in: "https://nexusmods.com/games/stardewvalley/collections/cozy-farm/", domain: "stardewvalley", slug: "cozy-farm", ok: true},
		{in: "https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm?tab=mods", domain: "stardewvalley", slug: "cozy-farm", ok: true},
		{in: "https://www.nexusmods.com/stardewvalley/mods/1915"},
		{in: "https://www.nexusmods.com/games/stardewvalley/mods/1915"},
		{in: "mortar://stardew/p/abc"},
	}
	for _, test := range tests {
		domain, slug, revision, ok := parseCollectionURL(test.in)
		if ok != test.ok || domain != test.domain || slug != test.slug || revision != test.revision {
			t.Errorf("parseCollectionURL(%q) = %q, %q, %d, %v", test.in, domain, slug, revision, ok)
		}
	}
}

func TestPreviewLinkCollection(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	fm := fakeMeta{
		pages: map[int]meta.Page{
			100: page(100, "One", dsFile(1, "1.0", meta.Mod{UniqueID: "A.One", Version: "1.0"})),
			200: page(200, "Two", dsFile(2, "1.0", meta.Mod{UniqueID: "A.Two", Version: "1.0"})),
			300: page(300, "Three", dsFile(3, "1.0", meta.Mod{UniqueID: "A.Three", Version: "1.0"})),
		},
		coll: meta.Collection{
			Name: "Cozy Farm",
			Slug: "cozy-farm",
			Files: []meta.CollectionFile{
				{ModID: 100, FileID: 1},
				{ModID: 200, FileID: 2, Optional: true},
				{ModID: 300, FileID: 3},
			},
		},
	}
	s := NewService(Deps{
		Profiles: profiles, Meta: fm,
		Files: func(_ context.Context, id int) ([]nexus.File, error) {
			return []nexus.File{nf(id%100, "1.0", "MAIN", true)}, nil
		},
		SignedIn: func() bool { return true }, Premium: func() bool { return true },
		Env: func(string) problems.Environment { return problems.Environment{} }, Queue: &recorder{},
	})
	pv, err := s.PreviewLink(context.Background(), "stardew", "https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm", "")
	if err != nil {
		t.Fatal(err)
	}
	if pv.Name != "Cozy Farm" || len(pv.Mods) != 3 {
		t.Fatalf("preview = %+v", pv)
	}
}

func TestClassifyCollectionURL(t *testing.T) {
	got, ok := classify("https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm")
	if !ok || got.Kind != ArrivalLink {
		t.Fatalf("classify = %+v, %v", got, ok)
	}
}

func TestPreviewLinkCollectionWrongGame(t *testing.T) {
	s, _ := newService(t, true)
	_, err := s.PreviewLink(context.Background(), "stardew", "https://www.nexusmods.com/games/fallout4/collections/cozy-farm", "")
	if err == nil || !strings.Contains(err.Error(), "Stardew") {
		t.Fatalf("wrong game: %v", err)
	}
}

func cozyCollectionService(t *testing.T, revision int) (*Service, *profile.Store) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	fm := fakeMeta{
		pages: map[int]meta.Page{
			100: page(100, "One", dsFile(1, "1.0", meta.Mod{UniqueID: "A.One", Version: "1.0"})),
		},
		coll: meta.Collection{
			Name: "Cozy Farm", Slug: "cozy-farm", Revision: revision,
			Files: []meta.CollectionFile{{ModID: 100, FileID: 1}},
		},
	}
	s := NewService(Deps{
		Profiles: profiles, Meta: fm,
		Files: func(_ context.Context, id int) ([]nexus.File, error) {
			return []nexus.File{nf(id%100, "1.0", "MAIN", true)}, nil
		},
		SignedIn: func() bool { return true }, Premium: func() bool { return true },
		Env: func(string) problems.Environment { return problems.Environment{} }, Queue: &recorder{},
	})
	return s, profiles
}

func TestImportCollectionRecordsRefOnNewProfile(t *testing.T) {
	s, _ := cozyCollectionService(t, 3)
	pv, err := s.PreviewLink(context.Background(), "stardew", "https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm", "")
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Import(context.Background(), "stardew", pv.Session, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	ref := res.Profile.Collection
	if ref == nil || ref.Domain != "stardewvalley" || ref.Slug != "cozy-farm" || ref.Name != "Cozy Farm" || ref.Revision != 3 {
		t.Fatalf("collection = %+v", ref)
	}
	if res.Profile.Origin != profile.OriginCollection {
		t.Fatalf("origin = %q", res.Profile.Origin)
	}
}

func TestImportCollectionRecordsRefOnExistingProfile(t *testing.T) {
	s, profiles := cozyCollectionService(t, 2)
	p, err := profiles.Create("stardew", "Mine")
	if err != nil {
		t.Fatal(err)
	}
	pv, err := s.PreviewLink(context.Background(), "stardew", "https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Import(context.Background(), "stardew", pv.Session, p.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	ref := res.Profile.Collection
	if ref == nil || ref.Slug != "cozy-farm" || ref.Revision != 2 {
		t.Fatalf("collection = %+v", ref)
	}
	s.d.Meta = fakeMeta{
		pages: map[int]meta.Page{
			100: page(100, "One", dsFile(1, "1.0", meta.Mod{UniqueID: "A.One", Version: "1.0"})),
		},
		coll: meta.Collection{
			Name: "Cozy Farm", Slug: "cozy-farm", Revision: 4,
			Files: []meta.CollectionFile{{ModID: 100, FileID: 1}},
		},
	}
	pv, err = s.PreviewCollectionUpdate(context.Background(), "stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	res, err = s.Import(context.Background(), "stardew", pv.Session, p.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Profile.Collection == nil || res.Profile.Collection.Revision != 4 {
		t.Fatalf("updated collection = %+v", res.Profile.Collection)
	}
}

func TestCollectionStatusNewerSameAndError(t *testing.T) {
	s, profiles := cozyCollectionService(t, 5)
	p, err := profiles.Create("stardew", "Linked")
	if err != nil {
		t.Fatal(err)
	}
	st, err := s.CollectionStatus(context.Background(), "stardew", p.ID)
	if err != nil || st.Linked {
		t.Fatalf("unlinked: %+v, %v", st, err)
	}
	if _, err := profiles.SetCollection("stardew", p.ID, profile.CollectionRef{
		Domain: "stardewvalley", Slug: "cozy-farm", Name: "Cozy Farm", Revision: 3,
	}); err != nil {
		t.Fatal(err)
	}
	st, err = s.CollectionStatus(context.Background(), "stardew", p.ID)
	if err != nil || !st.Linked || st.Revision != 3 || st.Latest != 5 || st.Name != "Cozy Farm" ||
		st.URL != "https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm" {
		t.Fatalf("newer: %+v, %v", st, err)
	}
	s.d.Meta = fakeMeta{coll: meta.Collection{Name: "Cozy Farm", Revision: 3}}
	st, err = s.CollectionStatus(context.Background(), "stardew", p.ID)
	if err != nil || st.Latest != 3 || st.Revision != 3 {
		t.Fatalf("same: %+v, %v", st, err)
	}
	s.d.Meta = fakeMeta{collErr: errors.New("nexus down")}
	st, err = s.CollectionStatus(context.Background(), "stardew", p.ID)
	if err != nil || !st.Linked || st.Latest != 0 || st.Revision != 3 {
		t.Fatalf("error: %+v, %v", st, err)
	}
}
