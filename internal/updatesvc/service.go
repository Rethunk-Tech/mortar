// Package updatesvc checks for, stages and applies Mortar's own updates through Wails' updater.
package updatesvc

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

// ManifestURL is the signed update manifest attached to the latest GitHub release.
const ManifestURL = "https://github.com/Rethunk-AI/mortar/releases/latest/download/manifest.json"

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
	// Off is "dev" when this build never checks for updates: a development or server build, or a dev version.
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
	errOff      = errors.New("updates are off in development builds")
	errUnsigned = errors.New("the update is not signed, so Mortar will not install it")
	errNone     = errors.New("no update to install; check for updates first")
)

// Service is the updater as the window sees it.
type Service struct {
	u    Updater
	info Info

	mu      sync.Mutex
	cond    *sync.Cond
	once    sync.Once
	inCheck bool
	found   *Release
}

func (s *Service) lock() {
	s.once.Do(func() { s.cond = sync.NewCond(&s.mu) })
	s.mu.Lock()
}

// Configure points s at u, which reads ManifestURL and trusts only publicKey. It is a function rather than a method
// so the binding generator does not hand it to the window. Outside a production build, or for a dev version, u is
// left unconfigured and every call reports updates as off.
func Configure(s *Service, u Updater, version string, publicKey []byte) error {
	return configure(s, u, version, publicKey, production && !application.System.IsServer())
}

func configure(s *Service, u Updater, version string, publicKey []byte, production bool) error {
	s.u, s.info = u, Info{Version: version}
	if !production || strings.Contains(version, "dev") {
		s.info.Off = "dev"
		return nil
	}
	p, err := endpoint.New(endpoint.Config{URL: ManifestURL})
	if err != nil {
		return err
	}
	return u.Init(updater.Config{
		CurrentVersion:  version,
		Providers:       []updater.Provider{p},
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
	if s.found != nil && s.found.Staged {
		r := *s.found
		s.mu.Unlock()
		return &r, nil
	}
	s.inCheck = true
	s.mu.Unlock()
	rel, err := s.u.Check(ctx)
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
	if err != nil || rel == nil {
		s.found = nil
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
	// Held throughout: a Check in between would point the updater at whatever release it found, signed or not.
	s.lock()
	defer s.mu.Unlock()
	for s.inCheck {
		s.cond.Wait()
	}
	if s.found == nil {
		return errNone
	}
	if err := s.u.DownloadAndInstall(ctx); err != nil {
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
	defer s.mu.Unlock()
	return s.u.Restart(ctx)
}
