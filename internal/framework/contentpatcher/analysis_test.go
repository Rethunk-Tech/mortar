package contentpatcher_test

import (
	"os/exec"
	"strings"
	"testing"
)

// The library package must import nothing under internal/: it is the half of Content Patcher's analysis that moves
// to its own module, so a Mortar type or helper reaching into it would travel with the move.
func TestLibraryImportsNoMortarInternalPackage(t *testing.T) {
	out, err := exec.CommandContext(t.Context(), "go", "list", "-deps", "./cp").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for pkg := range strings.SplitSeq(string(out), "\n") {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" || strings.HasSuffix(pkg, "/internal/framework/contentpatcher/cp") {
			continue
		}
		if strings.Contains(pkg, "mortar/internal/") {
			t.Errorf("cp imports %s", pkg)
		}
	}
}
