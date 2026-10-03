//go:build !windows

package launch

import "os/exec"

func hideWindow(*exec.Cmd, bool) {}
