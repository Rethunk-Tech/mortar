//go:build windows

package lan

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (s *Service) FirewallBlocked() bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	output, err := exec.CommandContext(context.Background(), "netsh.exe", "advfirewall", "firewall", "show", "rule", "name=all", "verbose").Output()
	if err != nil {
		return false
	}
	return blockedFirewallRule(string(output), executable)
}

func (s *Service) FixFirewall() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find Mortar executable: %w", err)
	}
	script := fmt.Sprintf(
		"$delete = @('advfirewall','firewall','delete','rule','name=Mortar'); "+
			"Start-Process -FilePath 'netsh.exe' -Verb RunAs -Wait -ArgumentList $delete; "+
			"$add = @('advfirewall','firewall','add','rule','name=Mortar','dir=in','action=allow',"+
			"'program=\"%s\"','enable=yes','profile=private,domain'); "+
			"$process = Start-Process -FilePath 'netsh.exe' -Verb RunAs -Wait -PassThru -ArgumentList $add; "+
			"if ($process.ExitCode -ne 0) { exit $process.ExitCode }",
		powerShellQuote(executable),
	)
	command := exec.CommandContext(context.Background(), "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "-")
	command.Stdin = strings.NewReader(script)
	if err := command.Run(); err != nil {
		return fmt.Errorf("update Windows Firewall rule: %w", err)
	}
	return nil
}

func blockedFirewallRule(output, executable string) bool {
	for rule := range strings.SplitSeq(strings.ReplaceAll(output, "\r\n", "\n"), "\n\n") {
		if firewallField(rule, "Enabled") != "yes" ||
			firewallField(rule, "Direction") != "in" ||
			firewallField(rule, "Action") != "block" {
			continue
		}
		if strings.EqualFold(firewallField(rule, "Program"), executable) {
			return true
		}
	}
	return false
}

func firewallField(rule, name string) string {
	for line := range strings.SplitSeq(rule, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(key), name) {
			return strings.ToLower(strings.TrimSpace(value))
		}
	}
	return ""
}

func powerShellQuote(value string) string {
	if value == "" {
		return ""
	}
	return strings.ReplaceAll(value, "'", "''")
}
