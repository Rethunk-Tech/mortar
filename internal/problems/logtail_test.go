package problems

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func TestReadLogTailKeepsOnlyTheLastMaxLogBytes(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "Player.log")
	big := "FIRST-LINE\n" + strings.Repeat("x", launch.MaxLogBytes) + "\nLAST-LINE\n"
	if err := os.WriteFile(path, []byte(big), 0o600); err != nil {
		t.Fatal(err)
	}
	got := readLogTail(path)
	if len(got) > launch.MaxLogBytes || strings.Contains(got, "FIRST-LINE") || !strings.HasSuffix(got, "LAST-LINE\n") {
		t.Fatalf("tail of %d bytes: starts %.20q", len(got), got)
	}
	small := filepath.Join(t.TempDir(), "g.log")
	if err := os.WriteFile(small, []byte("a\nb\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readLogTail(small); got != "a\nb\n" {
		t.Fatalf("small log = %q", got)
	}
	if readLogTail(filepath.Join(t.TempDir(), "none")) != "" {
		t.Fatal("a missing log read as text")
	}
}
