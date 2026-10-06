//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestLastRunIssuesNameTheBepInExPackageBehindAPlugin(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ps, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	const lc = "lethal-company"
	p, err := ps.Create(lc, "A")
	if err != nil {
		t.Fatal(err)
	}
	dll, err := os.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	zip := tsPackage(t, "Fixture", map[string]string{"plugins/Mod.dll": string(dll)})
	if _, err := ps.InstallSource(lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Fixture", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	dir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "BepInEx"), 0o700); err != nil {
		t.Fatal(err)
	}
	const body = "[Info   :   BepInEx] BepInEx 5.4.22 - Lethal Company\n" +
		"[Error  :Fixture Plugin] boom\n" +
		"[Warning:com.fixture.plugin] careful\n" +
		"[Error  :Someone Else] not ours\n"
	if err := os.WriteFile(filepath.Join(dir, "BepInEx", "LogOutput.log"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), nil, ps)
	svc.record(game.Find(lc), p.ID, time.Now(), false)

	got, err := svc.LastRunIssues(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Mods) != 1 || got.Mods[0].Name != "Fixture" || got.Mods[0].Errors != 1 || got.Mods[0].Warnings != 1 {
		t.Fatalf("issues = %+v", got.Mods)
	}
}
