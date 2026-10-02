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
	rm := &recordingMeta{}
	rm.compat = map[string]meta.UpdateResult{
		"me.a":   {Known: true, Suggested: &meta.Update{Version: "2.0.0", URL: "https://example.test/a"}},
		"me.off": {Known: true, Suggested: &meta.Update{Version: "1.1.0", URL: "https://example.test/off"}},
		"me.b":   {Known: true},
	}
	env := Environment{GameVersion: "1.6.15", APIVersion: "4.3.2", Platform: "Linux"}
	got := CheckUpdates(context.Background(), rm, env, []Installed{bundled, mod("k1", "me.a", "1.0.0", true), mod("k3", "me.b", "1.0.0", true), off}, false)
	want := []Update{
		{Key: "k1", UniqueID: "me.a", Name: "me.a", Installed: "1.0.0", Version: "2.0.0", URL: "https://example.test/a", Source: "example.test"},
		{Key: "k2", UniqueID: "me.off", Name: "me.off", Installed: "1.0.0", Version: "1.1.0", URL: "https://example.test/off", Source: "example.test"},
	}
	if !reflect.DeepEqual(got.Updates, want) || got.Unknown {
		t.Fatalf("got %+v", got)
	}
	if rm.got.APIVersion != "4.3.2" || rm.got.GameVersion != "1.6.15" || rm.got.Platform != "Linux" || len(rm.got.Mods) != 3 {
		t.Fatalf("asked %+v", rm.got)
	}
}

func TestCheckUpdatesNamesTheGitHubRepo(t *testing.T) {
	gh := mod("k1", "me.a", "1.0.0", true)
	gh.UpdateKeys = []string{"Nexus:5", "GitHub:me/a"}
	elsewhere := mod("k2", "me.b", "1.0.0", true)
	elsewhere.UpdateKeys = []string{"GitHub:me/b"}
	rm := fakeMeta{compat: map[string]meta.UpdateResult{
		"me.a": {Known: true, Suggested: &meta.Update{Version: "2.0.0", URL: "https://github.com/me/a/releases/tag/2.0.0"}},
		"me.b": {Known: true, Suggested: &meta.Update{Version: "2.0.0", URL: "https://www.nexusmods.com/stardewvalley/mods/9"}},
	}}
	got := CheckUpdates(context.Background(), rm, Environment{}, []Installed{gh, elsewhere}, false).Updates
	if len(got) != 2 || got[0].GitHubRepo != "me/a" || got[0].NexusID != 5 || got[1].GitHubRepo != "" {
		t.Fatalf("got %+v", got)
	}
}

func TestCheckUpdatesIncludesUnofficialWithoutReplacingSuggested(t *testing.T) {
	rm := fakeMeta{compat: map[string]meta.UpdateResult{
		"me.a": {
			Known: true, Suggested: &meta.Update{Version: "2.0.0", URL: "https://n.test/a"},
			Unofficial: &meta.Update{Version: "2.1.0-unofficial.1-x", URL: "https://smapi.io/u"},
		},
	}}
	got := CheckUpdates(context.Background(), rm, Environment{}, []Installed{mod("k1", "me.a", "1.0.0", true)}, false).Updates
	if len(got) != 2 || got[0].Unofficial || got[0].Version != "2.0.0" || !got[1].Unofficial || got[1].Version != "2.1.0-unofficial.1-x" {
		t.Fatalf("got %+v", got)
	}
}

func TestHideHeldDropsPinnedAndSkipped(t *testing.T) {
	r := UpdatesResult{Updates: []Update{
		{Key: "pin", Version: "2.0.0"},
		{Key: "skip", Version: "2.0.0"},
		{Key: "later", Version: "2.1.0"},
		{Key: "open", Version: "2.0.0"},
	}}
	got := HideHeld(r, []Installed{
		{Key: "pin", Pinned: true},
		{Key: "skip", SkipVersion: "2.0.0"},
		{Key: "later", SkipVersion: "2.0.0"},
		{Key: "open"},
	}, true)
	if len(got.Updates) != 2 || got.Updates[0].Key != "later" || got.Updates[1].Key != "open" {
		t.Fatalf("got %+v", got.Updates)
	}
}

