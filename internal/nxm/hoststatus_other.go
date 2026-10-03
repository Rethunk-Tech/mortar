//go:build !linux && !windows

package nxm

// NativeHostStatus is empty where browsers do not use these host manifests.
func (System) NativeHostStatus() []HostStatus { return nil }
