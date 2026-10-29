//go:build !windows

package shortcut

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-desktopenv/portal"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
)

func TestFlatpakLaunchersUseAppIDAndRefuseSteam(t *testing.T) {
	t.Setenv("FLATPAK_ID", appversion.AppID)
	// The launcher portal only installs desktop ids that start with the app id.
	if id := launcherID(Arg("stardew", "p1")); id != "tech.rethunk.Mortar.play-stardew-p1.desktop" || !strings.HasPrefix(id, appversion.AppID+".") {
		t.Fatalf("launcher id = %s", id)
	}
	if _, err := (&Service{}).AddToSteam("stardew", "Stardew", "p1", "Main"); !errors.Is(err, ErrFlatpakSteamShortcut) {
		t.Fatalf("AddToSteam err = %v", err)
	}
}

func TestFlatpakRemoveGivesADismissedDialogMortarsWording(t *testing.T) {
	t.Setenv("FLATPAK_ID", appversion.AppID)
	old := uninstallLauncher
	t.Cleanup(func() { uninstallLauncher = old })
	var got string
	uninstallLauncher = func(id string) error { got = id; return portal.ErrCancelled }
	if err := Removed("stardew", "p1"); err == nil || err.Error() != "cancelled in the system dialog" {
		t.Fatalf("Removed err = %v", err)
	}
	if got != launcherID(Arg("stardew", "p1")) {
		t.Fatalf("uninstalled %q", got)
	}
}

func TestFlatpakRenameReinstallsOnlyAnExistingLauncher(t *testing.T) {
	t.Setenv("FLATPAK_ID", appversion.AppID)
	oldExists, oldInstall := launcherExists, installLauncher
	t.Cleanup(func() { launcherExists, installLauncher = oldExists, oldInstall })
	exists := false
	var got []string
	launcherExists = func(string) (bool, error) { return exists, nil }
	installLauncher = func(id, name, arg string) error { got = []string{id, name, arg}; return nil }
	if err := Renamed("stardew", "p1", "Main", "Stardew"); err != nil || got != nil {
		t.Fatalf("missing launcher: installed %v, %v", got, err)
	}
	exists = true
	if err := Renamed("stardew", "p1", "Main", "Stardew"); err != nil {
		t.Fatal(err)
	}
	if want := []string{launcherID(Arg("stardew", "p1")), "Main (Stardew)", Arg("stardew", "p1")}; !slices.Equal(got, want) {
		t.Fatalf("installed %v, want %v", got, want)
	}
}
