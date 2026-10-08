//go:build !windows

package lan

import "context"

func firewallBlocked(context.Context, bool) (bool, error) { return false, nil }

func fixFirewall(context.Context, bool) error { return nil }

func networkIsPublic() bool { return false }

// AllowFirewall is the Windows self-elevated firewall step; elsewhere there is nothing to allow.
func AllowFirewall(bool) error { return nil }
