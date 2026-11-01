package control

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/savedgroups"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestGroupsCreateNamesAThunderstoreModByItsBareID(t *testing.T) {
	s, items := servicesWithItems(t)
	const gameID = "lethal-company"
	prof, err := s.Profiles.Create(gameID, "Crew")
	if err != nil {
		t.Fatal(err)
	}
	files := t.TempDir()
	for name, body := range map[string]string{"A.dll": "a", "manifest.json": `{"name":"A","namespace":"Ns","version_number":"1.0.0","dependencies":[]}`} {
		if err := os.WriteFile(filepath.Join(files, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	key := store.PackageKey("Ns-A", "1.0.0")
	if err := items.AddDir(t.Context(), gameID, key, files); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.AddEntry(gameID, prof.ID, key, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-A", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	s.SavedGroups = savedgroups.NewService(s.Store, filepath.Join(t.TempDir(), "data"))

	res, err := s.savedGroupsWrite("groups.create", Params{Game: gameID, Profile: prof.ID, Name: "Core", IDs: []string{"Ns-A"}})
	if err != nil {
		t.Fatal(err)
	}
	g, ok := res.(savedgroups.SavedGroup)
	if !ok || len(g.Mods) != 1 || g.Mods[0].ID != mod.ID("thunderstore:Ns-A") {
		t.Fatalf("created group = %#v", res)
	}
}

func TestSavedGroupsWriteNeedsTheService(t *testing.T) {
	t.Parallel()
	_, err := (&Services{}).savedGroupsWrite("groups.create", Params{})
	if err == nil || err.Error() != "saved groups are unavailable" {
		t.Fatalf("err = %v", err)
	}
}

func TestTypedIDsKeepsOrderAndMixesKnownAndNew(t *testing.T) {
	t.Parallel()
	known := []mod.ID{"thunderstore:Ns-A", "thunderstore:Ns-B"}
	got, err := typedIDs(known, []string{"Ns-B", "Ns-C", "smapi:X.Y"})
	if err != nil {
		t.Fatal(err)
	}
	want := []mod.ID{"thunderstore:Ns-B", "thunderstore:Ns-C", "smapi:X.Y"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("typedIDs[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
