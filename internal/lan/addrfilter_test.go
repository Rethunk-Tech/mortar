package lan

import (
	"errors"
	"net"
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

type fakeConn struct {
	net.Conn
	remote net.Addr
	closed bool
}

func (c *fakeConn) RemoteAddr() net.Addr { return c.remote }
func (c *fakeConn) Close() error         { c.closed = true; return nil }

type fakeListener struct {
	net.Listener
	conns []*fakeConn
}

func (l *fakeListener) Accept() (net.Conn, error) {
	if len(l.conns) == 0 {
		return nil, net.ErrClosed
	}
	c := l.conns[0]
	l.conns = l.conns[1:]
	return c, nil
}

func tcpConn(ip string) *fakeConn {
	return &fakeConn{remote: &net.TCPAddr{IP: net.ParseIP(ip), Port: 40000}}
}

func TestListenerClosesConnectionsFromOutsideTheLAN(t *testing.T) {
	t.Parallel()
	wan, home := tcpConn("8.8.8.8"), tcpConn("192.168.1.5")
	l := &lanListener{Listener: &fakeListener{conns: []*fakeConn{wan, home}}, allowAny: func() bool { return false }}
	got, err := l.Accept()
	if err != nil || got != net.Conn(home) {
		t.Fatalf("Accept = %v, %v; want the LAN conn", got, err)
	}
	if !wan.closed || home.closed {
		t.Fatalf("closed: wan=%v lan=%v", wan.closed, home.closed)
	}

	wan2 := tcpConn("8.8.8.8")
	l = &lanListener{Listener: &fakeListener{conns: []*fakeConn{wan2}}, allowAny: func() bool { return true }}
	if got, err := l.Accept(); err != nil || got != net.Conn(wan2) || wan2.closed {
		t.Fatalf("allowAny: %v, %v, closed=%v", got, err, wan2.closed)
	}

	unix := &fakeConn{remote: &net.UnixAddr{Name: "/tmp/x", Net: "unix"}}
	l = &lanListener{Listener: &fakeListener{conns: []*fakeConn{unix}}, allowAny: func() bool { return false }}
	if _, err := l.Accept(); !errors.Is(err, net.ErrClosed) || !unix.closed {
		t.Fatalf("unix addr: err=%v closed=%v", err, unix.closed)
	}
}
