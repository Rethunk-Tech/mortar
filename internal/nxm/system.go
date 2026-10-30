//go:build linux || windows

package nxm

import (
	"github.com/Rethunk-Tech/go-desktopenv/nativemessaging"
	"github.com/Rethunk-Tech/go-desktopenv/schemehandler"
)

// System is the system's registration of the source link schemes and of the browser extension's native host.
type System struct {
	exe string
	// dataHome is the user's XDG data folder, for the icons and the .mortar MIME definition (Linux only).
	dataHome string
	// run runs the desktop tools Mortar starts itself and is the scheme handler's and native host's runner (Linux only).
	run Runner
	h   *schemehandler.Handler
	nh  nativemessaging.Host
}

func (s *System) Owner(scheme string) (Owner, error) {
	o, err := s.h.Owner(scheme)
	return Owner(o), err
}

// ForwardOther runs the handler recorded in previous on link.
func (s *System) ForwardOther(link, previous string) error { return s.h.Forward(link, previous) }

// WriteNativeHosts lets the Mortar browser extension start this copy of Mortar.
func (s *System) WriteNativeHosts() error { return s.nh.Install() }

func (s *System) removeNativeHosts() error { return s.nh.Uninstall() }

// NativeHostStatus lists each installed browser's Mortar native-messaging manifest state.
func (s *System) NativeHostStatus() []HostStatus {
	rows, err := s.nh.Status()
	if err != nil {
		return nil
	}
	out := make([]HostStatus, 0, len(rows))
	for _, r := range rows {
		out = append(out, HostStatus{Browser: r.Browser, ManifestPath: r.Manifest, State: string(r.State)})
	}
	return out
}
