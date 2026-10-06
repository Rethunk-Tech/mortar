//go:build !windows

package launchsvc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestRunForBisectStopsWhenStartupReportAppears(t *testing.T) {
	t.Parallel()
	svc, p := parallelStartEnv(t)
	if _, err := svc.profiles.SetOverride("stardew", p.ID, "defaultLaunchMethod", settings.LaunchDirect); err != nil {
		t.Fatal(err)
	}
	folder := svc.settings.Get().GameFolders["stardew"]
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.profiles.SetLaunchSettings("stardew", p.ID, "", fakeGameEnvLines); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"StardewValley", "StardewValley-original", "StardewModdingAPI"} {
		if err := os.Symlink(self, filepath.Join(folder, name)); err != nil {
			t.Fatal(err)
		}
	}
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(mods, "bridge-1.3.0", bridge.SMAPI.ModFolder), 0o700); err != nil {
		t.Fatal(err)
	}
	prof, err := svc.profiles.ProfileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	startup := filepath.Join(prof, startupDir)
	if err := os.MkdirAll(startup, 0o700); err != nil {
		t.Fatal(err)
	}
	svc.EnsureLoader = func(context.Context, string, string, bool) error { return nil }

	done := make(chan struct{})
	var healthy bool
	var runErr error
	go func() {
		healthy, _, runErr = svc.RunForBisect(context.Background(), "stardew", p.ID)
		close(done)
	}()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		st, err := svc.Status("stardew")
		if err == nil && (st.State == Running || st.State == Launching) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(startup, "2099-01-01T00-00-00.000Z.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("bisect did not finish after the title-screen report")
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
	if !healthy {
		t.Fatal("title-screen report must count the step healthy")
	}
}
