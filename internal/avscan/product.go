package avscan

import (
	"context"
	"fmt"
	"os"
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

// Status is what Settings and `mortar antivirus status` report: which scanner the settings resolve to, the product
// behind it, and whether a scan can run now.
type Status struct {
	Mode    string `json:"mode"`
	Scanner string `json:"scanner"`
	Product string `json:"product"`
	Ready   bool   `json:"ready"`
	Problem string `json:"problem,omitempty"`
}

// StatusOf resolves c. Ready comes from scanning an empty folder, which opens the scanner (an AMSI session, a clamd
// connection, the command's program) without any file to flag.
func StatusOf(ctx context.Context, c Config) Status {
	st := Status{Mode: c.Mode, Scanner: kindOf(c), Product: Product(ctx, c)}
	if st.Mode == "" {
		st.Mode = ModeAutomatic
	}
	dir, err := os.MkdirTemp("", "mortar-av-status-")
	if err != nil {
		st.Problem = err.Error()
		return st
	}
	defer func() { _ = os.RemoveAll(dir) }()
	ctx, cancel := context.WithTimeout(ctx, productTimeout)
	defer cancel()
	_, _, err = New(c).Scan(ctx, dir)
	if err == nil && st.Scanner == "clamd" {
		// An empty folder never reaches the daemon, so ask it directly.
		_, err = clamd{socket: c.Socket}.Version(ctx)
	}
	if err != nil {
		st.Problem = err.Error()
		return st
	}
	st.Ready = true
	return st
}

func kindOf(c Config) string {
	switch c.Mode {
	case ModeOff:
		return "off"
	case ModeClamd:
		return "clamd"
	case ModeCommand:
		return "command"
	}
	if runtime.GOOS == "windows" {
		return "amsi"
	}
	return "clamd"
}
