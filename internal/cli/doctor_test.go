package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/doctor"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/problems"
)

func TestDoctorExitsOneOnFindings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// No nxm handler is a warning, so the report has a finding.
	r := invoke(t, map[string]any{"doctor": control.Doctor{DataDir: "/data/mortar"}}, "doctor")
	if r.code != 1 || !strings.Contains(r.out, "nxm:// links: off") {
		t.Fatalf("live doctor with a finding: code %d out %q", r.code, r.out)
	}
}

func TestOfflineDoctorExitsOnFindings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	// Ask datadir for the folder: the name and location differ per OS (LOCALAPPDATA/Mortar on Windows), and only it
	// knows where `mortar doctor` looks.
	datadirtest.Use(t, t.TempDir())
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"settings.json", "store/index.json"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, p), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	offline := func(method string, _ control.Params, _ any, _ time.Duration) error {
		if method == "doctor" {
			return controlwire.ErrNotRunning
		}
		return nil
	}
	doctorRun := func() (int, string) {
		var o, e bytes.Buffer
		code := run("9.9.9", offline, []string{"doctor"}, &o, &e)
		return code, o.String()
	}
	// The app is not running, so doctor scans the folder itself: a healthy one exits 0.
	if code, out := doctorRun(); code != 0 || !strings.Contains(out, "Offline doctor: 0 findings.") {
		t.Fatalf("healthy offline doctor: code %d out %q", code, out)
	}
	if err := os.Remove(filepath.Join(dir, "settings.json")); err != nil {
		t.Fatal(err)
	}
	if code, out := doctorRun(); code != 1 || !strings.Contains(out, "settings.json is unreadable or missing") {
		t.Fatalf("offline doctor with a finding: code %d out %q", code, out)
	}
}

func TestDoctorPrintsSharedChecks(t *testing.T) {
	// The report also lists the browsers found on this machine: none under an empty home on Linux, whatever the
	// registry holds on Windows, so the expectation asks for the same rows.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
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
	wantReport := doctor.LiveWithNativeHosts(doctor.Live{
		Version: d.Version, CommandVersion: "9.9.9", DataDir: d.DataDir,
		Games: d.Games, Environment: d.Environment, NxmHandled: d.NxmHandled, NxmPrevious: d.NxmPrevious,
	})
	want := doctor.PlainText(wantReport)
	// Windows reads installed browsers from the registry, and an installed browser with no host manifest is a finding,
	// so there the same report exits 1; this test is about the rows, and the exit code follows the report either way.
	wantCode := 0
	if findings, _ := doctor.Findings(wantReport); len(findings) > 0 {
		wantCode = 1
	}
	r := invoke(t, map[string]any{"doctor": d}, "doctor")
	if r.code != wantCode {
		t.Fatalf("code %d want %d stderr %q", r.code, wantCode, r.errOut)
	}
	if r.out != want {
		t.Fatalf("got %q want %q", r.out, want)
	}
	if !strings.Contains(r.out, "Stardew Valley 1.6") || !strings.Contains(r.out, "nxm:// links: Mortar") {
		t.Fatalf("report %q", r.out)
	}
}
