package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

func TestProfileShortcutAndSteam(t *testing.T) {
	r := invoke(t, map[string]any{
		"profile.shortcut": control.ShortcutResult{Path: "/tmp/x.lnk"},
	}, "profile", "shortcut", "stardew", "Farm")
	if r.code != 0 || r.calls[0].method != "profile.shortcut" || r.calls[0].params.Remove {
		t.Fatalf("create: %+v", r)
	}
	if !strings.Contains(r.out, "/tmp/x.lnk") {
		t.Fatalf("out: %q", r.out)
	}
	r = invoke(t, map[string]any{"profile.shortcut": control.ShortcutResult{Removed: true}},
		"profile", "shortcut", "stardew", "Farm", "--remove")
	if r.code != 0 || !r.calls[0].params.Remove || !strings.Contains(r.out, "Removed") {
		t.Fatalf("remove: %+v %q", r.calls[0], r.out)
	}
	r = invoke(t, map[string]any{"profile.steam": control.SteamShortcutResult{Result: steam.Added}},
		"profile", "steam", "stardew", "Farm")
	if r.code != 0 || r.calls[0].method != "profile.steam" || !strings.Contains(r.out, "Added") {
		t.Fatalf("steam: %+v %q", r, r.out)
	}
}
