package launchsvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestLastPlayedRecordsOnRunningOnly(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), set, nil)
	svc.set(Status{Game: "stardew", State: Launching, Profile: "p1"})
	if len(set.Get().LastPlayed) != 0 {
		t.Fatalf("Launching recorded %#v", set.Get().LastPlayed)
	}
	svc.set(Status{Game: "stardew", State: Failed, Profile: "p1", Error: "no"})
	if len(set.Get().LastPlayed) != 0 {
		t.Fatalf("Failed recorded %#v", set.Get().LastPlayed)
	}
	svc.set(Status{Game: "stardew", State: Running, Profile: "p1"})
	got := set.Get().LastPlayed["stardew"]
	if got.Profile != "p1" || got.At == "" {
		t.Fatalf("Running lastPlayed = %#v", got)
	}
}
