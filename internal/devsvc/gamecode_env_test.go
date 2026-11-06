package devsvc

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIlspyRunsWithDotnetRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake tool is a shell script")
	}
	dir := t.TempDir()
	tool := filepath.Join(dir, "ilspycmd")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\necho \"root=$DOTNET_ROOT\"\n"), 0o700); err != nil { // #nosec G306 -- a fake tool the test must execute
		t.Fatal(err)
	}
	prev := flatpakDotnetRoot
	flatpakDotnetRoot = func() string { return "/app/extensions/dotnet10/lib" }
	t.Cleanup(func() { flatpakDotnetRoot = prev })

	out, err := (&Service{}).run(t.Context(), dir, tool)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != "root=/app/extensions/dotnet10/lib" {
		t.Fatalf("ilspycmd env: %q", got)
	}
	if isDotnetTool("git") {
		t.Fatal("git is not a dotnet tool")
	}
}
