package doctor

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/problems"
)

func TestFromLiveMatchesCLIPrint(t *testing.T) {
	in := Live{
		Version:        "1.2.3",
		CommandVersion: "9.9.9",
		DataDir:        "/data/mortar",
		Games: []game.GameInfo{
			{ID: "stardew", Name: "Stardew Valley", Installed: true, InstallDir: "/games/Stardew Valley", Store: "steam"},
			{ID: "lethal", Name: "Lethal Company"},
		},
		Environment: map[string]problems.Environment{
			"stardew": {GameVersion: "1.6.15", APIVersion: "4.1.10", Platform: "Linux"},
			"lethal":  {Platform: "Linux"},
		},
		NxmHandled:  true,
		NxmPrevious: "Vortex",
	}
	got := PlainText(FromLive(in))
	want := "Mortar 1.2.3 (this command 9.9.9)\n" +
		"Data folder: /data/mortar\n" +
		"Stardew Valley 1.6.15 with SMAPI 4.1.10 in \"/games/Stardew Valley\" (steam, Linux)\n" +
		"Lethal Company: not installed\n" +
		"nxm:// links: Mortar (other games go to Vortex)\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	r := FromLive(in)
	if r.Checks[0].Status != Pass || r.Checks[2].Status != Pass || r.Checks[3].Status != Pass || r.Checks[4].Status != Pass {
		t.Fatalf("statuses: %+v", r.Checks)
	}
}

func TestScanFlagsCorruptSettingsAndPassesAHealthyFolder(t *testing.T) {
	bad := t.TempDir()
	if err := os.WriteFile(filepath.Join(bad, "settings.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := Scan(bad)
	findings, _ := Findings(r)
	if !slices.Contains(findings, "settings.json is corrupt") {
		t.Fatalf("corrupt: %+v", findings)
	}
	if !slices.Contains(findings, "store index.json is unreadable or corrupt") {
		t.Fatalf("store: %+v", findings)
	}

	ok := t.TempDir()
	if err := os.WriteFile(filepath.Join(ok, "settings.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ok, "store"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ok, "store", "index.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	prof := filepath.Join(ok, "profiles", "stardew", "p")
	if err := os.MkdirAll(prof, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prof, "profile.json"), []byte(`{"id":"p"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ctrl := filepath.Join(ok, "control.json")
	if err := os.WriteFile(ctrl, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(ctrl, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}
	r = Scan(ok)
	findings, _ = Findings(r)
	if len(findings) != 0 {
		t.Fatalf("healthy findings: %s", strings.Join(findings, "; "))
	}
}

func TestScanStaleControlAndDamagedProfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "store"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "store", "index.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	prof := filepath.Join(dir, "profiles", "stardew", "p")
	if err := os.MkdirAll(prof, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prof, "profile.json"), []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctrl := filepath.Join(dir, "control.json")
	if err := os.WriteFile(ctrl, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(ctrl, old, old); err != nil {
		t.Fatal(err)
	}
	findings, _ := Findings(Scan(dir))
	if !slices.Contains(findings, "control.json is stale") {
		t.Fatalf("stale: %+v", findings)
	}
	wantProf := "damaged profile.json: " + filepath.Join(prof, "profile.json")
	if !slices.Contains(findings, wantProf) {
		t.Fatalf("profile: %+v", findings)
	}
}
