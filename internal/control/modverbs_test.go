package control

import (
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// installedFarm makes a profile holding one installed mod.
func installedFarm(t *testing.T, s *Services) profile.Profile {
	t.Helper()
	p, err := s.Profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Store.InstallSource(t.Context(), "stardew", p.ID, writeModZip(t, "Mod", "Test.Alpha", "Alpha"),
		profile.Source{Kind: profile.KindNexus, Name: "alpha.zip", ModID: 12, FileID: 2, Version: "1.0"})
	if err != nil {
		t.Fatal(err)
	}
	return res.Profile
}

func TestModsGroupNeedsAModAndUsesItsFirstKey(t *testing.T) {
	s := services(t)
	prof := installedFarm(t, s)
	id := string(prof.Entries[0].Mods[0].ID)
	key := prof.Entries[0].Key

	if _, err := s.modsGroup(Params{Sub: "add", Name: "Core"}, prof, prof.ID); err == nil || err.Error() != "mods group add needs a mod id" {
		t.Fatalf("add without a mod: %v", err)
	}
	if _, err := s.modsGroup(Params{Sub: "remove", Name: "Core"}, prof, prof.ID); err == nil || err.Error() != "mods group remove needs a mod id" {
		t.Fatalf("remove without a mod: %v", err)
	}
	if _, err := s.modsGroup(Params{Sub: "nope"}, prof, prof.ID); err == nil || err.Error() != "mods group needs list, create, rename, delete, add, remove, on, or off" {
		t.Fatalf("an unknown sub: %v", err)
	}

	res, err := s.modsGroup(Params{Game: "stardew", Sub: "create", Name: "Core"}, prof, prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if created, ok := res.(profile.Profile); !ok || len(created.Groups) != 1 || created.Groups[0].Name != "Core" {
		t.Fatalf("create group = %#v", res)
	}
	prof, err = s.Profiles.Get("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.modsGroup(Params{Game: "stardew", Sub: "add", Name: "Core", IDs: []string{id}}, prof, prof.ID); err != nil {
		t.Fatal(err)
	}
	prof, err = s.Profiles.Get("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(prof.Groups) != 1 || !slices.Equal(prof.Groups[0].Keys, []string{key}) {
		t.Fatalf("group after add = %+v, want the entry key %q", prof.Groups, key)
	}
	if _, err := s.modsGroup(Params{Game: "stardew", Sub: "remove", Name: "Core", IDs: []string{id}}, prof, prof.ID); err != nil {
		t.Fatal(err)
	}
	prof, err = s.Profiles.Get("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(prof.Groups) != 1 || len(prof.Groups[0].Keys) != 0 {
		t.Fatalf("group after remove = %+v", prof.Groups)
	}
}

func TestModsGroupListIsAnEmptyListNotNil(t *testing.T) {
	res, err := (&Services{}).modsGroup(Params{Sub: "list"}, profile.Profile{}, "")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := res.([]profile.Group)
	if !ok || got == nil || len(got) != 0 {
		t.Fatalf("a profile without groups lists %#v", res)
	}
}

func TestModsPresetRefusesAnEmptyName(t *testing.T) {
	s := services(t)
	prof := installedFarm(t, s)
	id := string(prof.Entries[0].Mods[0].ID)

	if _, err := s.modsPreset(Params{Sub: "save"}, prof.ID, prof); err == nil || err.Error() != "mods preset needs a mod" {
		t.Fatalf("save without a mod: %v", err)
	}
	for _, tc := range []struct{ sub, want string }{
		{"save", "mods preset save needs a name"},
		{"apply", "mods preset apply needs a name"},
		{"delete", "mods preset delete needs a name"},
	} {
		if _, err := s.modsPreset(Params{Sub: tc.sub, IDs: []string{id}}, prof.ID, prof); err == nil || err.Error() != tc.want {
			t.Fatalf("%s without a name: %v", tc.sub, err)
		}
	}
	if _, err := s.modsPreset(Params{Sub: "nope", IDs: []string{id}}, prof.ID, prof); err == nil || err.Error() != "mods preset needs list, save, apply, or delete" {
		t.Fatalf("an unknown sub: %v", err)
	}
}

func TestModsChannelSetsAndClearsTheChannel(t *testing.T) {
	s := services(t)
	prof := installedFarm(t, s)
	id := string(prof.Entries[0].Mods[0].ID)

	if _, err := s.modsChannel(Params{}, prof, prof.ID); err == nil || err.Error() != "mods channel needs main, optional, or beta" {
		t.Fatalf("without a channel: %v", err)
	}
	if res, err := s.modsChannel(Params{Game: "stardew", Value: "beta", IDs: []string{id}}, prof, prof.ID); err != nil {
		t.Fatalf("beta: %v", err)
	} else if rows, ok := res.([]ModRow); !ok || len(rows) != 1 {
		t.Fatalf("beta = %#v", res)
	}
	got, err := s.Profiles.Get("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].UpdateChannel == "" {
		t.Fatal("beta was not recorded")
	}
	if _, err := s.modsChannel(Params{Game: "stardew", Value: "main", IDs: []string{id}}, got, prof.ID); err != nil {
		t.Fatal(err)
	}
	got, err = s.Profiles.Get("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].UpdateChannel != "" {
		t.Fatalf("main must clear the recorded channel, got %q", got.Entries[0].UpdateChannel)
	}
}

func TestModsSplitAndWinNameWhatTheyNeed(t *testing.T) {
	s := services(t)
	prof := installedFarm(t, s)
	id := string(prof.Entries[0].Mods[0].ID)

	if _, err := s.modsSplit(Params{IDs: []string{id}}, prof, prof.ID); err == nil || err.Error() != "mods split needs a mod and a file" {
		t.Fatalf("split with one id: %v", err)
	}
	if _, err := s.modsSplit(Params{IDs: []string{id, "a.package", "b.package"}}, prof, prof.ID); err == nil || err.Error() != "mods split needs a mod and a file" {
		t.Fatalf("split with three ids: %v", err)
	}
	if _, err := s.modsWin(Params{IDs: []string{id}}, prof, prof.ID); err == nil || err.Error() != "mods win needs a winner and a loser" {
		t.Fatalf("win with one id: %v", err)
	}
}

func TestExtraKeyOfMatchesTheKeyOrAFolderUnderIt(t *testing.T) {
	e := profile.Entry{
		Key:            "pkg-1#a.package",
		ExtraStoreKeys: []string{"pkg-1"},
		Mods:           []profile.Component{{ID: "smapi:Pack.Sub", Name: "Sub", Folder: "pkg-1/Sub"}},
	}
	for _, id := range []string{"pkg-1", "smapi:Pack.Sub", "Sub"} {
		if got, err := extraKeyOf(e, id); err != nil || got != "pkg-1" {
			t.Fatalf("extraKeyOf(%q) = %q, %v", id, got, err)
		}
	}
	if _, err := extraKeyOf(e, "pkg-2"); err == nil {
		t.Fatal("a mod outside the entry's extra files must be refused")
	}
}

func TestModsConfigNeedsAKnownMod(t *testing.T) {
	s := services(t)
	prof := installedFarm(t, s)
	if _, err := s.Handle(t.Context(), "mods.config", Params{Game: "stardew", Profile: prof.ID}); err == nil || err.Error() != "mods config needs a mod" {
		t.Fatalf("without a mod: %v", err)
	}
	if _, err := s.Handle(t.Context(), "mods.config", Params{Game: "stardew", Profile: prof.ID, IDs: []string{"Test.Nope"}}); err == nil || !strings.Contains(err.Error(), "has no mod") {
		t.Fatalf("an unknown mod: %v", err)
	}
}

func TestSetModFavouritesRefusesAModWithNoSite(t *testing.T) {
	s := services(t)
	s.SettingsSvc = settings.NewService(s.Settings)
	prof := profile.Profile{ID: "p1", Name: "Farm", Entries: []profile.Entry{{
		Key:    "local-1",
		Source: profile.Source{Kind: profile.KindLocal},
		Mods:   []profile.Component{{ID: "smapi:Local.Mod", Name: "Local"}},
	}}}
	if _, err := s.setModFavourites("stardew", prof, []string{"smapi:Local.Mod"}, true); err == nil || !strings.Contains(err.Error(), "came from no mod site, so it cannot be starred") {
		t.Fatalf("a local mod with no name: %v", err)
	}
	if got := s.Settings.Get().GamePrefs("stardew").FavouriteMods; len(got) != 0 {
		t.Fatalf("a refused star was still stored: %v", got)
	}
}
