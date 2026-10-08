//go:build !windows

package lan

import "context"

func firewallBlocked(context.Context) (bool, error) { return false, nil }

func fixFirewall(context.Context) error { return nil }

// AllowFirewall is the Windows self-elevated firewall step; elsewhere there is nothing to allow.
func AllowFirewall() error { return nil }
