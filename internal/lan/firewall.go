package lan

import "context"

// FirewallBlocked reports whether the operating system firewall would stop nearby computers reaching Mortar.
func (s *Service) FirewallBlocked() bool {
	return firewallBlocked(context.Background())
}

// FixFirewall lets Mortar through the Windows Firewall on the Private profile; it does nothing elsewhere.
func (s *Service) FixFirewall() error {
	return fixFirewall(context.Background())
}
