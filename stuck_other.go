//go:build !windows

package main

import (
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/i18n"
)

func processExe(pid int) (string, error) {
	return os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
}

// waitGone waits up to 5 seconds for an ended process to be gone; it is not this process's child, so Wait cannot.
func waitGone(p *os.Process) {
	for range 50 {
		if p.Signal(syscall.Signal(0)) != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func askEndStuck(msg string) bool {
	tool := findDialogTool()
	if tool == "" {
		return false
	}
	args := []string{"--question", "--title", "Mortar", "--text", msg, "--ok-label", i18n.T("End it"), "--cancel-label", i18n.T("Leave it")}
	if tool == "kdialog" {
		args = []string{"--title", "Mortar", "--yes-label", i18n.T("End it"), "--no-label", i18n.T("Leave it"), "--warningyesno", msg}
	}
	exit, _ := runDialog(tool, args)
	return exit == 0
}
