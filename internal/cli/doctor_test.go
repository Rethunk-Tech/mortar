package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/doctor"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/problems"
)

func TestDoctorPrintsSharedChecks(t *testing.T) {
	d := control.Doctor{
		Version: "1.2.3",
		DataDir: "/data/mortar",
		Games: []game.GameInfo{
			{ID: "stardew", Name: "Stardew Valley", Installed: true, InstallDir: "/games/Stardew Valley", Store: "steam"},
		},
		Environment: map[string]problems.Environment{
			"stardew": {GameVersion: "1.6.15", APIVersion: "4.1.10", Platform: "Linux"},
		},
		NxmHandled: true,
	}
	want := doctor.PlainText(doctor.FromLive(doctor.Live{
		Version: d.Version, CommandVersion: "9.9.9", DataDir: d.DataDir,
		Games: d.Games, Environment: d.Environment, NxmHandled: d.NxmHandled, NxmPrevious: d.NxmPrevious,
	}))
	r := invoke(t, map[string]any{"doctor": d}, "doctor")
	if r.code != 0 {
		t.Fatalf("code %d stderr %q", r.code, r.errOut)
	}
	if r.out != want {
		t.Fatalf("got %q want %q", r.out, want)
	}
	if !strings.Contains(r.out, "Stardew Valley: installed yes") || !strings.Contains(r.out, "nxm:// links: Mortar") {
		t.Fatalf("report %q", r.out)
	}
}
