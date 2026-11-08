//go:build windows

package lan

import (
	"context"
	"fmt"
	"os"

	"github.com/Rethunk-Tech/go-desktopenv/firewall"
)

const (
	ruleName       = "Mortar"
	publicRuleName = "Mortar (public networks)"
)

func firewallBlocked(_ context.Context, anyAddr bool) (bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return true, fmt.Errorf("find Mortar executable: %w", err)
	}
	rules, err := firewall.Rules()
	if err != nil {
		return true, err
	}
	return firewall.Blocked(rules, exe, anyAddr), nil
}

func networkIsPublic() bool {
	current, err := firewall.CurrentProfileTypes()
	return publicNetwork(current, err)
}

// AllowFirewall is the elevated step. It writes Mortar's inbound allow rule (and the Public-network block unless
// anyAddr) so Windows never prompts.
func AllowFirewall(anyAddr bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return firewall.Add(exe, ruleName, publicRuleName, anyAddr)
}

func fixFirewall(_ context.Context, anyAddr bool) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find Mortar executable: %w", err)
	}
	flag := AllowFirewallFlag
	if anyAddr {
		flag = AllowFirewallAny
	}
	return firewall.Elevate(exe, flag)
}
