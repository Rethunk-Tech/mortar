package nxm

import (
	"errors"

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

func (w *System) Register() error {
	if err := w.WriteNativeHosts(); err != nil {
		return err
	}
	return w.h.Register(source.Schemes()...)
}

func (w *System) RegisterSchemes(schemes []string) error { return w.h.Register(schemes...) }

// Restore hands each claimed scheme back to its previous owner; a scheme with none has its keys deleted. A scheme another
// app owns is left alone. Every scheme is tried and the errors are joined.
func (w *System) Restore(previous map[string]string) error {
	return errors.Join(w.removeNativeHosts(), w.h.Restore(previous))
}

// Release hands the given schemes back to their previous owners.
func (w *System) Release(schemes []string, previous map[string]string) error {
	return w.h.Release(schemes, previous)
}

// RegisterLinks does nothing: the installer registers the mortar scheme and the .mortar file type.
func (w *System) RegisterLinks() error { return nil }

// Refresh does nothing: the installer fixes where Mortar lives.
func (w *System) Refresh() error { return nil }
