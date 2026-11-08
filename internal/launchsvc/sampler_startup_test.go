//go:build !windows

package launchsvc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestWaitForSampleProcessReturnsTheFirstLoaderProcess(t *testing.T) {
	svc, p := startEnv(t)
	g := game.Find("stardew")
	sl := svc.profileSlot(g, p.ID)
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	fakeProc(t, svc, "77", "", "/g/StardewModdingAPI", "--mods-path", mods)
	got, err := svc.waitForSampleProcess(t.Context(), sl, p.ID, mods)
	if err != nil || got.PID != 77 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestWaitForSampleProcessHonoursCancel(t *testing.T) {
	svc, p := startEnv(t)
	g := game.Find("stardew")
	sl := svc.profileSlot(g, p.ID)
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := svc.waitForSampleProcess(ctx, sl, p.ID, mods); err == nil {
		t.Fatal("cancelled wait returned a process")
	}
}

func TestWaitForSampleReportReturnsANewFileThenStopsWhenTheGameIsGone(t *testing.T) {
	svc, p := startEnv(t)
	g := game.Find("stardew")
	sl := svc.profileSlot(g, p.ID)
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	startup := t.TempDir()
	const reportID = "20200101T000000Z"
	testfs.WriteAt(t, filepath.Join(startup, reportID+".json"), []byte(`{}`))
	id, ok := svc.waitForSampleReport(t.Context(), sl, p.ID, mods, startup, nil)
	if !ok || id != reportID {
		t.Fatalf("id = %q, ok = %v", id, ok)
	}
	empty := t.TempDir()
	id, ok = svc.waitForSampleReport(t.Context(), sl, p.ID, mods, empty, map[string]bool{reportID: true})
	if ok || id != "" {
		t.Fatalf("gone game: id = %q, ok = %v", id, ok)
	}
}

func TestStartSamplerStopsWhenTheContextIsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	session, file, err := startSampler(ctx, 1)
	if err == nil || session != nil {
		t.Fatalf("got %v, %v, %v", session, file, err)
	}
	if file != nil {
		_ = file.Close()
	}
}

func TestWriteStartupSamplesNeedsATrace(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "empty-*.nettrace")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if _, err := writeStartupSamples(file, t.TempDir(), "r", t.TempDir()); err == nil {
		t.Fatal("an empty trace must fail")
	}
}
