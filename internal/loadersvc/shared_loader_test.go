package loadersvc

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// sharedLoader is one loader id that two games use, each with a pack of its own (BepInEx 5 for Lethal Company and
// Valheim): the fetched file names the game, and each game's latest version differs.
type sharedLoader struct{}

func (sharedLoader) ID() string             { return "w7shared" }
func (sharedLoader) Formats() []string      { return nil }
func (sharedLoader) ProfileFiles() []string { return []string{"w7shared"} }
func (sharedLoader) Status(t loader.Target) (loader.Status, error) {
	b, err := fsx.ReadFile(filepath.Join(t.ProfileDir, "w7shared"))
	return loader.Status{Installed: err == nil, Version: string(b)}, nil
}

func (sharedLoader) Install(_ context.Context, t loader.Target, p loader.Package, _ func(loader.Step)) (string, error) {
	return p.Version, fsx.WriteFile(filepath.Join(t.ProfileDir, "w7shared"), []byte(p.Version), 0o600)
}

func (sharedLoader) Contribute(context.Context, *launchplan.Plan, loader.ProfileView) error {
	return nil
}

func (sharedLoader) Latest(_ context.Context, g components.GameInfo) (string, error) {
	if g.ID == "valheim" {
		return "2.0.0", nil
	}
	return "1.0.0", nil
}

func (sharedLoader) Versions(context.Context, components.GameInfo) ([]string, error) {
	return []string{"1.0.0", "2.0.0"}, nil
}

func (sharedLoader) Fetch(_ context.Context, g components.GameInfo, _, dst string) error {
	return fsx.WriteFile(dst, []byte("pack for "+g.ID), 0o600)
}

var _ = loader.Register(sharedLoader{})

func TestTwoGamesOnOneLoaderKeepTheirOwnInstall(t *testing.T) {
	t.Setenv("MORTAR_ENABLE_GAMES", "lethal-company,valheim")
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("LOCALAPPDATA", data)
	var games []components.GameInfo
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"lethal-company", "valheim"} {
		info, _ := components.Game(id)
		info.Enabled = true
		info.Loaders = []components.GameLoader{{ID: "w7shared", Name: "Shared"}}
		games = append(games, info)
		folder := t.TempDir()
		put(t, filepath.Join(folder, info.Marker), "x")
		if _, err := set.Update(func(v *settings.Settings) { v.GameFolders[id] = folder }); err != nil {
			t.Fatal(err)
		}
	}
	client := components.NewClient(nil)
	client.SetManifest(components.Manifest{Serial: 1, Games: games})
	t.Cleanup(func() { game.ConfigureComponents(nil) })
	items, profiles := testenv.Stores(t)
	svc := testService(t, set, items, profiles, client)
	testenv.Profile(t, profiles, "lethal-company", "Crew")
	testenv.Profile(t, profiles, "valheim", "Vik")

	if st, err := svc.Ensure(t.Context(), "lethal-company", "", false); err != nil || st.Version != "1.0.0" {
		t.Fatalf("lethal company = %+v, %v", st, err)
	}
	if st, err := svc.LocalStatus("valheim", ""); err != nil || st.Installed {
		t.Fatalf("valheim counts Lethal Company's install as its own: %+v, %v", st, err)
	}
	if st, err := svc.Ensure(t.Context(), "valheim", "", false); err != nil || st.Version != "2.0.0" {
		t.Fatalf("valheim = %+v, %v", st, err)
	}
	// Valheim moving to Lethal Company's version keeps two packs: the same version of one loader id is not one file.
	if st, err := svc.InstallVersion(t.Context(), "valheim", "", "1.0.0"); err != nil || st.Version != "1.0.0" {
		t.Fatalf("valheim to 1.0.0 = %+v, %v", st, err)
	}
	rec := set.Get().Loaders
	if rec[settings.LoaderKey("lethal-company", "w7shared")] != "1.0.0" || rec[settings.LoaderKey("valheim", "w7shared")] != "1.0.0" || len(rec) != 2 {
		t.Fatalf("recorded = %v", rec)
	}
	for _, id := range []string{"lethal-company", "valheim"} {
		dir, err := items.Dir(id, store.LoaderKey("w7shared", "1.0.0"))
		if err != nil {
			t.Fatal(err)
		}
		matches, _ := filepath.Glob(filepath.Join(dir, "*"))
		found := false
		for _, m := range matches {
			if b, err := fsx.ReadFile(m); err == nil && string(b) == "pack for "+id {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s's stored pack is not its own: %v", id, matches)
		}
	}
}
