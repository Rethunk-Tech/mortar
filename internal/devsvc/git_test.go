package devsvc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseStatus(t *testing.T) {
	out := "# branch.oid 1234\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +2 -0\n" +
		"1 .M N... 100644 100644 100644 aa bb src/Plugin.cs\n? notes.txt\n"
	if got := parseStatus(out); got != (GitState{Branch: "main", Changed: 2}) {
		t.Errorf("parseStatus = %+v", got)
	}
	// A detached HEAD names no branch; git prints "(detached)".
	if got := parseStatus("# branch.oid 1234\n# branch.head (detached)\n"); got.Branch != "(detached)" || got.Changed != 0 {
		t.Errorf("detached = %+v", got)
	}
}

func TestGitStateIsTheProjectsOwnRepository(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	svc.Run = nil // real git
	root := t.TempDir()
	nested := filepath.Join(root, "mods", "Nested")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.run(t.Context(), root, "git", "init", "-q"); err != nil {
		t.Skipf("no git: %v", err)
	}
	if svc.gitState(t.Context(), root) == nil {
		t.Error("the repository's own folder has no state")
	}
	if got := svc.gitState(t.Context(), nested); got != nil {
		t.Errorf("a folder inside another repository took its state: %+v", *got)
	}
}
