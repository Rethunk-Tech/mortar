package contentpatcher_test

import (
	"os/exec"
	"strings"
	"testing"
)

// The library package must import nothing under internal/: it is the half of Content Patcher's analysis that moves
// to its own module, so a Mortar type or helper reaching into it would travel with the move.
func TestLibraryImportsNoMortarInternalPackage(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "./cp")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		pkg := strings.TrimSpace(line)
		if pkg == "" || strings.HasSuffix(pkg, "/internal/framework/contentpatcher/cp") {
			continue
		}
		if strings.Contains(pkg, "mortar/internal/") {
			t.Errorf("cp imports %s", pkg)
		}
	}
}
