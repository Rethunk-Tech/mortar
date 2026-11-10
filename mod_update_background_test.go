//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// The background mod update scan walks the profiles of the games that are installed and reads, skipping a game nobody
// installed and a profile that failed to load, so one broken profile never stops the scan.
func TestProfileModUpdatesScansInstalledGamesProfiles(t *testing.T) {
	settingsStore, profiles := testenv.Env(t)
	home := t.TempDir()
	install := t.TempDir()
	testfs.WriteFile(t, install, "Stardew Valley.dll", "x")
	if _, err := settingsStore.Update(func(s *settings.Settings) {
		s.GameFolders["stardew"] = install
	}); err != nil {
		t.Fatal(err)
	}
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	broken := testenv.Profile(t, profiles, "stardew", "Broken")
	dir, err := profiles.ProfileDir("stardew", broken.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.json"), []byte("{ not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := modUpdateScanSource{
		gamesSvc:    game.NewService(home, settingsStore),
		profiles:    profiles,
		problemsSvc: problems.NewService(home, settingsStore, profiles, &meta.Client{CacheDir: t.TempDir()}),
	}
	out, err := s.ProfileModUpdates(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Game != "stardew" || out[0].ProfileID != p.ID || out[0].ProfileName != "Farm" {
		t.Fatalf("scan = %+v, want the readable profile of the installed game", out)
	}
}
