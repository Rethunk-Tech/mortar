package launchsvc

import (
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

func TestRecentLaunchesOrdersByNewestRun(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	svc, p1, _, _ := runEnv(t)
	p2, err := svc.profiles.Create("stardew", "B")
	if err != nil {
		t.Fatal(err)
	}
	g := game.Find("stardew")
	mods1, err := svc.profiles.ModsDir("stardew", p1.ID)
	if err != nil {
		t.Fatal(err)
	}
	mods2, err := svc.profiles.ModsDir("stardew", p2.ID)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	svc.record(g, p1.ID, old, false)
	svc.record(g, p2.ID, time.Now().Add(-time.Minute), false)
	_ = mods1
	_ = mods2
	svc.settings = set
	got, err := svc.RecentLaunches("stardew", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ProfileID != p2.ID || got[1].ProfileID != p1.ID {
		t.Fatalf("order = %#v", got)
	}
	if got[0].Name != "B" || got[1].Name != "A" {
		t.Fatalf("names = %#v", got)
	}
}

func TestRecentLaunchesIncludesLastPlayedBeforeRuns(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	svc, p, _, _ := runEnv(t)
	svc.settings = set
	at := time.Now().Add(-time.Minute)
	if _, err := set.RecordLastPlayed("stardew", p.ID, at, ""); err != nil {
		t.Fatal(err)
	}
	got, err := svc.RecentLaunches("stardew", 3)
	if err != nil || len(got) != 1 || got[0].ProfileID != p.ID {
		t.Fatalf("got %#v, %v", got, err)
	}
}

func TestRecentLaunchesUnknownGame(t *testing.T) {
	svc := NewService(t.TempDir(), nil, nil)
	if _, err := svc.RecentLaunches("nope", 3); err == nil {
		t.Fatal("expected error")
	}
}
