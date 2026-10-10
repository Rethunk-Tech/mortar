package updatesvc

import (
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestTheBackgroundModCheckWaitsThePlayersInterval(t *testing.T) {
	t.Parallel()
	if got := modUpdateEvery(settings.Settings{UpdateCheckIntervalMinutes: 45}); got != 45*time.Minute {
		t.Fatalf("interval = %v, want the setting's 45 minutes", got)
	}
	want := time.Duration(settings.DefaultUpdateCheckIntervalMinutes) * time.Minute
	for _, bad := range []int{0, settings.MinUpdateCheckIntervalMinutes - 1, settings.MaxUpdateCheckIntervalMinutes + 1} {
		if got := modUpdateEvery(settings.Settings{UpdateCheckIntervalMinutes: bad}); got != want {
			t.Fatalf("interval for %d = %v, want the code default %v", bad, got, want)
		}
	}
}
