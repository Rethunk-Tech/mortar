package lan

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"strings"
	"unicode/utf16"
)

// FirewallBlocked reports whether the operating system firewall would stop nearby computers reaching Mortar.
func (s *Service) FirewallBlocked() bool {
	return firewallBlocked(context.Background())
}

// FixFirewall lets Mortar through the Windows Firewall on the Private profile; it does nothing elsewhere.
func (s *Service) FixFirewall() error {
	return fixFirewall(context.Background())
}

// scriptWithExe prefixes body with $exe set to the program path as a single-quoted PowerShell literal. The elevated
// step starts from a fresh environment, so the path cannot travel as an environment variable.
func scriptWithExe(body, exe string) string {
	return "$exe = '" + strings.ReplaceAll(exe, "'", "''") + "'\n" + body
}

// encodeCommand is script as PowerShell's -EncodedCommand value: base64 of UTF-16LE, which carries any user name.
func encodeCommand(script string) string {
	units := utf16.Encode([]rune(script))
	raw := make([]byte, 0, len(units)*2)
	for _, u := range units {
		raw = binary.LittleEndian.AppendUint16(raw, u)
	}
	return base64.StdEncoding.EncodeToString(raw)
}
