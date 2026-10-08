package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnshareGivesALinkedFileItsOwnCopy(t *testing.T) {
	staging, game := t.TempDir(), t.TempDir()
	orig := filepath.Join(staging, "StardewModdingAPI.dll")
	if err := os.WriteFile(orig, []byte("smapi"), 0o600); err != nil {
		t.Fatal(err)
	}
	sym, hard, own := filepath.Join(game, "sym.dll"), filepath.Join(game, "hard.dll"), filepath.Join(game, "own.dll")
	if err := os.Symlink(orig, sym); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(orig, hard); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(own, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !Shared(sym) || !Shared(hard) || Shared(own) {
		t.Fatalf("Shared: sym %v hard %v own %v", Shared(sym), Shared(hard), Shared(own))
	}
	for _, p := range []string{sym, hard, own} {
		if err := Unshare(p); err != nil {
			t.Fatal(err)
		}
		if Shared(p) {
			t.Fatalf("%s still shared", p)
		}
		if err := os.WriteFile(p, []byte("mortar"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if b, _ := ReadFile(orig); string(b) != "smapi" {
		t.Fatalf("staging file changed to %q", b)
	}
}
