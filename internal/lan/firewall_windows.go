//go:build windows

package lan

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/nowindow"
)

// readyScript prints "ready" when an enabled inbound allow rule for the program covers the Private profile and no
// enabled inbound block rule for it applies to a network category currently connected. Rules are matched by program
// path and read as enums, so the result does not depend on the Windows display language.
const readyBody = `
$ErrorActionPreference = 'Stop'
$mask = 0
foreach ($c in Get-NetConnectionProfile) { $mask = $mask -bor @{ 2 = 1; 0 = 4; 1 = 2 }[[int]$c.NetworkCategory] }
if ($mask -eq 0) { $mask = 7 }
$rules = @(Get-NetFirewallApplicationFilter | Where-Object { $_.Program -ieq $exe } | Get-NetFirewallRule |
  Where-Object { $_.Enabled -eq 'True' -and $_.Direction -eq 'Inbound' })
$allow = @($rules | Where-Object { $_.Action -eq 'Allow' -and ([int]$_.Profile -eq 0 -or ([int]$_.Profile -band 2)) })
$block = @($rules | Where-Object { $_.Action -eq 'Block' -and ([int]$_.Profile -eq 0 -or ([int]$_.Profile -band $mask)) })
if ($allow.Count -gt 0 -and $block.Count -eq 0) { 'ready' }
`

// fixScript removes every inbound block rule for the program (Windows names the ones it auto-creates after the file
// description, not "Mortar") and adds one allow rule scoped to the Private profile.
const fixBody = `
$ErrorActionPreference = 'Stop'
Get-NetFirewallApplicationFilter | Where-Object { $_.Program -ieq $exe } | Get-NetFirewallRule |
  Where-Object { $_.Direction -eq 'Inbound' -and $_.Action -eq 'Block' } | Remove-NetFirewallRule
Remove-NetFirewallRule -DisplayName 'Mortar' -ErrorAction SilentlyContinue
New-NetFirewallRule -DisplayName 'Mortar' -Direction Inbound -Action Allow -Program $exe -Profile Private -Enabled True | Out-Null
`

// firewallBlocked reports whether Windows Firewall would stop nearby computers reaching Mortar: no Private allow
// rule for it, or a block rule that applies.
func firewallBlocked(ctx context.Context) bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "-")
	cmd.Stdin = strings.NewReader(runEncoded(scriptWithExe(readyBody, executable)))
	nowindow.Set(cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(out.String()) != "ready"
}

// fixFirewall asks for elevation once and installs the Private-only allow rule, replacing any block rules.
func fixFirewall(ctx context.Context) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find Mortar executable: %w", err)
	}
	outer := "$ErrorActionPreference = 'Stop'; " +
		"$p = Start-Process -FilePath 'powershell.exe' -Verb RunAs -WindowStyle Hidden -Wait -PassThru " +
		"-ArgumentList '-NoProfile','-NonInteractive','-EncodedCommand','" + encodeCommand(scriptWithExe(fixBody, executable)) + "'; " +
		"exit $p.ExitCode"
	command := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "-")
	command.Stdin = strings.NewReader(outer)
	nowindow.Set(command)
	if err := command.Run(); err != nil {
		return fmt.Errorf("update Windows Firewall rule: %w", err)
	}
	return nil
}

// runEncoded is an ASCII one-liner that runs script, so a path with non-ASCII characters survives PowerShell 5.1's
// stdin decoding.
func runEncoded(script string) string {
	return "& ([ScriptBlock]::Create([Text.Encoding]::Unicode.GetString([Convert]::FromBase64String('" + encodeCommand(script) + "'))))"
}
