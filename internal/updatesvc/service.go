// Package updatesvc checks for, stages and applies Mortar's own updates through Wails' updater.
package updatesvc

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

// ManifestURL is the signed update manifest attached to the latest GitHub release.
const ManifestURL = "https://github.com/Rethunk-AI/mortar/releases/latest/download/manifest.json"

// linuxAppImage is the GitHub release asset the signed manifest lists for this GOARCH.
func linuxAppImage(goarch string) string {
	if goarch == "arm64" {
		return "mortar-linux-aarch64.AppImage"
	}
	return "mortar-linux-x86_64.AppImage"
}

// Updater is the part of app.Updater the service drives.
type Updater interface {
	Init(cfg updater.Config) error
	Check(ctx context.Context) (*updater.Release, error)
	DownloadAndInstall(ctx context.Context) error
	Restart(ctx context.Context) error
}

// Info is what Settings shows before any check.
type Info struct {
	Version string `json:"version"`
	// Off is "dev" when this build never checks for updates (development, server, or a dev version)
	// and "packaged" when a distro package owns updates.
	Off string `json:"off"`
}

// Release is an update found by Check.
type Release struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
	// Staged is set once Install has downloaded the release; only Restart is left.
	Staged bool `json:"staged"`
}

var (
	errOff         = errors.New("updates are off in development builds")
	errUnsigned    = errors.New("the update is not signed, so Mortar will not install it")
	errNone        = errors.New("no update to install; check for updates first")
	errUnreachable = errors.New("unreachable")
	errNoRelease   = errors.New("none")
	errCheckFailed = errors.New("failed")
)

// WhatsNew is shown once after Mortar starts on a newer version than last run.
type WhatsNew struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
}

// Service is the updater as the window sees it.
type Service struct {
	u    Updater
	info Info
	dir  string
	// empty runs when Check finds no newer release, so a missing manifest is not reported as up to date.
	empty func(context.Context) error

	mu            sync.Mutex
	cond          *sync.Cond
	once          sync.Once
	inCheck       bool
	inInstall     bool
	found         *Release
	restartChosen bool
}

func (s *Service) lock() {
	s.once.Do(func() { s.cond = sync.NewCond(&s.mu) })
	s.mu.Lock()
}

// Configure points s at u, which reads ManifestURL and trusts only publicKey. It is a function rather than a method
// so the binding generator does not hand it to the window. Outside a production build, for a dev version, or when
// packaged is set (nfpm, Flatpak, AUR), u is left unconfigured and every call reports updates as off.
func Configure(s *Service, u Updater, version string, publicKey []byte, packaged string, includeBeta func() bool, dataDir string) error {
	s.dir = dataDir
	if err := configure(s, u, version, publicKey, production && !application.System.IsServer(), packaged, includeBeta); err != nil {
		return err
	}
	if s.info.Off == "" {
		s.empty = checkPublished
	}
	return nil
}

func configure(s *Service, u Updater, version string, publicKey []byte, production bool, packaged string, includeBeta func() bool) error {
	s.u, s.info = u, Info{Version: version}
	if packaged != "" {
		s.info.Off = "packaged"
		return nil
	}
	if !production || strings.Contains(version, "dev") {
		s.info.Off = "dev"
		return nil
	}
	stable, err := endpoint.New(endpoint.Config{URL: ManifestURL})
	if err != nil {
		return err
	}
	chain := updater.Provider(stable)
	if includeBeta != nil {
		chain = &channelProvider{
			stable:      stable,
			beta:        newPrereleaseProvider(),
			includeBeta: includeBeta,
		}
	}
	return u.Init(updater.Config{
		CurrentVersion:  version,
		Providers:       []updater.Provider{chain},
		PublicKey:       publicKey,
		OnUpdateApplied: onApplied(version),
	})
}

// Info returns the running version and whether updates are checked.
func (s *Service) Info() Info { return s.info }

