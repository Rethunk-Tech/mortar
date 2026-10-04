package launchsvc

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/backup"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
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

func TestLaunchBackupLocationFollowsGameSetting(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	custom := filepath.Join(t.TempDir(), "launch-backups")
	var set settings.Settings
	if err := settings.ApplyKeyGame(&set, "backupLocation", custom, "stardew"); err != nil {
		t.Fatal(err)
	}
	write, _, err := backup.Locations(settings.Resolve(set, "backupLocation", "stardew", nil))
	if err != nil {
		t.Fatal(err)
	}
	if write != custom {
		t.Fatalf("write dir = %q, want %q", write, custom)
	}
}

func TestBackupNeededWhenGameVersionChanged(t *testing.T) {
	lastRun := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	if !backupNeeded("", nil, lastRun, "1.6.14", "1.6.15") {
		t.Fatal("a changed game version must trigger a launch backup")
	}
	if backupNeeded("", nil, lastRun, "1.6.15", "1.6.15") {
		t.Fatal("an unchanged game version must not trigger a launch backup")
	}
	if backupNeeded("never", nil, lastRun, "1.6.14", "1.6.15") {
		t.Fatal("never must skip a launch backup")
	}
	if !backupNeeded("always", nil, lastRun, "1.6.15", "1.6.15") {
		t.Fatal("always must take a launch backup")
	}
}
