package datadir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

type versioned struct {
	FormatVersion int    `json:"formatVersion"`
	Name          string `json:"name"`
}

func TestWriteVersionedAddsVersionAndReadsMissingAsCurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.json")
	if err := os.WriteFile(path, []byte(`{"name":"old"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteVersioned(path, versioned{FormatVersion: FormatVersion, Name: "new"}); err != nil {
		t.Fatalf("a file without a version must be writable: %v", err)
	}
	b, _ := fsx.ReadFile(path)
	if !strings.Contains(string(b), `"formatVersion": 1`) {
		t.Fatalf("version not written: %s", b)
	}
}

func TestWriteVersionedRefusesNewerAndKeepsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.json")
	const newer = `{"formatVersion":99,"name":"future"}`
	if err := os.WriteFile(path, []byte(newer), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteVersioned(path, versioned{FormatVersion: FormatVersion}); err == nil {
		t.Fatal("expected refusal")
	}
	for _, p := range []string{path, path + ".newer"} {
		if b, _ := fsx.ReadFile(p); string(b) != newer {
			t.Fatalf("%s = %q", p, b)
		}
	}
}
