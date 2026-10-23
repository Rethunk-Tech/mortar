package datadir

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// Size returns the total size in bytes of the regular files under dir, counting hard-linked files once. Unreadable entries below dir are
// skipped, so the size is a best-effort total; only a failure to read dir itself is an error.
func Size(dir string) (int64, error) { return size(dir, true) }

// sizeWalkers bounds the folders sized at once; more only contends in the filesystem.
const sizeWalkers = 8

// size is Size; with dedupe false every name counts in full, as it does on a target that refuses hard links.
// The folders directly under dir are walked concurrently, which is where a mods folder's time goes.
func size(dir string, dedupe bool) (int64, error) {
	var (
		total atomic.Int64
		mu    sync.Mutex
		seen  = map[fileKey]bool{}
	)
	visit := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return rootError(path, dir, err)
		}
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			return nil
		}
		if info, infoErr := d.Info(); infoErr == nil {
			if key, linked := linkedKey(path, info); linked && dedupe {
				mu.Lock()
				dup := seen[key]
				seen[key] = true
				mu.Unlock()
				if dup {
					return nil
				}
			}
			total.Add(info.Size())
		}
		return nil
	}
	if st, err := os.Lstat(dir); err != nil || !st.IsDir() {
		err = filepath.WalkDir(dir, visit)
		return total.Load(), err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	var wg sync.WaitGroup
	slots := make(chan struct{}, sizeWalkers)
	for _, e := range ents {
		path := filepath.Join(dir, e.Name())
		if !e.IsDir() {
			_ = visit(path, e, nil)
			continue
		}
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			_ = filepath.WalkDir(path, visit)
		})
	}
	wg.Wait()
	return total.Load(), nil
}

// fileKey identifies one inode on one device; the third word is the high half of a 128-bit Windows file id.
type fileKey [3]uint64

func rootError(path, root string, err error) error {
	if path == root {
		return err
	}
	return nil
}
