package launchsvc

import (
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestChangedSinceLastRun(t *testing.T) {
	lastRun := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	event := func(at time.Time) profile.HistoryEvent {
		return profile.HistoryEvent{At: at}
	}
	tests := []struct {
		name   string
		events []profile.HistoryEvent
		want   bool
	}{
		{name: "no events", want: false},
		{name: "older event", events: []profile.HistoryEvent{event(lastRun.Add(-time.Second))}, want: false},
		{name: "same time", events: []profile.HistoryEvent{event(lastRun)}, want: false},
		{name: "newer event", events: []profile.HistoryEvent{event(lastRun.Add(time.Second))}, want: true},
		{name: "no previous run", events: []profile.HistoryEvent{event(lastRun)}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			previous := lastRun
			if test.name == "no previous run" {
				previous = time.Time{}
			}
			if got := changedSinceLastRun(test.events, previous); got != test.want {
				t.Fatalf("changedSinceLastRun() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestBackupNeededWhenTheGameWasUpdated(t *testing.T) {
	lastRun := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	if !backupNeeded("", nil, lastRun, true) {
		t.Fatal("a changed game version must trigger a launch backup")
	}
	if backupNeeded("", nil, lastRun, false) {
		t.Fatal("an unchanged game version must not trigger a launch backup")
	}
	if backupNeeded("never", nil, lastRun, true) {
		t.Fatal("never must skip a launch backup")
	}
	if !backupNeeded("always", nil, lastRun, false) {
		t.Fatal("always must take a launch backup")
	}
}

func TestAGameWithNoSaveFolderSkipsTheBackupBeforePlay(t *testing.T) {
	if err := (&Service{}).backupChangedSaves(t.Context(), "riskofrain2", "p", "", t.TempDir()); err != nil {
		t.Fatalf("backup = %v, want nothing to do", err)
	}
}
