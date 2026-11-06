package devsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/dotnetroot"
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

func TestFlatpakDotnet(t *testing.T) {
	prevInside, prevRoot := flatpakInside, flatpakDotnetRoot
	flatpakInside = func() bool { return true }
	t.Cleanup(func() { flatpakInside, flatpakDotnetRoot = prevInside, prevRoot })

	flatpakDotnetRoot = func() string { return "" }
	svc, _ := testService(t, testTemplate)
	svc.Run = func(context.Context, string, string, ...string) ([]byte, error) { return nil, errors.New("absent") }
	if got := svc.Tools(t.Context(), "lethal-company").FlatpakDotnetInstall; got != dotnetroot.InstallCommand {
		t.Fatalf("hint without the extension = %q", got)
	}

	if runtime.GOOS == "windows" {
		return
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "dotnet"), []byte("#!/bin/sh\necho \"ext $1\"\n"), 0o700); err != nil { // #nosec G306 -- a fake dotnet the test must execute
		t.Fatal(err)
	}
	flatpakDotnetRoot = func() string { return root }
	if got := svc.Tools(t.Context(), "lethal-company").FlatpakDotnetInstall; got != "" {
		t.Fatalf("hint with the extension = %q", got)
	}
	svc.Run = nil
	out, err := svc.run(t.Context(), root, "dotnet", "--list-sdks")
	if err != nil || strings.TrimSpace(string(out)) != "ext --list-sdks" {
		t.Fatalf("dotnet ran as %q, %v", out, err)
	}
}
