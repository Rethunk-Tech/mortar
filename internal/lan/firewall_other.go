//go:build !windows

package lan

import "context"

func firewallBlocked(context.Context) bool { return false }

func fixFirewall(context.Context) error { return nil }
