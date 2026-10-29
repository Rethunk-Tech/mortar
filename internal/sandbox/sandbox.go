// Package sandbox holds the two host-folder lookups internal/pack still calls; it goes once pack calls
// go-desktopenv/userdirs directly.
package sandbox

import "github.com/Rethunk-Tech/go-desktopenv/userdirs"

// HostConfigHome is userdirs.HostConfigHome.
func HostConfigHome(home string) string { return userdirs.HostConfigHome(home) }

// HostDataHome is userdirs.HostDataHome.
func HostDataHome(home string) string { return userdirs.HostDataHome(home) }
