package settings

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/overlay"
)

const (
	MinOverlayPort     = 1024
	MaxOverlayPort     = 65535
	DefaultOverlayPort = 8123
	overlayTokenBytes  = 32
)

// NewOverlayToken returns 32 random bytes as hex. Never log the result.
func NewOverlayToken() (string, error) {
	var b [overlayTokenBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func normalizeOverlay(s *Settings) {
	if s.OverlayPort < MinOverlayPort || s.OverlayPort > MaxOverlayPort {
		s.OverlayPort = DefaultOverlayPort
	}
}

func validateOverlay(s Settings) error {
	if s.OverlayPort < MinOverlayPort || s.OverlayPort > MaxOverlayPort {
		return fmt.Errorf("overlay port must be %d to %d, got %d", MinOverlayPort, MaxOverlayPort, s.OverlayPort)
	}
	return nil
}

func writeOverlayPage() error {
	dir, err := datadir.Dir()
	if err != nil {
		return err
	}
	return overlay.WritePage(dir)
}

func overlayFileURL(s Settings) (string, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return overlay.FileURL(overlay.PagePath(dir), s.OverlayPort, s.OverlayToken), nil
}

func (s *Service) SetOverlayEnabled(on bool) error {
	token := s.store.Get().OverlayToken
	if on && token == "" {
		next, err := NewOverlayToken()
		if err != nil {
			return err
		}
		token = next
	}
	if err := s.set(func(v *Settings) {
		v.OverlayEnabled = on
		if on {
			v.OverlayToken = token
		}
	}); err != nil {
		return err
	}
	if !on {
		return nil
	}
	return writeOverlayPage()
}

func (s *Service) SetOverlayPort(n int) error {
	return s.set(func(v *Settings) { v.OverlayPort = n })
}

func (s *Service) RegenerateOverlayToken() error {
	token, err := NewOverlayToken()
	if err != nil {
		return err
	}
	if err := s.set(func(v *Settings) { v.OverlayToken = token }); err != nil {
		return err
	}
	if s.store.Get().OverlayEnabled {
		return writeOverlayPage()
	}
	return nil
}

func (s *Service) OverlayURL() (string, error) {
	cur := s.store.Get()
	if cur.OverlayEnabled {
		if err := writeOverlayPage(); err != nil {
			return "", err
		}
	}
	return overlayFileURL(cur)
}
