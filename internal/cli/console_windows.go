package cli

import (
	"os"

	"golang.org/x/sys/windows"
)

// attachParentProcess is ATTACH_PARENT_PROCESS ((DWORD)-1).
const attachParentProcess = ^uintptr(0)

var procAttachConsole = windows.NewLazySystemDLL("kernel32.dll").NewProc("AttachConsole")

// attachConsole gives the GUI-subsystem exe its parent console, so output reaches the terminal it ran in. Streams
// the caller redirected (a pipe or file) are kept.
func attachConsole() {
	if r, _, _ := procAttachConsole.Call(attachParentProcess); r == 0 {
		return
	}
	reopen := func(f **os.File, std uint32) {
		if h, err := windows.GetStdHandle(std); err == nil && h != 0 && h != windows.InvalidHandle {
			if t, err := windows.GetFileType(h); err == nil && t != windows.FILE_TYPE_UNKNOWN {
				return
			}
		}
		if con, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
			*f = con
		}
	}
	reopen(&os.Stdout, windows.STD_OUTPUT_HANDLE)
	reopen(&os.Stderr, windows.STD_ERROR_HANDLE)
}
