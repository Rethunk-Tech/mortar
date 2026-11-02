package nexussvc

import (
	"context"
	"errors"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/vsmoddb"
)

type fakeModDB map[string]vsmoddb.Resolved

func (f fakeModDB) Resolve(_ context.Context, id, _ string) (vsmoddb.Resolved, error) {
	if r, ok := f[id]; ok {
		return r, nil
	}
	return vsmoddb.Resolved{}, errors.New("not on the Mod DB")
}

func TestModDBSourcesMatchOneModIDAndSkipAmbiguousOrUnknown(t *testing.T) {
	local := profile.Source{Kind: profile.KindLocal}
	entry := func(key string, src profile.Source, ids ...string) profile.Entry {
		e := profile.Entry{Key: key, Source: src}
		for _, id := range ids {
			e.Mods = append(e.Mods, profile.Component{ModID: id, Version: "1.0.0"})
		}
		return e
	}
	got := modDBSources(t.Context(), []profile.Entry{
		entry("carryon", local, "carryon"),
		entry("pair", local, "carryon", "carryon"),
		entry("split", local, "carryon", "chiseltools"),
		entry("gone", local, "nowhere"),
		entry("sourced", profile.Source{Kind: profile.KindModrinth}, "carryon"),
	}, fakeModDB{
		"carryon":     {ID: "890-4405", ReleaseID: 7},
		"chiseltools": {ID: "604-2751"},
	})
	if len(got) != 2 || got["carryon"].Name != "890-4405" || got["carryon"].Kind != profile.KindVSModDB || got["pair"].Name != "890-4405" {
		t.Fatalf("got %+v", got)
	}
}

func TestThunderstoreSourcesNameTheTwinOfASourcelessPackage(t *testing.T) {
	got := thunderstoreSources([]profile.Entry{
		{Key: "a", Package: true, Source: profile.Source{Kind: profile.KindLocal}},
		{Key: "b", Package: true, Source: profile.Source{Kind: profile.KindGitHub}},
		{Key: "c", Package: true, Source: profile.Source{Kind: profile.KindLocal}},
	}, map[string]string{"a": "Owner-Pack", "b": "Owner-Other"})
	if len(got) != 1 || got["a"].Name != "Owner-Pack" || got["a"].Kind != profile.KindThunderstore {
		t.Fatalf("got %+v", got)
	}
}
