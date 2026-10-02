package datadir

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

var (
	ErrInside   = errors.New("the new folder is inside the current data folder")
	ErrNotEmpty = errors.New("the new folder is not empty")
	ErrNoSpace  = errors.New("not enough free space")
)

// SpaceError says how many bytes the move needs.
type SpaceError struct {
	Need int64
}

type RelocateEstimate struct {
	Bytes     int64 `json:"bytes"`
	FreeBytes int64 `json:"freeBytes"`
}

func EstimateRelocate(src, dest string) (RelocateEstimate, error) {
	need, err := Size(src)
	if err != nil {
		return RelocateEstimate{}, err
	}
	free, err := freeBytes(dest)
	if err != nil {
		free, err = freeBytes(filepath.Dir(dest))
		if err != nil {
			return RelocateEstimate{}, err
		}
	}
	return RelocateEstimate{Bytes: need, FreeBytes: free}, nil
}

func (e *SpaceError) Error() string {
	return fmt.Sprintf("not enough free space: this move needs about %d MB free", (e.Need+1024*1024-1)/(1024*1024))
}

func (e *SpaceError) Unwrap() error { return ErrNoSpace }

func hashFile(path string) ([32]byte, error) {
	var sum [32]byte
	f, err := fsx.Open(path)
	if err != nil {
		return sum, err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return sum, err
	}
	copy(sum[:], h.Sum(nil))
	return sum, nil
}

func verifyCopy(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if rel == PointerName {
			return nil
		}
		other := filepath.Join(dst, rel)
		a, err := hashFile(p)
		if err != nil {
			return err
		}
		b, err := hashFile(other)
		if err != nil {
			return err
		}
		if a != b {
			return fmt.Errorf("copy of %s did not match", rel)
		}
		return nil
	})
}

func emptyDir(path string) error {
	ents, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, 0o700)
		}
		return err
	}
	if len(ents) > 0 {
		return ErrNotEmpty
	}
	return nil
}

func removeOld(src, def string) error {
	if filepath.Clean(src) == filepath.Clean(def) {
		ents, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range ents {
			if e.Name() == PointerName {
				continue
			}
			if err := os.RemoveAll(filepath.Join(src, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	return os.RemoveAll(src)
}

// Relocate copies src into dest (which must be empty), verifies the copy, writes the pointer at def, then removes src.
func Relocate(src, dest, def string, reports ...func(CopyProgress)) error {
	src, dest, def = filepath.Clean(src), filepath.Clean(dest), filepath.Clean(def)
	var report func(CopyProgress)
	if len(reports) > 0 {
		report = reports[0]
	}
	if UnderRoot(src, dest) {
		return ErrInside
	}
	if err := emptyDir(dest); err != nil {
		return err
	}
	estimate, err := EstimateRelocate(src, dest)
	if err != nil {
		return err
	}
	if estimate.FreeBytes < estimate.Bytes {
		return &SpaceError{Need: estimate.Bytes}
	}
	if err := copyTree(src, dest, report); err != nil {
		_ = os.RemoveAll(dest)
		return err
	}
	if err := verifyCopy(src, dest); err != nil {
		_ = os.RemoveAll(dest)
		return err
	}
	if err := os.MkdirAll(def, 0o700); err != nil {
		return err
	}
	if err := WriteFile(filepath.Join(def, PointerName), []byte(dest+"\n"), 0o600); err != nil {
		return err
	}
	return removeOld(src, def)
}
