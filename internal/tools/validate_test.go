package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateExecutable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "tool")
	if err := os.WriteFile(file, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateTool(Tool{Name: "a", Executable: file}); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if err := validateTool(Tool{Name: "a", Executable: dir}); err == nil {
		t.Fatal("directory should fail")
	}
	if err := validateTool(Tool{Name: "a", Executable: filepath.Join(dir, "missing")}); err == nil {
		t.Fatal("missing should fail")
	}
}
