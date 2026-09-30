package nexus

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed client.go
var clientSrc string

//go:embed seen.go
var seenSrc string

//go:embed account.go
var accountSrc string

//go:embed mod.go
var modSrc string

func TestSourceDoesNotInvokeCallbacksUnderMutex(t *testing.T) {
	for path, src := range map[string]string{
		"client.go":  clientSrc,
		"seen.go":    seenSrc,
		"account.go": accountSrc,
		"mod.go":     modSrc,
	} {
		depth := 0
		for i, line := range strings.Split(src, "\n") {
			trim := strings.TrimSpace(line)
			if strings.Contains(trim, ".Lock()") && !strings.Contains(trim, "RLock") {
				depth++
			}
			if strings.Contains(trim, ".Unlock()") && depth > 0 {
				depth--
			}
			if depth == 0 {
				continue
			}
			if strings.Contains(trim, "SetLimitsHook") {
				continue
			}
			if strings.Contains(trim, "onLimits(") || strings.Contains(trim, "Emit(") || strings.Contains(trim, "Hook(") {
				t.Fatalf("%s:%d invokes a callback while a mutex is held: %s", path, i+1, trim)
			}
		}
	}
}
