package steam

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteGridUsesDecimalAppIDAndExtension(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "cover.webp")
	body := []byte("image")
	if err := os.WriteFile(source, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeGrid(root, 123456789, source); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"123456789p.webp", "123456789.webp", "123456789_hero.webp"} {
		gridRoot, err := os.OpenRoot(filepath.Join(root, "grid"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := gridRoot.ReadFile(name)
		_ = gridRoot.Close()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !bytes.Equal(got, body) {
			t.Errorf("%s = %q, want %q", name, got, body)
		}
	}
}
