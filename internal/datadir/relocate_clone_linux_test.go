//go:build linux

package datadir

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"golang.org/x/sys/unix"
)

// firstPhysical is the disk address of f's first extent, via FS_IOC_FIEMAP; clones share it.
func firstPhysical(t *testing.T, path string) uint64 {
	t.Helper()
	f, err := fsx.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var q struct {
		start, length             uint64
		flags, mapped, count, pad uint32
		logical, physical, extLen uint64
		res                       [2]uint64
		extFlags                  uint32
		extPad                    [3]uint32
	}
	q.length, q.count, q.flags = ^uint64(0), 1, 1 // FIEMAP_FLAG_SYNC: allocate delayed extents first
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), 0xc020660b, reflect.ValueOf(&q).Pointer()); errno != 0 || q.mapped == 0 {
		t.Skipf("no FIEMAP: %v", errno)
	}
	return q.physical
}

func TestRelocatorClonesProfileFilesFromTheCopiedStoreAcrossFilesystems(t *testing.T) {
	const size = 1 << 20
	root := t.TempDir()
	if !clonesSupported(root) {
		t.Skip("temp dir cannot clone files")
	}
	src, dest := filepath.Join(root, "src"), filepath.Join(root, "dest")
	storeMod := filepath.Join(src, "store", "stardew", "k")
	profMod := filepath.Join(src, "profiles", "stardew", "p", "mods", "k")
	for _, d := range []string{storeMod, profMod} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	a, b := bytes.Repeat([]byte("a"), size), bytes.Repeat([]byte("b"), size)
	for _, f := range []struct {
		path string
		data []byte
	}{
		{filepath.Join(storeMod, "a.dll"), a},
		{filepath.Join(storeMod, "b.dll"), b},
		{filepath.Join(profMod, "a.dll"), a},
		{filepath.Join(profMod, "b.dll"), bytes.Repeat([]byte("c"), size)},
	} {
		if err := os.WriteFile(f.path, f.data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r := newRelocator(dest)
	r.same = func(string, string) bool { return false }
	if _, err := copyTreeFirst(src, dest, "store", nil, r.put); err != nil {
		t.Fatal(err)
	}
	if err := r.verifyCopy(); err != nil {
		t.Fatal(err)
	}
	prof := filepath.Join(dest, "profiles", "stardew", "p", "mods", "k")
	store := filepath.Join(dest, "store", "stardew", "k")
	if firstPhysical(t, filepath.Join(prof, "a.dll")) != firstPhysical(t, filepath.Join(store, "a.dll")) {
		t.Error("identical profile file was copied, not cloned")
	}
	if firstPhysical(t, filepath.Join(prof, "b.dll")) == firstPhysical(t, filepath.Join(store, "b.dll")) {
		t.Error("edited profile file shares the store file's extent")
	}
}
