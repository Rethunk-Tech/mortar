package lan

import (
	"net/netip"
	"testing"
)

func TestIsLANAddr(t *testing.T) {
	t.Parallel()
	for addr, want := range map[string]bool{
		"10.1.2.3": true, "172.16.0.1": true, "172.31.255.255": true, "172.32.0.1": false,
		"192.168.1.5": true, "169.254.10.1": true, "127.0.0.1": true, "::1": true,
		"fd12:3456::1": true, "fe80::1": true, "::ffff:192.168.1.5": true,
		"100.64.0.1": false, "100.127.255.254": false, "8.8.8.8": false,
		"2001:db8::1": false, "::ffff:8.8.8.8": false,
	} {
		if got := isLANAddr(netip.MustParseAddr(addr)); got != want {
			t.Errorf("isLANAddr(%s) = %v, want %v", addr, got, want)
		}
	}
}
