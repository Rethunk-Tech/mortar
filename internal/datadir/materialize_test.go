package datadir

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
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
	}
	for rel, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := fsx.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
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
			b, err = fsx.ReadFile(target)
		} else {
			b, err = fsx.ReadFile(p)
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

func opsFor(t *testing.T, tier Tier) Ops {
	t.Helper()
	fail := func(error) func(string, string) error {
		return func(string, string) error { return syscall.EOPNOTSUPP }
	}
	switch tier {
	case TierClone:
		return Ops{Tiers: []Tier{TierClone}, Clone: CopyFile, DisableCache: true}
	case TierHardlink:
		return Ops{Tiers: []Tier{TierHardlink, TierCopy}, Hardlink: os.Link, Copy: CopyFile, DisableCache: true}
	case TierSymlink:
		return Ops{
			Tiers: []Tier{TierSymlink, TierCopy},
			Symlink: func(src, dst string) error {
				abs, err := filepath.Abs(src)
				if err != nil {
					return err
				}
				return os.Symlink(abs, dst)
			},
			Copy:         CopyFile,
			DisableCache: true,
		}
	case TierCopy:
		return Ops{Tiers: []Tier{TierCopy}, Copy: CopyFile, Clone: fail(nil), Hardlink: fail(nil), Symlink: fail(nil), DisableCache: true}
	default:
		t.Fatalf("tier %d", tier)
		return Ops{}
	}
}

func TestMaterializeTiersMatchContent(t *testing.T) {
	src := t.TempDir()
	seedTree(t, src)
	want := readAll(t, src)
	for _, tier := range []Tier{TierClone, TierHardlink, TierSymlink, TierCopy} {
		dst := t.TempDir()
		if err := MaterializeTreeOps(src, dst, opsFor(t, tier)); err != nil {
			t.Fatalf("tier %d: %v", tier, err)
		}
		got := readAll(t, dst)
		if len(got) != len(want) {
			t.Fatalf("tier %d files %d want %d", tier, len(got), len(want))
		}
		for k, v := range want {
			if got[k] != v {
				t.Fatalf("tier %d %s = %q want %q", tier, k, got[k], v)
			}
		}
	}
}

func TestConfigJSONNeverHardlinkedOrSymlinked(t *testing.T) {
	src := t.TempDir()
	seedTree(t, src)
	for _, tier := range []Tier{TierHardlink, TierSymlink} {
		dst := t.TempDir()
		if err := MaterializeTreeOps(src, dst, opsFor(t, tier)); err != nil {
			t.Fatal(err)
		}
		for _, rel := range []string{"A/config.json", "A/nested/config.json", "A/data/save.json", "A/saves/slot.dat", "A/readme.txt"} {
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
	for _, tier := range []Tier{TierClone, TierHardlink, TierSymlink, TierCopy} {
		a := t.TempDir()
		b := t.TempDir()
		ops := opsFor(t, tier)
		if err := MaterializeTreeOps(src, a, ops); err != nil {
			t.Fatal(err)
		}
		if err := MaterializeTreeOps(src, b, ops); err != nil {
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
	for _, tier := range []Tier{TierClone, TierHardlink, TierSymlink, TierCopy} {
		dst := t.TempDir()
		if err := MaterializeTreeOps(src, dst, opsFor(t, tier)); err != nil {
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

func TestMaterializeFallbackChain(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(src, "f.bin"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	var seq []string
	track := func(name string, err error) func(string, string) error {
		return func(s, d string) error {
			seq = append(seq, name)
			if err != nil {
				return err
			}
			return CopyFile(s, d)
		}
	}
	ops := Ops{
		Tiers:        []Tier{TierClone, TierHardlink, TierSymlink, TierCopy},
		Clone:        track("clone", errNoClone),
		Hardlink:     track("hardlink", errCrossDevice),
		Symlink:      track("symlink", errBadLink),
		Copy:         track("copy", nil),
		DisableCache: true,
	}
	if err := MaterializeTreeOps(src, dst, ops); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, filepath.Join(dst, "f.bin"))) != "x" {
		t.Fatal("missing copy")
	}
	if len(seq) != 4 || seq[0] != "clone" || seq[1] != "hardlink" || seq[2] != "symlink" || seq[3] != "copy" {
		t.Fatalf("seq %v", seq)
	}
	cfgSrc := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(cfgSrc, "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	seq = nil
	if err := MaterializeTreeOps(cfgSrc, t.TempDir(), ops); err != nil {
		t.Fatal(err)
	}
	for _, s := range seq {
		if s == "hardlink" || s == "symlink" {
			t.Fatalf("writable used %s: %v", s, seq)
		}
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := fsx.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
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

func TestMaterializeFallbackNonFallbackError(t *testing.T) {
	src := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(src, "f.bin"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := errors.New("disk")
	err := MaterializeTreeOps(src, t.TempDir(), Ops{
		Tiers:        []Tier{TierClone},
		Clone:        func(string, string) error { return want },
		DisableCache: true,
	})
	if !errors.Is(err, want) {
		t.Fatalf("err %v", err)
	}
}

// MaterializeTreeOps is MaterializeTree with injectable tiers.
func MaterializeTreeOps(src, dst string, ops Ops) error {
	if len(ops.Tiers) == 0 {
		ops.Tiers = defaultOps().Tiers
	}
	if ops.Clone == nil {
		ops.Clone = cloneFile
	}
	if ops.Hardlink == nil {
		ops.Hardlink = os.Link
	}
	if ops.Symlink == nil {
		ops.Symlink = defaultOps().Symlink
	}
	if ops.Copy == nil {
		ops.Copy = CopyFile
	}
	_, err := copyTree(src, dst, ops.put)
	return err
}
