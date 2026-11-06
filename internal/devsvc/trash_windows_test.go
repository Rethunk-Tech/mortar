package devsvc

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A folder another program holds is not moved, and the error names that program.
func TestTrashNamesTheProgramHoldingTheProject(t *testing.T) {
	svc, dir := trashableProject(t)
	file := filepath.Join(dir, "Held.cs")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, script := range []string{
		"$f = [IO.File]::Open($env:HOLD, 'Open', 'Read', 'ReadWrite'); 'ready'; Start-Sleep 60",
		"[Environment]::CurrentDirectory = Split-Path $env:HOLD; 'ready'; Start-Sleep 60",
	} {
		cmd := exec.CommandContext(t.Context(), "powershell", "-NoProfile", "-NonInteractive", "-Command", "& ([scriptblock]::Create($env:HOLD_SCRIPT))")
		cmd.Env = append(os.Environ(), "HOLD="+file, "HOLD_SCRIPT="+script)
		out, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		if line, err := bufio.NewReader(out).ReadString('\n'); err != nil || strings.TrimSpace(line) != "ready" {
			t.Fatalf("holder said %q, %v", line, err)
		}
		err = svc.Trash("lethal-company", dir)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Logf("%s\n  -> %v", script, err)
		held, ok := errors.AsType[*inUseError](err)
		if !ok || !slices.Contains(held.Holders, "Windows PowerShell") || len(svc.Projects("lethal-company")) != 1 {
			t.Fatalf("trash of a held project: %v, projects %d", err, len(svc.Projects("lethal-company")))
		}
	}
}
