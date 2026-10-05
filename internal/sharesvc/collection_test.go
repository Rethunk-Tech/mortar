package sharesvc

import (
	"context"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
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
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
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
		Files: func(_ context.Context, _ nexus.Title, id int) ([]nexus.File, error) {
			return []nexus.File{nf(id%100, "1.0", "MAIN", true)}, nil
		},
		SignedIn: func() bool { return true }, Premium: func() bool { return true },
		Env: func(string) problems.Environment { return stardewEnv }, Queue: &recorder{},
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
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
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
		Files: func(_ context.Context, _ nexus.Title, id int) ([]nexus.File, error) {
			return []nexus.File{nf(id%100, "1.0", "MAIN", true)}, nil
		},
		SignedIn: func() bool { return true }, Premium: func() bool { return true },
		Env: func(string) problems.Environment { return stardewEnv }, Queue: &recorder{},
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
	p := testenv.Profile(t, profiles, "stardew", "Mine")
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
	p := testenv.Profile(t, profiles, "stardew", "Linked")
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

func TestReadCollectionArchive(t *testing.T) {
	raw, err := os.ReadFile("testdata/collection-archive.7z")
	if err != nil {
		t.Fatal(err)
	}
	d, err := readCollectionArchive(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]map[string][]string{"Install Type": {"Options": {"Full"}}, "Extras": {"Bonus": {"A", "B"}}}
	if len(d.Fomod) != 1 || !reflect.DeepEqual(d.Fomod[modFile{100, 11}], want) {
		t.Fatalf("fomod = %+v", d.Fomod)
	}
	var paths []string
	for _, c := range d.Configs {
		if c.UniqueID != "Pat.Tweaks" {
			t.Errorf("config for %q", c.UniqueID)
		}
		paths = append(paths, c.Path)
	}
	slices.Sort(paths)
	if !reflect.DeepEqual(paths, []string{"config.json", "config/extra.json"}) {
		t.Fatalf("config paths = %v", paths)
	}
	if _, err := readCollectionArchive([]byte("not an archive")); err == nil {
		t.Fatal("garbage archive accepted")
	}
}

func TestCollectionImportAppliesArchiveForPremiumOnly(t *testing.T) {
	raw, err := os.ReadFile("testdata/collection-archive.7z")
	if err != nil {
		t.Fatal(err)
	}
	for _, premium := range []bool{true, false} {
		testfs.DataHome(t)
		_, profiles := testenv.Stores(t)
		var fetched []string
		rec := &recorder{}
		s := NewService(Deps{
			Profiles: profiles,
			Meta: fakeMeta{
				pages: map[int]meta.Page{100: page(100, "One", dsFile(11, "1.0", meta.Mod{UniqueID: "A.One", Version: "1.0"}))},
				coll: meta.Collection{
					Name: "Cozy Farm", Slug: "cozy-farm", Revision: 1, Instructions: "Start a new save.",
					DownloadLink: "/v2/collections/1/revisions/2/download_link",
					External:     []meta.CollectionExternal{{Name: "Hand Mod", Type: "browse", URL: "https://example.com/mod"}},
					Files:        []meta.CollectionFile{{ModID: 100, FileID: 11}},
				},
			},
			Files: func(context.Context, nexus.Title, int) ([]nexus.File, error) {
				return []nexus.File{nf(11, "1.0", "MAIN", true)}, nil
			},
			SignedIn: func() bool { return true }, Premium: func() bool { return premium },
			CollectionArchive: func(_ context.Context, link string) ([]byte, error) {
				fetched = append(fetched, link)
				return raw, nil
			},
			Env: func(string) problems.Environment { return stardewEnv }, Queue: rec,
		})
		pv, err := s.PreviewLink(context.Background(), "stardew", "https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm", "")
		if err != nil {
			t.Fatal(err)
		}
		c := pv.Collection
		wantDetails := DetailsListed
		if premium {
			wantDetails = DetailsArchive
		}
		if c == nil || c.Instructions != "Start a new save." || c.Details != wantDetails || len(c.External) != 1 || !c.External[0].InstallYourself || c.External[0].URL != "https://example.com/mod" {
			t.Fatalf("premium=%v collection info = %+v", premium, c)
		}
		if len(fetched) != 0 {
			t.Fatal("archive fetched before Import")
		}
		res, err := s.Import(context.Background(), "stardew", pv.Session, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(res.Profile.Notes, "Start a new save.") || !strings.Contains(res.Profile.Notes, "https://example.com/mod") {
			t.Fatalf("notes = %q", res.Profile.Notes)
		}
		if !premium {
			if len(fetched) != 0 || res.Collection != nil {
				t.Fatalf("free account: fetched %v, applied %+v", fetched, res.Collection)
			}
			continue
		}
		if res.Collection == nil || res.Collection.FomodMods != 1 || res.Collection.Configs != 2 || res.Collection.Error != "" {
			t.Fatalf("applied = %+v", res.Collection)
		}
		if len(rec.reqs) != 1 || !reflect.DeepEqual(rec.reqs[0].Fomod["Install Type"]["Options"], []string{"Full"}) {
			t.Fatalf("queued = %+v", rec.reqs)
		}
	}
}
