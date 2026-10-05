package problems

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/source/modrinth"
	nexussource "github.com/Rethunk-Tech/mortar/internal/source/nexus"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

type fakeThunderstore struct {
	id    string
	items []source.Item
}

func (f fakeThunderstore) ID() string { return f.id }
func (f fakeThunderstore) Name() string {
	return map[string]string{"thunderstore": "Thunderstore"}[f.id]
}
func (fakeThunderstore) Modes() []source.Acquire { return nil }
func (f fakeThunderstore) Search(context.Context, source.Query) (source.Page, error) {
	return source.Page{Items: f.items}, nil
}

func TestThunderstoreUpdatesOfferNewerVersionsAndMarkSourceSwitches(t *testing.T) {
	// The game's other sources answer nothing, so the test never reaches the network.
	source.Register(fakeThunderstore{id: "nexus"})
	source.Register(fakeThunderstore{id: "thunderstore", items: []source.Item{
		{ID: "Alice-Cool", Name: "Cool", Author: "Alice", Version: "2.0.0", Repo: "alice/cool", URL: "https://t/cool"},
	}})
	t.Cleanup(func() {
		source.Register(thunderstore.Driver{})
		source.Register(nexussource.Driver{})
	})
	turns, nexusTurns := 0, 0
	s := &Service{Throttle: func(_ context.Context, src string) (func(), error) {
		turns++
		if src == profile.KindNexus {
			nexusTurns++
		}
		return func() {}, nil
	}}
	same := framework.Mod{Key: "a", SourceKind: profile.KindThunderstore, SourceName: "Alice-Cool", SourceVersion: "1.0.0"}
	same.Name, same.Version = "Cool", "1.0.0"
	viaRepo := framework.Mod{Key: "b", SourceKind: profile.KindGitHub, SourceRepo: "Alice/Cool"}
	viaRepo.Name, viaRepo.Version = "Other", "1.0.0"
	current := framework.Mod{Key: "c", SourceKind: profile.KindThunderstore, SourceName: "Alice-Cool", SourceVersion: "2.0.0"}
	current.Name, current.Version = "Cool", "2.0.0"
	fromNexus := framework.Mod{Key: "d", SourceKind: profile.KindNexus}
	fromNexus.Name, fromNexus.Version = "Nexus Mod", "1.0.0"
	got := s.sourceUpdates(context.Background(), "lethal-company", []framework.Mod{same, viaRepo, current, fromNexus}, nil)
	if len(got) != 2 {
		t.Fatalf("updates = %+v", got)
	}
	if got[0].Key != "a" || got[0].Switch || got[0].Package != "Alice-Cool" || got[0].Source != "Thunderstore" {
		t.Errorf("same-source update = %+v", got[0])
	}
	if got[1].Key != "b" || !got[1].Switch {
		t.Errorf("cross-source update = %+v", got[1])
	}
	if turns == 0 {
		t.Error("searches did not wait for their source's turn")
	}
	if nexusTurns != 0 {
		t.Errorf("Nexus was searched %d times; its updates come from update keys", nexusTurns)
	}
	covered := s.sourceUpdates(context.Background(), "lethal-company", []framework.Mod{viaRepo}, []Update{{Key: "b", Version: "2.0.0"}})
	if len(covered) != 0 {
		t.Errorf("an update already offered was offered again: %+v", covered)
	}
}

func TestNexusModsWithoutUpdateKeysAreAskedInOneBatch(t *testing.T) {
	asks := 0
	s := &Service{NexusPages: func(_ context.Context, game string, ids []int) (map[int]nexus.Page, error) {
		asks++
		if game != "lethal-company" || len(ids) != 3 {
			t.Errorf("asked %s %v", game, ids)
		}
		return map[int]nexus.Page{
			10: {Version: "1.2.0", Available: true},
			11: {Version: "9.0.0", Available: false},
			13: {Version: "2.0", Available: true},
		}, nil
	}}
	mk := func(key string, id int, version string) framework.Mod {
		m := framework.Mod{Key: key, SourceKind: profile.KindNexus, SourceModID: id, SourceVersion: version}
		m.Name, m.Version = key, version
		return m
	}
	keyed := mk("keyed", 12, "1.0.0")
	keyed.UpdateKeys = []string{"Nexus:12"}
	got := s.nexusPageUpdates(context.Background(), "lethal-company", []framework.Mod{
		mk("old", 10, "1.0.0"), mk("gone", 11, "1.0.0"), mk("same", 10, "1.2.0"), mk("unordered", 13, "1.0.0"), keyed,
	}, nil)
	if asks != 1 || len(got) != 1 || got[0].Key != "old" || got[0].Version != "1.2.0" || got[0].NexusID != 10 || got[0].Source != "Nexus" {
		t.Fatalf("asks = %d, updates = %+v", asks, got)
	}
}

type fakeListed struct {
	fakeThunderstore
	deps map[source.VersionRef][]string
}

func (f fakeListed) Dependencies(context.Context, string, string, []source.VersionRef) (map[source.VersionRef][]string, error) {
	return f.deps, nil
}

