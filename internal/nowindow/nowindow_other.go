//go:build !windows

// Package nowindow starts console programs from Mortar's GUI process without a console window.
package nowindow

import "os/exec"

// Set does nothing: only Windows opens a window for a console program a GUI process starts.
func Set(*exec.Cmd) {}
