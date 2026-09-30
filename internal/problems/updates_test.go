package problems

import (
	"context"
	"reflect"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/meta"
)

type recordingMeta struct {
	fakeMeta
	got meta.UpdateRequest
}

func (r *recordingMeta) CheckUpdates(ctx context.Context, req meta.UpdateRequest) []meta.UpdateResult {
	r.got = req
	return r.fakeMeta.CheckUpdates(ctx, req)
}

func TestCheckUpdates(t *testing.T) {
	bundled := mod("smapi-4", "SMAPI.ConsoleCommands", "4.0.0", true)
	bundled.SourceKind = "smapi"
	off := mod("k2", "me.off", "1.0.0", false)
	rm := &recordingMeta{fakeMeta: fakeMeta{compat: map[string]meta.UpdateResult{
		"me.a":   {Known: true, Suggested: &meta.Update{Version: "2.0.0", URL: "https://example.test/a"}},
		"me.off": {Known: true, Suggested: &meta.Update{Version: "1.1.0", URL: "https://example.test/off"}},
		"me.b":   {Known: true},
	}}}
	env := Environment{GameVersion: "1.6.15", APIVersion: "4.3.2", Platform: "Linux"}
	got := CheckUpdates(context.Background(), rm, env, []Installed{bundled, mod("k1", "me.a", "1.0.0", true), mod("k3", "me.b", "1.0.0", true), off})
	want := []Update{
		{Key: "k1", UniqueID: "me.a", Name: "me.a", Installed: "1.0.0", Version: "2.0.0", URL: "https://example.test/a"},
		{Key: "k2", UniqueID: "me.off", Name: "me.off", Installed: "1.0.0", Version: "1.1.0", URL: "https://example.test/off"},
	}
	if !reflect.DeepEqual(got.Updates, want) || got.Unknown {
		t.Fatalf("got %+v", got)
	}
	if rm.got.APIVersion != "4.3.2" || rm.got.GameVersion != "1.6.15" || rm.got.Platform != "Linux" || len(rm.got.Mods) != 3 {
		t.Fatalf("asked %+v", rm.got)
	}
}

func TestCheckUpdatesUnknownNeverBlocks(t *testing.T) {
	got := CheckUpdates(context.Background(), fakeMeta{updatesOff: true}, Environment{}, []Installed{mod("k1", "me.a", "1.0.0", true)})
	if !got.Unknown || len(got.Updates) != 0 {
		t.Fatalf("got %+v", got)
	}
	none := CheckUpdates(context.Background(), fakeMeta{}, Environment{}, nil)
	if none.Unknown || none.Updates == nil {
		t.Fatalf("empty = %+v", none)
	}
}

func TestRelate(t *testing.T) {
	a := mod("k1", "me.a", "1.0.0", true, req("me.core", "2.0.0"), manifest.Dependency{UniqueID: "me.opt"})
	a.UpdateKeys = []string{"Chucklefish:1", "Nexus:42@x"}
	core := mod("k2", "me.core", "1.0.0", true)
	user := mod("k3", "me.user", "1.0.0", true, req("me.a", ""))
	r, ok := Relate([]Installed{a, core, user}, "k1", "me.a")
	if !ok || r.PageURL != "https://www.nexusmods.com/stardewvalley/mods/42" {
		t.Fatalf("relations = %+v, %v", r, ok)
	}
	states := []Need{
		{UniqueID: "me.core", Name: "me.core", MinimumVersion: "2.0.0", Required: true, State: "outdated", InstalledVersion: "1.0.0"},
		{UniqueID: "me.opt", Name: "me.opt", State: "absent"},
	}
	if !reflect.DeepEqual(r.Needs, states) {
		t.Fatalf("needs = %+v", r.Needs)
	}
	if !reflect.DeepEqual(r.NeededBy, []Dependent{{Key: "k3", UniqueID: "me.user", Name: "me.user"}}) {
		t.Fatalf("neededBy = %+v", r.NeededBy)
	}
	if _, ok := Relate([]Installed{a}, "k9", "me.a"); ok {
		t.Fatal("unknown key related")
	}
}

func TestPageURL(t *testing.T) {
	cases := map[string]string{"GitHub:me/repo": "https://github.com/me/repo", "GitHub:me": "", "ModDrop:5": ""}
	for key, want := range cases {
		if got := pageURL([]string{key}); got != want {
			t.Errorf("pageURL(%s) = %q, want %q", key, got, want)
		}
	}
}
