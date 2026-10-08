//go:build !windows

package launch

import (
	"os/exec"
	"syscall"
)

// killTree ends the process group the launch started in; startCmd made the process its leader.
func killTree(pid int) { _ = syscall.Kill(-pid, syscall.SIGTERM) }

func trackTree(int)   {}
func untrackTree(int) {}

// holdStart and releaseStart are the Windows start-suspended handshake; a process group needs none.
func holdStart(*exec.Cmd)    {}
func releaseStart(int) error { return nil }
