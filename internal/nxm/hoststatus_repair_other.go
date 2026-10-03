//go:build !linux && !windows

package nxm

func repairNativeHosts(string) error { return nil }
