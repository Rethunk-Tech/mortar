package loadersvc

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

type fakeSMAPI struct {
	releases  []string
	installed string
	downloads int
}

func testServiceWithReleases(t *testing.T, releases []string) (*Service, *fakeSMAPI) {
	t.Helper()
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("LOCALAPPDATA", data)
	folder := t.TempDir()
	put(t, filepath.Join(folder, "Stardew Valley.dll"), "x")
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders["stardew"] = folder }); err != nil {
		t.Fatal(err)
	}
	items, profiles := testenv.Stores(t)
	svc := testService(t, set, items, profiles)
	fake := &fakeSMAPI{releases: releases}
	svc.listVersions = func(context.Context, string) ([]string, error) { return fake.releases, nil }
	svc.fetchInstall = func(_ context.Context, _, version string) error {
		fake.downloads++
		fake.installed = version
		return nil
	}
	return svc, fake
}

func TestEnsureHonoursPin(t *testing.T) {
	svc, fake := testServiceWithReleases(t, []string{"4.1.0", "4.0.0"})
	var applyErr error
	if _, err := svc.settings.Update(func(v *settings.Settings) { applyErr = settings.ApplyKeyGame(v, "smapiPin", "4.0.0", "stardew") }); err != nil {
		t.Fatal(err)
	}
	if applyErr != nil {
		t.Fatal(applyErr)
	}
	if _, err := svc.Ensure(context.Background(), "stardew", "", false); err != nil {
		t.Fatal(err)
	}
	if fake.installed != "4.0.0" {
		t.Fatalf("installed %q, want pin 4.0.0", fake.installed)
	}
}

func TestInstallVersionUsesCache(t *testing.T) {
	svc, fake := testServiceWithReleases(t, []string{"4.1.0", "4.0.0"})
	dir := t.TempDir()
	testfs.WriteFile(t, dir, "ok", "x")
	if err := svc.items.AddDir("stardew", store.LoaderKey("smapi", "4.0.0"), dir); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.InstallVersion(context.Background(), "stardew", "", "4.0.0"); err != nil {
		t.Fatal(err)
	}
	if fake.downloads != 0 {
		t.Fatalf("downloads %d, want cached apply", fake.downloads)
	}
}

func TestInstallVersionUnknown(t *testing.T) {
	svc, _ := testServiceWithReleases(t, []string{"4.1.0"})
	_, err := svc.InstallVersion(context.Background(), "stardew", "", "9.9.9")
	if !errors.Is(err, errUnknownVersion) {
		t.Fatalf("err = %v, want unknown SMAPI", err)
	}
}
