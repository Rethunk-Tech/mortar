package datadir

import (
	"errors"
	"fmt"
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
	free, err := FreeBytes(dest)
	if err != nil {
		free, err = FreeBytes(filepath.Dir(dest))
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
		a, err := fsx.SHA256(p)
		if err != nil {
			return err
		}
		b, err := fsx.SHA256(other)
		if err != nil {
			return err
		}
		if a != b {
			return fmt.Errorf("copy of %s did not match", rel)
		}
		return nil
	})
}

// linkingCopy copies each file, but re-creates a hard link for every further name of an inode already copied, so
// files the store shares with profiles stay shared on the target. A link the target refuses becomes a copy.
func linkingCopy() func(from, to, rel string) error {
	copied := map[fileKey]string{}
	return func(from, to, _ string) error {
		info, err := os.Stat(from)
		if err != nil {
			return err
		}
		key, linked := linkedKey(info)
		if !linked {
			return CopyFile(from, to)
		}
		if first, ok := copied[key]; ok && os.Link(first, to) == nil {
			return nil
		}
		if err := CopyFile(from, to); err != nil {
			return err
		}
		if _, ok := copied[key]; !ok {
			copied[key] = to
		}
		return nil
	}
}

// clampProgress keeps reported bytes within total, which counts each hard-linked inode once.
func clampProgress(report func(CopyProgress), total int64) func(CopyProgress) {
	if report == nil {
		return nil
	}
	return func(p CopyProgress) {
		p.Bytes = min(p.Bytes, total)
		p.TotalBytes = total
		report(p)
	}
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
	if err := copyTree(src, dest, clampProgress(report, estimate.Bytes), linkingCopy()); err != nil {
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
