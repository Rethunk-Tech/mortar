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
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// fakeLoader is a second loader that lives in each profile: a marker file stands for its files.
type fakeLoader struct{}

func (fakeLoader) ID() string        { return "w6fake" }
func (fakeLoader) Formats() []string { return nil }
func (fakeLoader) InProfile()        {}
func (fakeLoader) Status(t loader.Target) (loader.Status, error) {
	b, err := fsx.ReadFile(filepath.Join(t.ProfileDir, "w6fake"))
	return loader.Status{Installed: err == nil, Version: string(b)}, nil
}

func (fakeLoader) Install(_ context.Context, t loader.Target, p loader.Package, _ func(loader.Step)) (string, error) {
	return p.Version, fsx.WriteFile(filepath.Join(t.ProfileDir, "w6fake"), []byte(p.Version), 0o600)
}

func (fakeLoader) Contribute(context.Context, *launchplan.Plan, loader.ProfileView) error {
	return nil
}

func (fakeLoader) Latest(context.Context, components.GameInfo) (string, error) { return "1.0.0", nil }

func (fakeLoader) Versions(context.Context, components.GameInfo) ([]string, error) {
	return []string{"1.0.0"}, nil
}

func (fakeLoader) Fetch(_ context.Context, _ components.GameInfo, _, dst string) error {
	return fsx.WriteFile(dst, []byte("zip"), 0o600)
}

var _ = loader.Register(fakeLoader{})

// A catalog game that lists two loaders: each profile runs its own, and an install touches only the profiles that run it.
func TestAProfileRunsItsOwnLoaderOfTheGamesTwo(t *testing.T) {
	t.Setenv("MORTAR_ENABLE_GAMES", "lethal-company")
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("LOCALAPPDATA", data)
	info, _ := components.BundledGame("lethal-company")
	info.Enabled = true
	info.Loaders = append(info.Loaders, components.GameLoader{ID: "w6fake", Name: "Fake"})
	client := components.NewClient(nil)
	client.SetManifest(components.Manifest{Serial: 1, Games: []components.GameInfo{info}})
	t.Cleanup(func() { game.ConfigureComponents(nil) })

	folder := t.TempDir()
	put(t, filepath.Join(folder, "Lethal Company.exe"), "x")
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders["lethal-company"] = folder }); err != nil {
		t.Fatal(err)
	}
	items, profiles := testenv.Stores(t)
	svc := NewService(t.TempDir(), set, items, profiles, client)
	a := testenv.Profile(t, profiles, "lethal-company", "A")
	b := testenv.Profile(t, profiles, "lethal-company", "B")
	if _, err := profiles.SetLoader("lethal-company", b.ID, "nope"); err == nil {
		t.Fatal("a loader the game does not list was accepted")
	}
	if _, err := profiles.SetLoader("lethal-company", b.ID, "w6fake"); err != nil {
		t.Fatal(err)
	}
	if got := profiles.LoaderID("lethal-company", a.ID); got != "bepinex5" {
		t.Fatalf("default loader = %q", got)
	}

	st, err := svc.Ensure(t.Context(), "lethal-company", "w6fake", false)
	if err != nil || !st.Installed || st.Version != "1.0.0" {
		t.Fatalf("ensure = %+v, %v", st, err)
	}
	for id, want := range map[string]bool{a.ID: false, b.ID: true} {
		dir, _ := profiles.ProfileDir("lethal-company", id)
		if got, _ := (fakeLoader{}).Status(loader.Target{ProfileDir: dir}); got.Installed != want {
			t.Fatalf("profile %s has the second loader: %v, want %v", id, got.Installed, want)
		}
	}
	if set.Get().Loaders["w6fake"] != "1.0.0" || set.Get().Loaders["bepinex5"] != "" {
		t.Fatalf("recorded versions = %v", set.Get().Loaders)
	}
}
