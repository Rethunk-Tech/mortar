package datadir

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/go-safefs"
)

func seedTree(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"dll/mod.dll":          "dll-bytes",
		"A/config.json":        `{"a":1}`,
		"A/data/save.json":     `{"s":1}`,
		"A/saves/slot.dat":     "slot",
		"A/readme.txt":         "readme",
		"A/nested/config.json": `{"n":1}`,
		"A/mod.dll.mortar-old": "old",
	}
	for rel, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		testfs.WriteAt(t, p, []byte(body))
	}
}

func readAll(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		var b []byte
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, linkErr := os.Readlink(p)
			if linkErr != nil {
				return linkErr
			}
			b, err = safefs.ReadFile(target)
		} else {
			b, err = safefs.ReadFile(p)
		}
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func linked(t *testing.T, a, b string) bool {
	t.Helper()
	ia, err := os.Lstat(a)
	if err != nil {
		t.Fatal(err)
	}
	ib, err := os.Lstat(b)
	if err != nil {
		t.Fatal(err)
	}
	if ia.Mode()&os.ModeSymlink != 0 || ib.Mode()&os.ModeSymlink != 0 {
		return true
	}
	return os.SameFile(ia, ib)
}

func TestConfigJSONNeverHardlinkedOrSymlinked(t *testing.T) {
	src := t.TempDir()
	seedTree(t, src)
	for _, tier := range []safefs.Tier{safefs.TierHardlink, safefs.TierSymlink} {
		dst := t.TempDir()
		if err := onTier(tier).Tree(src, dst); err != nil {
			t.Fatal(err)
		}
		for _, rel := range []string{"A/config.json", "A/nested/config.json", "A/data/save.json", "A/saves/slot.dat", "A/readme.txt", "A/mod.dll.mortar-old"} {
			from := filepath.Join(src, filepath.FromSlash(rel))
			to := filepath.Join(dst, filepath.FromSlash(rel))
			if linked(t, from, to) {
				t.Fatalf("tier %d linked writable %s", tier, rel)
			}
		}
		if !linked(t, filepath.Join(src, "dll", "mod.dll"), filepath.Join(dst, "dll", "mod.dll")) {
			t.Fatalf("tier %d did not link the dll", tier)
		}
	}
}

func TestWriteConfigLeavesStoreAndPeerUnchanged(t *testing.T) {
	src := t.TempDir()
	seedTree(t, src)
	orig := readAll(t, src)
	for _, tier := range []safefs.Tier{safefs.TierClone, safefs.TierHardlink, safefs.TierSymlink, safefs.TierCopy} {
		a := t.TempDir()
		b := t.TempDir()
		m := onTier(tier)
		if err := m.Tree(src, a); err != nil {
			t.Fatal(err)
		}
		if err := m.Tree(src, b); err != nil {
			t.Fatal(err)
		}
		cfg := filepath.Join(a, "A", "config.json")
		if err := os.WriteFile(cfg, []byte(`{"a":99}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := readAll(t, src); got["A/config.json"] != orig["A/config.json"] {
			t.Fatalf("tier %d mutated store", tier)
		}
		if got := readAll(t, b); got["A/config.json"] != orig["A/config.json"] {
			t.Fatalf("tier %d mutated peer", tier)
		}
	}
}

func TestRemoveProfileLeavesStore(t *testing.T) {
	src := t.TempDir()
	seedTree(t, src)
	orig := readAll(t, src)
	for _, tier := range []safefs.Tier{safefs.TierClone, safefs.TierHardlink, safefs.TierSymlink, safefs.TierCopy} {
		dst := t.TempDir()
		if err := onTier(tier).Tree(src, dst); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(dst); err != nil {
			t.Fatal(err)
		}
		if got := readAll(t, src); len(got) != len(orig) {
			t.Fatalf("tier %d store files missing: %v", tier, got)
		}
		for k, v := range orig {
			if got := readAll(t, src)[k]; got != v {
				t.Fatalf("tier %d store %s = %q", tier, k, got)
			}
		}
	}
}

func TestWritableRel(t *testing.T) {
	if !WritableRel("config.json") || !WritableRel("Mod/config.json") || !WritableRel("data/x") || !WritableRel("saves/a") {
		t.Fatal("expected writable")
	}
	if WritableRel("manifest.json") || WritableRel("assets/foo.png") {
		t.Fatal("expected shipped")
	}
}

func TestEditableTextNeverSharesAnInode(t *testing.T) {
	for _, rel := range []string{"manifest.json", "Mod/content.json", "Mod/i18n/de.json", "Mod/notes.TXT", "Mod/maps/Farm.tmx"} {
		if !EditableText(rel) {
			t.Fatalf("%s: text files must never share an inode with the store", rel)
		}
	}
	for _, rel := range []string{"assets/foo.png", "Mod/Mod.dll", "Mod/assets/Music.ogg", "Mod/x.xnb"} {
		if EditableText(rel) {
			t.Fatalf("%s: binaries stay shareable", rel)
		}
	}
}

// onTier is Mortar's materializer limited to tier, falling back to a copy as every tier does.
func onTier(tier safefs.Tier) safefs.Materializer {
	m := materializer
	m.Tiers = []safefs.Tier{tier}
	return m
}
