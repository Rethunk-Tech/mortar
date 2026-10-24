package updatesvc

import (
	"context"
	"errors"
	"io"
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

type checkOnly struct {
	name string
	rel  *updater.Release
	err  error
}

func (p checkOnly) Name() string { return p.name }
func (p checkOnly) Check(context.Context, updater.CheckRequest) (*updater.Release, error) {
	return p.rel, p.err
}

func (p checkOnly) Download(context.Context, *updater.Release, io.Writer, func(int64, int64)) error {
	return nil
}

// The updater counts any error as a failed check, so a prerelease lookup GitHub refuses must not hide a stable release.
func TestChannelOffersStableWhenThePrereleaseCheckFails(t *testing.T) {
	stable := &updater.Release{Version: "1.1.0"}
	p := &channelProvider{
		stable:      checkOnly{name: "stable", rel: stable},
		beta:        checkOnly{name: "beta", err: errors.New("github releases: HTTP 403")},
		includeBeta: func() bool { return true },
	}
	if got, err := p.Check(context.Background(), updater.CheckRequest{CurrentVersion: "1.0.0"}); err != nil || got != stable {
		t.Fatalf("got %+v, %v", got, err)
	}
}
