package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/bundles"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/gamesettings"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/testenv"
)

func TestTemplateRoundTripFromProfileToNewProfile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("LOCALAPPDATA", tmp)
	items, profiles := testenv.Stores(t)
	dataDir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte(`{"Name":"One","UniqueID":"A.One","Version":"1.0.0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := items.AddDir("stardew", "local-a", src); err != nil {
		t.Fatal(err)
	}
	from, err := profiles.Create("stardew", "Source")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AddEntry("stardew", from.ID, "local-a", profile.Source{Kind: profile.KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.SetLaunchOptions("stardew", from.ID, "--no-gui"); err != nil {
		t.Fatal(err)
	}
	mode := "borderless"
	stored := map[string]gamesettings.Settings{from.ID: {WindowMode: &mode}}
	svc := NewService(Deps{
		Profiles: profiles,
		Bundles:  bundles.NewService(profiles, dataDir),
		GameSettings: func(_, id string) (gamesettings.Settings, error) {
			return stored[id], nil
		},
		SetGameSettings: func(_, id string, s gamesettings.Settings) error {
			stored[id] = s
			return nil
		},
	}, dataDir)

	saved, err := svc.SaveTemplateFromProfile("stardew", from.ID, " Starter ")
	if err != nil || saved.Name != "Starter" || len(saved.Bundle) != 1 || saved.LaunchOptions != "--no-gui" {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	if _, err := svc.SaveTemplateFromProfile("stardew", from.ID, "starter"); err != nil {
		t.Fatal(err)
	}
	if list, err := svc.Templates("stardew"); err != nil || len(list) != 1 {
		t.Fatalf("list = %+v, %v", list, err)
	}

	res, err := svc.NewProfileFromTemplate("stardew", "Starter", "Fresh")
	if err != nil || res.Added != 1 || len(res.Missing) != 0 || res.Profile.Name != "Fresh" {
		t.Fatalf("result = %+v, %v", res, err)
	}
	if opts, _ := profiles.LaunchOptions("stardew", res.Profile.ID); opts != "--no-gui" {
		t.Fatalf("launch options = %q", opts)
	}
	if got := stored[res.Profile.ID]; got.WindowMode == nil || *got.WindowMode != "borderless" {
		t.Fatalf("settings = %+v", got)
	}

	if _, err := svc.NewProfileFromTemplate("stardew", "Nope", "X"); err == nil {
		t.Fatal("unknown template must fail")
	}
	if err := svc.DeleteTemplate("stardew", "STARTER"); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTemplate("stardew", "Starter"); err == nil {
		t.Fatal("second delete must fail")
	}
}
