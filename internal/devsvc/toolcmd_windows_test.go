//go:build windows

package devsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunStartsACommandFileInAFolderWithASpace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Ann Lee")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(dir, "tool.cmd")
	if err := os.WriteFile(tool, []byte("@echo [%1][%2]\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Service{}
	out, err := s.run(t.Context(), dir, tool, "-t", `C:\a & b\G.dll`)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != `["-t"]["C:\a & b\G.dll"]` {
		t.Fatalf("output %q", got)
	}
}
