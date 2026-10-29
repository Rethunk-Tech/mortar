//go:build !windows

package shortcut

import (
	"errors"
	"strings"
	"testing"

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
