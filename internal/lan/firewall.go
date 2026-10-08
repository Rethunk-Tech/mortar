package lan

import (
	"context"
	"strings"
)

// FirewallBlocked reports whether the operating system firewall is not yet set up so that Mortar can listen without a
// system prompt. When the state cannot be read it reports true, so the caller offers the fix.
func (s *Service) FirewallBlocked() bool {
	blocked, err := firewallBlocked(context.Background(), s.allowAnyAddress())
	return blocked || err != nil
}

// FixFirewall writes Mortar's Windows Firewall rules; it does nothing elsewhere.
func (s *Service) FixFirewall() error {
	return fixFirewall(context.Background(), s.allowAnyAddress())
}

// NetworkIsPublic reports that Windows has the connected network on its Public profile, where a LAN-only Mortar is
// blocked by design: nearby computers cannot reach it until the network is set to Private.
func (s *Service) NetworkIsPublic() bool {
	return networkIsPublic()
}

// AllowFirewallFlag is the internal argument of the elevated Mortar that writes the firewall rules; AllowFirewallAny
// is the same with the allow-any-address setting on.
const (
	AllowFirewallFlag = "--firewall-allow"
	AllowFirewallAny  = "--firewall-allow=any"
)

// Windows Firewall profile bits (NET_FW_PROFILE2_*); the all-profiles rule value is every bit set.
const (
	profileDomain  int32 = 1
	profilePrivate int32 = 2
	profilePublic  int32 = 4
	profilesAll    int32 = 0x7FFFFFFF
	// profilesLocal is where a LAN-only Mortar is allowed; Public gets an explicit block so Windows never prompts.
	profilesLocal = profileDomain | profilePrivate
	profilesEvery = profilesLocal | profilePublic
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

func (r fwRule) overlaps(profiles int32) bool {
	return r.Profiles == profilesAll || r.Profiles&profiles != 0
}

func (r fwRule) includes(profiles int32) bool {
	return r.Profiles == profilesAll || r.Profiles&profiles == profiles
}

// rulesBlock reports whether the rules are not yet the set that lets Mortar listen without Windows asking: an allow
// rule on every profile it should be reachable on, no block rule on those, and, for a LAN-only Mortar, a block rule
// on Public so every profile has a matching rule.
func rulesBlock(rules []fwRule, exe string, anyAddr bool) bool {
	reach := profilesLocal
	if anyAddr {
		reach = profilesEvery
	}
	allowed, publicBlocked := false, false
	for _, r := range rules {
		if !r.forExe(exe) {
			continue
		}
		switch {
		case r.Allow && r.includes(reach):
			allowed = true
		case !r.Allow && r.overlaps(reach):
			return true
		case !r.Allow && r.includes(profilePublic):
			publicBlocked = true
		}
	}
	return !allowed || (!anyAddr && !publicBlocked)
}
