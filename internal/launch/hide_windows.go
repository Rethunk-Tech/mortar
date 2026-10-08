//go:build windows

package launch

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// hideWindow starts a console program without a console. It must not also set HideWindow (STARTF_USESHOWWINDOW with
// SW_HIDE): Windows applies that to the first window the process shows, and SMAPI runs the game in its own process, so
// the game window would open hidden. CREATE_NO_WINDOW only withholds the console, which Windows Terminal would
// otherwise show however it was asked. SMAPI's output still reaches Mortar through the captured file and its log.
func hideWindow(cmd *exec.Cmd, hide bool) {
	if !hide {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
}

// ownsConsole reports a start whose console window is shown: the program then reads and writes that console itself,
// as it does when Vortex or Steam starts it. Redirecting a program that has a console breaks the console calls it makes
// (colours, cursor, window size), and SMAPI died mid-startup that way; Mortar follows SMAPI's log file instead.
func ownsConsole(hide bool) bool { return !hide }
