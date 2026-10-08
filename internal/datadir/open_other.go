//go:build !windows

package datadir

import "os/exec"

func setArgs(*exec.Cmd, string) {}
