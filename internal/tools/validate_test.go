package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateExecutable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Clean(filepath.Join(dir, "tool"))
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateTool(Tool{Name: "a", Executable: file}, Context{}); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if err := validateTool(Tool{Name: "a", Executable: dir}, Context{}); err == nil {
		t.Fatal("directory should fail")
	}
	missing := filepath.Clean(filepath.Join(dir, "missing"))
	if err := validateTool(Tool{Name: "a", Executable: missing}, Context{}); err == nil {
		t.Fatal("missing should fail")
	}
}

func TestValidateToolExpandsGame(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "tool")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateTool(Tool{Name: "a", Executable: "{game}/tool"}, Context{Game: dir}); err != nil {
		t.Fatalf("placeholder executable: %v", err)
	}
}