// Check asks the manifest for a newer release; nil means Mortar is up to date. Once a release is staged it is
// returned as is, so a later check cannot discard it before Restart.
func (s *Service) Check(ctx context.Context) (*Release, error) {
	if s.info.Off != "" {
		return nil, errOff
	}
	s.lock()
	for s.inInstall {
		s.cond.Wait()
	}
	if s.found != nil && s.found.Staged {
		r := *s.found
		s.mu.Unlock()
		return &r, nil
	}
	s.inCheck = true
	s.mu.Unlock()
	rel, err := s.u.Check(ctx)
	var missing error
	if err == nil && rel == nil && s.empty != nil {
		missing = s.empty(ctx)
	}
	s.lock()
	defer func() {
		s.inCheck = false
		s.cond.Broadcast()
		s.mu.Unlock()
	}()
	if s.found != nil && s.found.Staged {
		r := *s.found
		return &r, nil
	}
	if err != nil {
		log.Printf("updater check: %v", err)
		s.found = nil
		return nil, classifyCheckError(err)
	}
	if rel == nil {
		s.found = nil
		if missing != nil {
			log.Printf("updater check: %v", missing)
			return nil, classifyCheckError(missing)
		}
		return nil, err
	}
	// The verifier accepts a digest alone, and a digest proves nothing about who published the release.
	if rel.Verification == nil || len(rel.Verification.Signature) == 0 {
		s.found = nil
		return nil, errUnsigned
	}
	s.found = &Release{Version: rel.Version, Notes: rel.Notes}
	r := *s.found
	return &r, nil
}

// Install downloads, verifies and stages the release Check found; Restart applies it.
func (s *Service) Install(ctx context.Context) error {
	s.lock()
	for s.inCheck {
		s.cond.Wait()
	}
	if s.found == nil {
		s.mu.Unlock()
		return errNone
	}
	s.inInstall = true
	s.mu.Unlock()
	err := s.u.DownloadAndInstall(ctx)
	s.lock()
	defer func() {
		s.inInstall = false
		s.cond.Broadcast()
		s.mu.Unlock()
	}()
	if err != nil {
		return err
	}
	s.found.Staged = true
	return nil
}

// Restart quits Mortar, replaces it with the staged update and starts the new version.
func (s *Service) Restart(ctx context.Context) error {
	if s.info.Off != "" {
		return errOff
	}
	s.lock()
	for s.inInstall || s.inCheck {
		s.cond.Wait()
	}
	s.restartChosen = true
	s.mu.Unlock()
	return s.u.Restart(ctx)
}

// WhatsNew returns release notes when this build is newer than the last one Mortar ran; offline returns empty.
func (s *Service) WhatsNew(ctx context.Context) (WhatsNew, error) {
	var none WhatsNew
	if s.info.Off != "" || s.dir == "" {
		return none, nil
	}
	cur := s.info.Version
	last, err := readLastRun(s.dir)
	if err != nil {
		return none, err
	}
	if !upgraded(cur, last) {
		if last != cur {
			_ = writeLastRun(s.dir, cur)
		}
		return none, nil
	}
	notes, noteErr := mortarReleaseNotes(ctx, cur)
	if noteErr != nil {
		return none, noteErr
	}
	return WhatsNew{Version: cur, Notes: notes}, nil
}

// AckWhatsNew records that the user saw this version's notes.
func (s *Service) AckWhatsNew() error {
	if s.dir == "" {
		return nil
	}
	return writeLastRun(s.dir, s.info.Version)
}

func classifyCheckError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errUnreachable) || errors.Is(err, errNoRelease) || errors.Is(err, errCheckFailed) {
		return err
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return errUnreachable
	}
	if _, ok := errors.AsType[*net.DNSError](err); ok {
		return errUnreachable
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "404"), strings.Contains(msg, "not found"):
		return errNoRelease
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "timed out"), strings.Contains(msg, "i/o timeout"),
		strings.Contains(msg, "deadline exceeded"), strings.Contains(msg, "wsarecv"), strings.Contains(msg, "wsasend"),
		strings.Contains(msg, "connection refused"), strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "no such host"), strings.Contains(msg, "network is unreachable"),
		strings.Contains(msg, "temporary failure in name resolution"), strings.Contains(msg, "dial tcp"):
		return errUnreachable
	default:
		return errCheckFailed
	}
}

func checkPublished(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ManifestURL, nil)
	if err != nil {
		return nil
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return errNoRelease
	}
	return nil
}
