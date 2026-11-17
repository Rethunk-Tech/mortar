// Package frosted applies a compositor blur behind a translucent Linux window.
//
// GNOME 51+ advertises Wayland ext_background_effect_manager_v1. Older
// compositors, X11, and non-Linux builds keep the existing solid/translucent
// window and do not fail to start.
package frosted

import "unsafe"

// Host applies or removes the compositor blur region for one toplevel.
type Host interface {
	SetBlur(width, height int)
	Clear()
}

type noopHost struct{}

func (noopHost) SetBlur(int, int) {}
func (noopHost) Clear()           {}

// Session keeps the blur region in sync with translucency and window size.
type Session struct {
	host        Host
	translucent bool
	width       int
	height      int
	on          bool
}

// New starts with blur off. A nil host is a no-op (missing protocol, tests).
func New(host Host) *Session {
	if host == nil {
		host = noopHost{}
	}
	return &Session{host: host}
}

// Attach binds the native GTK window (or a no-op host) and applies blur when translucent.
func Attach(handle unsafe.Pointer, width, height int, translucent bool) *Session {
	s := New(nativeHost(handle))
	s.Resize(width, height)
	s.SetTranslucent(translucent)
	return s
}

// SetTranslucent turns the whole-window blur on or off. Off clears a region left from a previous on.
func (s *Session) SetTranslucent(on bool) {
	s.translucent = on
	s.sync()
}

// Resize updates the blur region to the new toplevel size while translucency is on.
func (s *Session) Resize(width, height int) {
	s.width = width
	s.height = height
	s.sync()
}

func (s *Session) sync() {
	if !s.translucent {
		if s.on {
			s.host.Clear()
			s.on = false
		}
		return
	}
	if s.width <= 0 || s.height <= 0 {
		return
	}
	s.host.SetBlur(s.width, s.height)
	s.on = true
}
