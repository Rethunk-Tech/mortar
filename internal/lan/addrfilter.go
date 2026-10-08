package lan

import (
	"net"
	"net/netip"
)

// isLANAddr reports whether addr is on the local network: private, link-local or loopback. Carrier-grade NAT
// (100.64.0.0/10) is deliberately not local, as it is shared with strangers on an ISP's network.
func isLANAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	return addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast()
}

// lanListener closes connections from outside the local network at accept time, before any request is read.
type lanListener struct {
	net.Listener
	allowAny func() bool
}

func (l *lanListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if l.allowAny() || remoteIsLAN(conn.RemoteAddr()) {
			return conn, nil
		}
		_ = conn.Close()
	}
}

func remoteIsLAN(addr net.Addr) bool {
	tcp, ok := addr.(*net.TCPAddr)
	if !ok {
		return false
	}
	ip, ok := netip.AddrFromSlice(tcp.IP)
	return ok && isLANAddr(ip)
}
