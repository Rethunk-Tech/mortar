//go:build !windows

package launchsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestTestCopiesAddTheDevHelperAndOnlyTheyLaunchWithIt(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	const lc, helper = "lethal-company", "Ns-Helper"
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	for i := range m.Games {
		if m.Games[i].ID != lc {
			continue
		}
		for j := range m.Games[i].Loaders {
			if tpl := m.Games[i].Loaders[j].DevTemplate; tpl != nil {
				cp := *tpl
				cp.DevHelper = &components.DevRecommend{Package: helper}
				m.Games[i].Loaders[j].DevTemplate = &cp
			}
		}
	}
	c := components.NewClient(nil)
	c.SetManifest(m)
	components.Use(c)
	t.Cleanup(func() { components.Use(nil) })

	_, ps := testenv.Stores(t)
	p := testenv.Profile(t, ps, lc, "Friends")
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), set, ps)
	svc.procDir = t.TempDir()
	svc.EnsureLoader = func(context.Context, string, string, bool) error { return errors.New("stop here") }
	installs := 0
	svc.InstallPackage = func(ctx context.Context, gameID, profileID, pkg string) error {
		installs++
		zip := tsPackage(t, "Helper", map[string]string{"plugins/Helper.dll": "helper"})
		_, err := ps.InstallSource(ctx, gameID, profileID, zip, profile.Source{Kind: profile.KindThunderstore, Name: pkg, Version: "1.0.0"})
		return err
	}

	for range 2 {
		_ = svc.StartInstances(t.Context(), lc, p.ID, 2)
	}
	got, err := ps.Get(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if installs != 1 || len(got.Entries) != 1 || !got.Entries[0].DevOnly {
		t.Fatalf("installs %d, entries %+v: want the helper added once and marked development-only", installs, got.Entries)
	}

	dir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	testfs.WriteFile(t, dir, "winhttp.dll", "proxy")
	testfs.WriteFile(t, dir, "doorstop_config.ini", "ini")
	g := game.Find(lc)
	inst := game.Install{ID: "lc1", Dir: t.TempDir()}
	dll := filepath.Join(dir, "BepInEx", "plugins", helper, "Helper.dll")
	for _, tc := range []struct {
		name       string
		testCopies bool
		wantPlaced bool
	}{
		{"test copies", true, true},
		{"a normal launch", false, false},
	} {
		ctx := t.Context()
		if tc.testCopies {
			ctx = context.WithValue(ctx, instancesKey{}, 2)
		}
		if _, err := svc.planProfile(ctx, g, inst, p.ID, launchplan.ModeProfile, "", "", ""); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(dll); (err == nil) != tc.wantPlaced {
			t.Errorf("%s: helper placed = %v, want %v", tc.name, err == nil, tc.wantPlaced)
		}
	}
}

func TestStartInstancesTouchesNothingWhileTheInstallIsBusy(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	const lc = "lethal-company"
	_, ps := testenv.Stores(t)
	p := testenv.Profile(t, ps, lc, "Friends")
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), set, ps)
	helped := false
	svc.InstallPackage = func(context.Context, string, string, string) error { helped = true; return nil }
	g := game.Find(lc)
	svc.preparing[keyOf(svc.profileSlot(g, p.ID))] = "other"
	before := ps.SeparateSaves(lc, p.ID)
	if err := svc.StartInstances(t.Context(), lc, p.ID, 2); err == nil {
		t.Fatal("a busy install was started")
	}
	if helped || ps.SeparateSaves(lc, p.ID) != before {
		t.Fatalf("a refused start changed the profile (helper %v, separate saves %v -> %v)", helped, before, ps.SeparateSaves(lc, p.ID))
	}
}
