package nxm

import (
	"errors"

	"github.com/Rethunk-Tech/go-desktopenv/nativemessaging"
	"github.com/Rethunk-Tech/go-desktopenv/schemehandler"
	"github.com/Rethunk-Tech/mortar/internal/nativehost"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// New returns the handler for this system, registering exe.
func New(exe string) (*System, error) {
	h, err := schemehandler.New(schemehandler.Config{Exe: exe, Schemes: source.Schemes()})
	if err != nil {
		return nil, err
	}
	return &System{exe: exe, h: h, nh: nativehost.Host(exe)}, nil
}

func (s *System) Register() error {
	if err := s.WriteNativeHosts(); err != nil {
		return err
	}
	return s.h.Register(source.Schemes()...)
}

func (s *System) RegisterSchemes(schemes []string) error { return s.h.Register(schemes...) }

// Restore hands each claimed scheme back to its previous owner; a scheme with none has its keys deleted. A scheme another
// app owns is left alone. Every scheme is tried and the errors are joined.
func (s *System) Restore(previous map[string]string) error {
	return errors.Join(s.removeNativeHosts(), s.h.Restore(previous))
}

// Release hands the given schemes back to their previous owners.
func (s *System) Release(schemes []string, previous map[string]string) error {
	return s.h.Release(schemes, previous)
}

// RegisterLinks does nothing: the installer registers the mortar scheme and the .mortar file type.
func (s *System) RegisterLinks() error { return nil }

// Refresh does nothing: the installer fixes where Mortar lives.
func (s *System) Refresh() error { return nil }

// System is the system's registration of the source link schemes and of the browser extension's native host.
type System struct {
	exe string
	h   *schemehandler.Handler
	nh  nativemessaging.Host
}
