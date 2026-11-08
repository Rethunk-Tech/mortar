package lan

import (
	"context"

	"github.com/Rethunk-Tech/go-desktopenv/firewall"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
	return s.networkPublic()
}

// publicNetwork reports Public only when the profile bits were read and name Public without Domain or Private. A
// failed read is not Public: Mortar withholds sharing only when Windows has actually classified the network.
func publicNetwork(current int32, err error) bool {
	return err == nil && firewall.OnlyPublic(current)
}

// errPublicNetwork is what LAN use and launch return instead of listening on a Public network.
func errPublicNetwork() error {
	return usererr.New(usererr.Permission, "This network is set to Public in Windows, so Mortar doesn't share on it. Set it to Private in Windows network settings to use LAN sharing.")
}

// publicBlocksListening reports whether listening is withheld because the network is Public.
func (s *Service) publicBlocksListening(anyAddr bool) bool {
	return !anyAddr && s.networkPublic()
}

// AllowFirewallFlag is the internal argument of the elevated Mortar that writes the firewall rules; AllowFirewallAny
// is the same with the allow-any-address setting on.
const (
	AllowFirewallFlag = "--firewall-allow"
	AllowFirewallAny  = "--firewall-allow=any"
)
