package avscan

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const productTimeout = 5 * time.Second

// Product names the scanner the settings resolve to, for Settings: the antivirus Windows Security Center lists, the
// clamd version found, the custom command's program, or what is missing.
func Product(ctx context.Context, c Config) string {
	ctx, cancel := context.WithTimeout(ctx, productTimeout)
	defer cancel()
	switch c.Mode {
	case ModeOff:
		return "Antivirus scanning is off"
	case ModeCommand:
		if parts := splitCommand(c.Command); len(parts) > 0 {
			return filepath.Base(parts[0])
		}
		return "No command set"
	case ModeClamd:
		return clamdProduct(ctx, clamd{socket: c.Socket})
	}
	if runtime.GOOS == "windows" {
		return windowsProduct(ctx)
	}
	return clamdProduct(ctx, clamd{socket: c.Socket})
}

func clamdProduct(ctx context.Context, c clamd) string {
	line, err := c.Version(ctx)
	if err != nil || line == "" {
		return "No antivirus found"
	}
	version, _, _ := strings.Cut(line, "/")
	return strings.Replace(version, "ClamAV", "clamd", 1)
}

// windowsProduct reads the product names from Security Center; without a readable answer AMSI still reaches whichever
// antivirus is registered, so that is what it says.
func windowsProduct(ctx context.Context) string {
	const fallback = "Windows antivirus (AMSI)"
	out, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		"(Get-CimInstance -Namespace root/SecurityCenter2 -ClassName AntiVirusProduct).displayName -join ', '").Output()
	if err != nil {
		return fallback
	}
	if name := strings.TrimSpace(string(out)); name != "" {
		return fmt.Sprintf("%s (AMSI)", name)
	}
	return fallback
}
