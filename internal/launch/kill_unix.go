//go:build !windows

package launch

import "syscall"

// killTree ends the process group the launch started in; startCmd made the process its leader.
func killTree(pid int) { _ = syscall.Kill(-pid, syscall.SIGTERM) }