func TestHideHeldDropsSkippedSources(t *testing.T) {
	r := UpdatesResult{Updates: []Update{
		{Key: "nexus", Version: "2.0.0", Source: "Nexus"},
		{Key: "github", Version: "2.0.0", Source: "GitHub"},
	}}
	got := HideHeld(r, []Installed{
		{Key: "nexus", SkipSources: []string{"Nexus"}},
		{Key: "github"},
	}, true)
	if len(got.Updates) != 1 || got.Updates[0].Key != "github" {
		t.Fatalf("got %+v", got.Updates)
	}
}

func TestUpdateSource(t *testing.T) {
	tests := []struct {
		update meta.Update
		nexus  int
		github string
		want   string
	}{
		{meta.Update{Source: "CurseForge:123"}, 0, "", "CurseForge"},
		{meta.Update{}, 0, "owner/repo", "GitHub"},
		{meta.Update{}, 42, "", "Nexus"},
		{meta.Update{URL: "https://www.moddrop.com/stardew/a"}, 0, "", "ModDrop"},
	}
	for _, test := range tests {
		if got := updateSource(test.update, test.nexus, test.github); got != test.want {
			t.Errorf("updateSource(%+v) = %q, want %q", test.update, got, test.want)
		}
	}
}

func TestCheckUpdatesUnknownNeverBlocks(t *testing.T) {
	got := CheckUpdates(context.Background(), fakeMeta{updatesOff: true}, Environment{}, []Installed{mod("k1", "me.a", "1.0.0", true)}, false)
	if !got.Unknown || len(got.Updates) != 0 {
		t.Fatalf("got %+v", got)
	}
	none := CheckUpdates(context.Background(), fakeMeta{}, Environment{}, nil, false)
	if none.Unknown || none.Updates == nil {
		t.Fatalf("empty = %+v", none)
	}
}

func TestCheckUpdatesEnabledOnlySkipsDisabled(t *testing.T) {
	rm := &recordingMeta{}
	rm.compat = map[string]meta.UpdateResult{
		"me.on":  {Known: true, Suggested: &meta.Update{Version: "2.0.0", URL: "https://example.test/on"}},
		"me.off": {Known: true, Suggested: &meta.Update{Version: "2.0.0", URL: "https://example.test/off"}},
	}
	mods := []Installed{
		mod("k1", "me.on", "1.0.0", true),
		mod("k2", "me.off", "1.0.0", false),
	}
	got := CheckUpdates(context.Background(), rm, Environment{}, mods, true)
	if len(rm.got.Mods) != 1 || rm.got.Mods[0].ID != "me.on" {
		t.Fatalf("asked %+v", rm.got)
	}
	if len(got.Updates) != 1 || got.Updates[0].UniqueID != "me.on" {
		t.Fatalf("got %+v", got)
	}
}

func TestHideHeldDropsPrereleaseUnlessInstalledIsPrerelease(t *testing.T) {
	r := UpdatesResult{Updates: []Update{
		{Key: "stable", Installed: "1.0.0", Version: "2.0.0-beta", URL: "https://example.test/beta"},
		{Key: "beta", Installed: "1.0.0-beta", Version: "2.0.0-beta", URL: "https://example.test/beta2"},
		{Key: "release", Installed: "1.0.0", Version: "2.0.0", URL: "https://example.test/stable"},
	}}
	mods := []Installed{{Key: "stable"}, {Key: "beta"}, {Key: "release"}}
	got := HideHeld(r, mods, false)
	if len(got.Updates) != 2 || got.Updates[0].Key != "beta" || got.Updates[1].Key != "release" {
		t.Fatalf("got %+v", got.Updates)
	}
	gotAll := HideHeld(r, mods, true)
	if len(gotAll.Updates) != 3 {
		t.Fatalf("include prerelease = %+v", gotAll.Updates)
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

func TestPagesKeysByEntryAndID(t *testing.T) {
	with, without := Installed{Key: "k1"}, Installed{Key: "k2"}
	with.UniqueID, with.UpdateKeys = "me.a", []string{"GitHub:me/repo"}
	without.UniqueID = "me.b"
	got := Pages([]Installed{with, without})
	if len(got) != 1 || got["k1/me.a"] != "https://github.com/me/repo" {
		t.Fatalf("got %v", got)
	}
}
