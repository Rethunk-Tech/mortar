//go:build !linux && !windows

package nxm

func statusForExecutable(string) []HostStatus { return nil }
