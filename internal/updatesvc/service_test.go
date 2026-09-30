package updatesvc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

type fake struct {
	cfg        *updater.Config
	rel        *updater.Release
	downloaded bool
}

func (f *fake) Init(cfg updater.Config) error { f.cfg = &cfg; return nil }
func (f *fake) Check(context.Context) (*updater.Release, error) {
	return f.rel, nil
}
func (f *fake) DownloadAndInstall(context.Context) error { f.downloaded = true; return nil }
func (f *fake) Restart(context.Context) error            { return nil }

func TestDevBuildsAndVersionsNeverCheck(t *testing.T) {
	for _, c := range []struct {
		version    string
		production bool
	}{{"1.0.0", false}, {"1.1.0-dev", true}} {
		f := &fake{}
		s := &Service{}
		err := configure(s, f, c.version, nil, c.production)
		if err != nil || s.Info().Off != "dev" || f.cfg != nil {
			t.Fatalf("%+v: off %q, configured %v, %v", c, s.Info().Off, f.cfg != nil, err)
		}
		if _, err := s.Check(context.Background()); !errors.Is(err, errOff) {
			t.Fatalf("%+v: check %v", c, err)
		}
	}
}

func TestOnlyASignedReleaseInstalls(t *testing.T) {
	f := &fake{rel: &updater.Release{Version: "1.1.0", Verification: &updater.Verification{Digest: []byte{1}}}}
	s := &Service{}
	err := configure(s, f, "1.0.0", []byte("key"), true)
	if err != nil || f.cfg == nil || f.cfg.CurrentVersion != "1.0.0" || string(f.cfg.PublicKey) != "key" {
		t.Fatalf("configured %+v, %v", f.cfg, err)
	}
	if _, err := s.Check(context.Background()); !errors.Is(err, errUnsigned) {
		t.Fatalf("unsigned: %v", err)
	}
	if err := s.Install(context.Background()); !errors.Is(err, errNone) || f.downloaded {
		t.Fatalf("install after unsigned: %v, downloaded %v", err, f.downloaded)
	}
	f.rel.Verification.Signature = []byte{2}
	if rel, err := s.Check(context.Background()); err != nil || rel.Version != "1.1.0" {
		t.Fatalf("signed: %+v, %v", rel, err)
	}
	if err := s.Install(context.Background()); err != nil || !f.downloaded {
		t.Fatalf("install: %v, downloaded %v", err, f.downloaded)
	}
}

func TestCheckKeepsAStagedRelease(t *testing.T) {
	f := &fake{rel: &updater.Release{Version: "1.1.0", Verification: &updater.Verification{Signature: []byte{2}}}}
	s := &Service{}
	if err := configure(s, f, "1.0.0", []byte("key"), true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.rel = nil
	rel, err := s.Check(context.Background())
	if err != nil || rel == nil || rel.Version != "1.1.0" || !rel.Staged {
		t.Fatalf("check after install: %+v, %v", rel, err)
	}
}

func TestRestartWaitsUntilInstallHasStaged(t *testing.T) {
	hold := make(chan struct{})
	started := make(chan struct{})
	f := &stall{
		dlHold:  hold,
		dlStart: started,
	}
	f.rel = &updater.Release{Version: "1.1.0", Verification: &updater.Verification{Signature: []byte{2}}}
	s := &Service{}
	if err := configure(s, f, "1.0.0", []byte("key"), true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- s.Install(context.Background()) }()
	<-started
	restarted := make(chan error, 1)
	go func() { restarted <- s.Restart(context.Background()) }()
	select {
	case err := <-restarted:
		t.Fatalf("Restart returned before Install finished: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if f.downloaded {
		t.Fatal("download finished before the hold was released")
	}
	close(hold)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-restarted; err != nil {
		t.Fatal(err)
	}
}

func TestCheckDoesNotHoldTheLockForTheNetwork(t *testing.T) {
	hold := make(chan struct{})
	started := make(chan struct{})
	f := &stall{
		checkHold:  hold,
		checkStart: started,
	}
	f.rel = &updater.Release{Version: "1.1.0", Verification: &updater.Verification{Signature: []byte{2}}}
	s := &Service{}
	if err := configure(s, f, "1.0.0", []byte("key"), true); err != nil {
		t.Fatal(err)
	}
	errc := make(chan error, 1)
	go func() {
		_, err := s.Check(context.Background())
		errc <- err
	}()
	<-started
	if err := s.Install(context.Background()); !errors.Is(err, errNone) {
		t.Fatalf("Install during Check: %v, want errNone", err)
	}
	close(hold)
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
}

type stall struct {
	fake
	checkHold, checkStart, dlHold, dlStart chan struct{}
}

func (s *stall) Check(ctx context.Context) (*updater.Release, error) {
	if s.checkStart != nil {
		close(s.checkStart)
		<-s.checkHold
	}
	return s.fake.Check(ctx)
}

func (s *stall) DownloadAndInstall(ctx context.Context) error {
	if s.dlStart != nil {
		close(s.dlStart)
		<-s.dlHold
	}
	return s.fake.DownloadAndInstall(ctx)
}