func TestUpdatesNameTheDependenciesTheNewVersionChanges(t *testing.T) {
	old, next := source.VersionRef{ID: "Alice-Cool", Version: "1.0.0"}, source.VersionRef{ID: "Alice-Cool", Version: "2.0.0"}
	source.Register(fakeThunderstore{id: "nexus"})
	base := fakeThunderstore{id: "thunderstore", items: []source.Item{{ID: "Alice-Cool", Name: "Cool", Author: "Alice", Version: "2.0.0"}}}
	source.Register(fakeListed{
		base,
		map[source.VersionRef][]string{
			old:  {"BepInEx-BepInExPack", "Bob-Lib", "Old-Gone"},
			next: {"bepinex-bepinexpack", "Bob-Lib", "New-Dep", "Another-Dep"},
		},
	})
	t.Cleanup(func() {
		source.Register(thunderstore.Driver{})
		source.Register(nexussource.Driver{})
	})
	m := framework.Mod{Key: "a", SourceKind: profile.KindThunderstore, SourceName: "Alice-Cool", SourceVersion: "1.0.0"}
	m.Name, m.Version = "Cool", "1.0.0"
	got := (&Service{}).sourceUpdates(context.Background(), "lethal-company", []framework.Mod{m}, nil)
	if len(got) != 1 {
		t.Fatalf("updates = %+v", got)
	}
	if want := []string{"Another-Dep", "New-Dep"}; !slices.Equal(got[0].AddedDeps, want) {
		t.Errorf("added = %v, want %v", got[0].AddedDeps, want)
	}
	if want := []string{"Old-Gone"}; !slices.Equal(got[0].RemovedDeps, want) {
		t.Errorf("removed = %v, want %v", got[0].RemovedDeps, want)
	}
}

type fakeChecker struct {
	asked   *[][]source.InstalledFile
	filters *components.GameSource
	latest  map[string]source.Latest
	deps    map[source.VersionRef][]string
}

func (fakeChecker) ID() string              { return profile.KindModrinth }
func (fakeChecker) Name() string            { return "Modrinth" }
func (fakeChecker) Modes() []source.Acquire { return nil }
func (f fakeChecker) Latest(_ context.Context, src components.GameSource, _ string, files []source.InstalledFile) (map[string]source.Latest, error) {
	*f.asked = append(*f.asked, files)
	*f.filters = src
	return f.latest, nil
}

func (f fakeChecker) Dependencies(context.Context, string, string, []source.VersionRef) (map[source.VersionRef][]string, error) {
	return f.deps, nil
}

func TestModrinthUpdatesAreCheckedInOneBatch(t *testing.T) {
	var asked [][]source.InstalledFile
	var filters components.GameSource
	source.Register(fakeChecker{
		asked: &asked, filters: &filters,
		latest: map[string]source.Latest{
			"sha512:aa": {ProjectID: "AANobbMI", Version: "0.6.0", VersionID: "SoD3", URL: "https://modrinth.com/project/AANobbMI/version/SoD3"},
		},
		deps: map[source.VersionRef][]string{
			{ID: "AANobbMI", Version: "0.5.0"}: {"P1", "P8"},
			{ID: "AANobbMI", Version: "0.6.0"}: {"P1", "P9"},
		},
	})
	t.Cleanup(func() { source.Register(modrinth.Driver{}) })
	mk := func(key, digest, version string) framework.Mod {
		m := framework.Mod{Key: key, SourceKind: profile.KindModrinth, SourceName: key, SourceVersion: version, SourceDigest: digest}
		m.Name, m.Version = key, version
		return m
	}
	turns := 0
	s := &Service{Throttle: func(context.Context, string) (func(), error) { turns++; return func() {}, nil }}
	// No shipped game lists Modrinth yet, so the catalog entry is the test's own.
	src := components.GameSource{ID: profile.KindModrinth, Loaders: []string{"fabric"}, GameVersions: []string{"1.21"}}
	got, err := s.searchUpdates(context.Background(), "test-game", src, []framework.Mod{
		mk("sodium", "sha512:aa", "0.5.0"), mk("lithium", "sha512:bb", "1.0"), mk("nodigest", "", "1.0"),
	}, nil)
	if err != nil || len(asked) != 1 || len(asked[0]) != 2 || turns != 1 {
		t.Fatalf("err %v, asked %v, turns %d", err, asked, turns)
	}
	if !slices.Equal(filters.Loaders, []string{"fabric"}) || !slices.Equal(filters.GameVersions, []string{"1.21"}) {
		t.Errorf("filters = %+v", filters)
	}
	want := Update{
		Key: "sodium", ID: mk("sodium", "", "").ModID(), Name: "sodium", Installed: "0.5.0", Version: "0.6.0", URL: "https://modrinth.com/project/AANobbMI/version/SoD3",
		Source: "Modrinth", Package: "AANobbMI", PackageSource: profile.KindModrinth, PackageVersion: "SoD3",
		AddedDeps: []string{"P9"}, RemovedDeps: []string{"P8"},
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("updates = %+v, want %+v", got, want)
	}
}
