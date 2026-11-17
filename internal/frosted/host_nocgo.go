//go:build !linux || !cgo

package frosted

import "unsafe"

func bindNative(*Session, unsafe.Pointer) {}
