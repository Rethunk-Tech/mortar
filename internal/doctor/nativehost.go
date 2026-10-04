package doctor

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/nxm"
)

const nativeHostFix = "Repair"

// AppendNativeHostChecks adds one doctor check per native-host status row.
func AppendNativeHostChecks(checks []Check, rows []nxm.HostStatus) []Check {
	for _, row := range rows {
		id := "nativeHost:" + nativeHostID(row.Browser)
		switch row.State {
		case nxm.HostOK:
			checks = append(checks, Check{
				ID: id, Status: Pass,
				Detail: fmt.Sprintf("Browser extension can reach Mortar in %s", row.Browser),
			})
		case nxm.HostMissing:
			checks = append(checks, Check{
				ID: id, Status: Warn,
				Detail: fmt.Sprintf("Browser extension cannot reach Mortar in %s (manifest missing)", row.Browser),
				Fix:    nativeHostFix,
			})
		case nxm.HostStale:
			checks = append(checks, Check{
				ID: id, Status: Warn,
				Detail: fmt.Sprintf("Browser extension cannot reach Mortar in %s (manifest points elsewhere)", row.Browser),
				Fix:    nativeHostFix,
			})
		case nxm.HostUnreadable:
			checks = append(checks, Check{
				ID: id, Status: Warn,
				Detail: fmt.Sprintf("Browser extension cannot reach Mortar in %s (manifest unreadable)", row.Browser),
				Fix:    nativeHostFix,
			})
		}
	}
	return checks
}

func nativeHostID(name string) string {
	s := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, strings.ToLower(name))
	return strings.Trim(s, "-")
}
