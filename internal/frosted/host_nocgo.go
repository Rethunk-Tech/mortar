//go:build !linux || !cgo

package frosted

import "unsafe"

func nativeHost(unsafe.Pointer) Host { return noopHost{} }
