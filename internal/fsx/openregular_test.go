package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRegularRefusesADirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if _, err := OpenRegular(dir); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("directory: err = %v", err)
	}
	file := filepath.Join(dir, "a")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := OpenRegular(file)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
}
