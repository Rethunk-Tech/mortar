//go:build !windows

package lan

import "testing"

func TestFirewallBlockedIsFalseOnNonWindows(t *testing.T) {
	service := NewService(Deps{})
	if service.FirewallBlocked() {
		t.Fatal("FirewallBlocked() = true on non-Windows")
	}
	if err := service.FixFirewall(); err != nil {
		t.Fatalf("FixFirewall() error = %v", err)
	}
}
