package lan

import (
	"context"
	"strings"
)

// FirewallBlocked reports whether the operating system firewall would stop nearby computers reaching Mortar. When
// the state cannot be read it reports true, so the caller offers the fix instead of assuming all is well.
func (s *Service) FirewallBlocked() bool {
	blocked, err := firewallBlocked(context.Background())
	return blocked || err != nil
}

// FixFirewall lets Mortar through the Windows Firewall on the Private profile; it does nothing elsewhere.
func (s *Service) FixFirewall() error {
	return fixFirewall(context.Background())
}

// AllowFirewallFlag is the internal argument of the elevated Mortar that writes the firewall rules.
const AllowFirewallFlag = "--firewall-allow"

// Windows Firewall profile bits (NET_FW_PROFILE2_*); the all-profiles rule value is every bit set.
const (
	profilePrivate int32 = 2
	profilesAll    int32 = 0x7FFFFFFF
)

// fwRule is the part of a Windows Firewall rule that decides whether Mortar is reachable.
type fwRule struct {
	App      string
	Inbound  bool
	Allow    bool
	Enabled  bool
	Profiles int32
}

func (r fwRule) forExe(exe string) bool {
	return r.Enabled && r.Inbound && strings.EqualFold(r.App, exe)
}

func (r fwRule) covers(profiles int32) bool {
	return r.Profiles == profilesAll || r.Profiles&profiles != 0
}

// rulesBlock reports whether the rules leave Mortar unreachable: no inbound allow rule covering Private, or a block
// rule for it on a network profile currently connected (current is that profile bitmask).
func rulesBlock(rules []fwRule, exe string, current int32) bool {
	allowed := false
	for _, r := range rules {
		if !r.forExe(exe) {
			continue
		}
		switch {
		case !r.Allow && r.covers(current):
			return true
		case r.Allow && r.covers(profilePrivate):
			allowed = true
		}
	}
	return !allowed
}
