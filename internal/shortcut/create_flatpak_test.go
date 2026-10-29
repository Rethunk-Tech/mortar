//go:build !windows

package shortcut

import (
	"errors"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

func TestFlatpakLaunchersUseAppIDAndRefuseSteam(t *testing.T) {
	old := sandbox.Getenv
	sandbox.Getenv = func(k string) string {
		if k == "FLATPAK_ID" {
			return sandbox.AppID
		}
		return ""
	}
	t.Cleanup(func() { sandbox.Getenv = old })
	// The launcher portal only installs desktop ids that start with the app id.
	if id := launcherID(Arg("stardew", "p1")); id != "tech.rethunk.Mortar.play-stardew-p1.desktop" || !strings.HasPrefix(id, sandbox.AppID+".") {
		t.Fatalf("launcher id = %s", id)
	}
	if _, err := (&Service{}).AddToSteam("stardew", "Stardew", "p1", "Main"); !errors.Is(err, ErrFlatpakSteamShortcut) {
		t.Fatalf("AddToSteam err = %v", err)
	}
}

func TestDesktopEntryEscapesBackslashes(t *testing.T) {
	if got := desktopValue(`Mods\new`); got != `Mods\\new` {
		t.Fatalf("desktopValue = %q", got)
	}
	if got := execArgEscaper.Replace("/a%b$c\\d\ne"); got != `/a%%b\\$c\\\\d\ne` {
		t.Fatalf("exec = %q", got)
	}
}
