package updatesvc

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

func TestPreferReleaseIgnoresPrereleaseWhenBetaOff(t *testing.T) {
	stable := &updater.Release{Version: "1.0.0"}
	beta := &updater.Release{Version: "2.0.0-beta"}
	if got := preferRelease(false, stable, beta); got != stable {
		t.Fatalf("beta off: got %+v, want stable", got)
	}
}

func TestPreferReleaseTakesNewerPrereleaseWhenBetaOn(t *testing.T) {
	stable := &updater.Release{Version: "1.0.0"}
	beta := &updater.Release{Version: "2.0.0-beta"}
	if got := preferRelease(true, stable, beta); got != beta {
		t.Fatalf("beta on: got %+v, want prerelease", got)
	}
}

func TestPreferReleaseKeepsStableWhenItIsNewer(t *testing.T) {
	stable := &updater.Release{Version: "2.0.0"}
	beta := &updater.Release{Version: "2.0.0-beta"}
	if got := preferRelease(true, stable, beta); got != stable {
		t.Fatalf("got %+v, want stable", got)
	}
}
