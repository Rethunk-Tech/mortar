//go:build linux

package datasvc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/testenv/testfs"

	"golang.org/x/sys/unix"
)

func TestMeasureReflinkCountedOnce(t *testing.T) {
	testfs.DataHome(t)
	root := t.TempDir()
	buf := make([]byte, 256*1024)
	for i := range buf {
		buf[i] = byte(i)
	}
	store := filepath.Join(root, "store", "g", "k", "m.bin")
	if err := os.MkdirAll(filepath.Dir(store), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store, buf, 0o600); err != nil {
		t.Fatal(err)
	}
	prof := filepath.Join(root, "profiles", "g", "0123456789abcdef", "mods", "k", "m.bin")
	if err := os.MkdirAll(filepath.Dir(prof), 0o700); err != nil {
		t.Fatal(err)
	}
	src, err := os.Open(filepath.Clean(store))
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.OpenFile(filepath.Clean(prof), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		_ = src.Close()
		t.Fatal(err)
	}
	err = unix.IoctlFileClone(int(dst.Fd()), int(src.Fd()))
	if cerr := src.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if cerr := dst.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		if errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.EXDEV) || errors.Is(err, unix.ENOTTY) {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	got, err := Measure(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	n := int64(len(buf))
	if got.Store < n {
		t.Fatalf("store = %d, want at least %d", got.Store, n)
	}
	if len(got.Profiles) != 1 || got.Profiles[0].Size > 4096 {
		t.Fatalf("profiles = %+v", got.Profiles)
	}
	if !got.SharedSavedKnown || got.SharedSaved < n {
		t.Fatalf("sharedSaved known=%v n=%d", got.SharedSavedKnown, got.SharedSaved)
	}
	if got.Total > got.Store+4096 {
		t.Fatalf("total = %d store = %d", got.Total, got.Store)
	}
}
