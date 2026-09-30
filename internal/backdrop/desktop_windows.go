//go:build windows

package backdrop

import (
	"syscall"
	"unsafe"
)

const spiGetDeskWallpaper = 0x0073

var systemParametersInfo = syscall.NewLazyDLL("user32.dll").NewProc("SystemParametersInfoW")

// DesktopWallpaper returns the path of the user's desktop wallpaper, or empty when it cannot be read.
func DesktopWallpaper() string {
	var buf [syscall.MAX_PATH]uint16
	ok, _, _ := systemParametersInfo.Call(spiGetDeskWallpaper, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])), 0)
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:])
}
