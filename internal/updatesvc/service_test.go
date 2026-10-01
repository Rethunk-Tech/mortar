package updatesvc

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

type fake struct {
	cfg           *updater.Config
	rel           *updater.Release
	err           error
	downloaded    bool
	restarted     bool
	appliedOnExit bool
}

func (f *fake) Init(cfg updater.Config) error { f.cfg = &cfg; return nil }
func (f *fake) Check(context.Context) (*updater.Release, error) {
	return f.rel, f.err
}
func (f *fake) DownloadAndInstall(context.Context) error { f.downloaded = true; return nil }
func (f *fake) Restart(context.Context) error {
	f.restarted = true
	return nil
}

func (f *fake) ApplyOnExit(context.Context) error {
	f.appliedOnExit = true
	return nil
}

func TestDevBuildsAndVersionsNeverCheck(t *testing.T) {
	for _, c := range []struct {
		version    string
		production bool
	}{{"1.0.0", false}, {"1.1.0-dev", true}} {
		f := &fake{}
		s := &Service{}
		err := configure(s, f, c.version, nil, c.production, "", nil)
		if err != nil || s.Info().Off != "dev" || f.cfg != nil {
			t.Fatalf("%+v: off %q, configured %v, %v", c, s.Info().Off, f.cfg != nil, err)
		}
		if _, err := s.Check(context.Background()); !errors.Is(err, errOff) {
			t.Fatalf("%+v: check %v", c, err)
		}
	}
}

func TestPackagedBuildsNeverCheck(t *testing.T) {
	f := &fake{}
	s := &Service{}
	err := configure(s, f, "1.0.0", []byte("key"), true, "deb", nil)
	if err != nil || s.Info().Off != "packaged" || f.cfg != nil {
		t.Fatalf("off %q, configured %v, %v", s.Info().Off, f.cfg != nil, err)
	}
	if _, err := s.Check(context.Background()); !errors.Is(err, errOff) {
		t.Fatalf("check %v", err)
	}
}

func TestOnlyASignedReleaseInstalls(t *testing.T) {
	f := &fake{rel: &updater.Release{Version: "1.1.0", Verification: &updater.Verification{Digest: []byte{1}}}}
	s := &Service{}
	err := configure(s, f, "1.0.0", []byte("key"), true, "", nil)
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
	if err := configure(s, f, "1.0.0", []byte("key"), true, "", nil); err != nil {
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
	if err := configure(s, f, "1.0.0", []byte("key"), true, "", nil); err != nil {
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

func TestRestartDoesNotHoldTheMutex(t *testing.T) {
	s := &Service{}
	f := &reenterRestart{s: s}
	f.rel = &updater.Release{Version: "1.1.0", Verification: &updater.Verification{Signature: []byte{2}}}
	if err := configure(s, f, "1.0.0", []byte("key"), true, "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- s.Restart(context.Background()) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Restart deadlocked holding the mutex")
	}
}

type reenterRestart struct {
	fake
	s *Service
}

func (r *reenterRestart) Restart(ctx context.Context) error {
	_, err := r.s.Check(ctx)
	return err
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
	if err := configure(s, f, "1.0.0", []byte("key"), true, "", nil); err != nil {
		t.Fatal(err)
	}
	errc := make(chan error, 1)
	go func() {
		_, err := s.Check(context.Background())
		errc <- err
	}()
	<-started
	installed := make(chan error, 1)
	go func() { installed <- s.Install(context.Background()) }()
	select {
	case err := <-installed:
		t.Fatalf("Install returned during Check: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(hold)
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
	if err := <-installed; err != nil || !f.downloaded {
		t.Fatalf("install after check: %v, downloaded %v", err, f.downloaded)
	}
}

func TestInstallReleasesTheMutexDuringDownload(t *testing.T) {
	hold := make(chan struct{})
	started := make(chan struct{})
	f := &stall{dlHold: hold, dlStart: started}
	f.rel = &updater.Release{Version: "1.1.0", Verification: &updater.Verification{Signature: []byte{2}}}
	s := &Service{}
	if err := configure(s, f, "1.0.0", []byte("key"), true, "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- s.Install(context.Background()) }()
	<-started
	locked := make(chan struct{})
	go func() {
		s.lock()
		close(locked)
		s.mu.Unlock()
	}()
	select {
	case <-locked:
	case <-time.After(time.Second):
		t.Fatal("Install still holds the mutex during DownloadAndInstall")
	}
	close(hold)
	if err := <-done; err != nil {
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

func TestClassifyCheckError(t *testing.T) {
	for _, c := range []struct {
		err  error
		want error
	}{
		{errors.New("updater: all providers failed: endpoint: endpoint: fetch manifest: Get \"https://github.com/Rethunk-AI/mortar/releases/latest/download/manifest.json\": wsarecv: A connection attempt failed"), errUnreachable},
		{errors.New("Get \"https://github.com/...\": i/o timeout"), errUnreachable},
		{errors.New("endpoint: manifest request failed: HTTP 404"), errNoRelease},
		{errors.New("endpoint: decode manifest: unexpected end of JSON"), errCheckFailed},
	} {
		if got := classifyCheckError(c.err); !errors.Is(got, c.want) {
			t.Errorf("classify(%q) = %v, want %v", c.err, got, c.want)
		}
	}
}

func TestCheckNetworkErrorHidesTheGoChain(t *testing.T) {
	f := &fake{err: errors.New("updater: all providers failed: endpoint: fetch manifest: Get ...: wsarecv: connection timed out")}
	s := &Service{}
	if err := configure(s, f, "1.0.0", []byte("key"), true, "", nil); err != nil {
		t.Fatal(err)
	}
	rel, err := s.Check(context.Background())
	if rel != nil || !errors.Is(err, errUnreachable) {
		t.Fatalf("check = %+v, %v", rel, err)
	}
	if err.Error() != errUnreachable.Error() {
		t.Fatalf("leaked detail: %v", err)
	}
}

func TestCheckMissingManifestIsNotUpToDate(t *testing.T) {
	f := &fake{}
	s := &Service{}
	if err := configure(s, f, "1.0.0", []byte("key"), true, "", nil); err != nil {
		t.Fatal(err)
	}
	s.empty = func(context.Context) error { return errors.New("HTTP 404") }
	rel, err := s.Check(context.Background())
	if rel != nil || !errors.Is(err, errNoRelease) {
		t.Fatalf("missing manifest = %+v, %v", rel, err)
	}
}

func TestCheckNilReleaseIsCurrentWhenTheManifestExists(t *testing.T) {
	f := &fake{}
	s := &Service{}
	if err := configure(s, f, "1.0.0", []byte("key"), true, "", nil); err != nil {
		t.Fatal(err)
	}
	s.empty = func(context.Context) error { return nil }
	rel, err := s.Check(context.Background())
	if rel != nil || err != nil {
		t.Fatalf("current = %+v, %v", rel, err)
	}
}

func TestLinuxAppImageMatchesGOARCH(t *testing.T) {
	assets := []string{"mortar-linux-x86_64.AppImage", "mortar-linux-aarch64.AppImage", "mortar-windows-amd64.exe"}
	for goarch, want := range map[string]string{
		"amd64": "mortar-linux-x86_64.AppImage",
		"arm64": "mortar-linux-aarch64.AppImage",
	} {
		got := linuxAppImage(goarch)
		if got != want {
			t.Errorf("linuxAppImage(%q) = %q, want %q", goarch, got, want)
		}
		if !slices.Contains(assets, got) {
			t.Errorf("linuxAppImage(%q) %q is not a published asset", goarch, got)
		}
	}
}
