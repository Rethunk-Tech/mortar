//go:build !windows

package lan

// FirewallBlocked reports whether the operating system blocks Mortar's LAN listener.
func (s *Service) FirewallBlocked() bool {
	return false
}

// FixFirewall does nothing on operating systems without the Windows Firewall.
func (s *Service) FixFirewall() error {
	return nil
}
