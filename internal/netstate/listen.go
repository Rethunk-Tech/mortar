package netstate

import "strings"

// LoopbackUnlessNamed is the address to listen on for a bare port ("6060" or ":6060"): this computer only. An address
// that names its host is returned as written, so reaching a service from another computer needs the host spelled out.
func LoopbackUnlessNamed(addr string) string {
	if !strings.Contains(addr, ":") {
		return "127.0.0.1:" + addr
	}
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	return addr
}
