//go:build linux && cgo

package frosted

/*
#cgo !gtk3 pkg-config: gtk4 wayland-client
#cgo gtk3 pkg-config: gtk+-3.0 gdk-3.0 wayland-client
#cgo gtk3 CFLAGS: -DFROSTED_GTK3
#include "wayland.h"
*/
import "C"
import "unsafe"

func bindNative(s *Session, handle unsafe.Pointer) {
	if handle == nil {
		return
	}
	s.host = &waylandHost{win: handle}
}

type waylandHost struct {
	win unsafe.Pointer
	st  unsafe.Pointer
}

func (h *waylandHost) bind() {
	if h.st != nil || h.win == nil {
		return
	}
	h.st = C.frosted_bind(h.win)
}

func (h *waylandHost) SetBlur(width, height int) {
	h.bind()
	if h.st == nil {
		return
	}
	C.frosted_set_region(h.st, C.int(width), C.int(height))
}

func (h *waylandHost) Clear() {
	if h.st == nil {
		return
	}
	C.frosted_clear(h.st)
}
