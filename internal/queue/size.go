package queue

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

const (
	// LargeDownloadBytes is the size above which a download waits for the player's go-ahead.
	LargeDownloadBytes int64 = 2048 << 20
	// spaceMargin is the free space left over after a download and its extraction, for the OS and Mortar's own files.
	spaceMargin int64 = 512 << 20
)

// NoSpaceError is a download refused before any bytes were fetched because it would not fit on Path's drive.
type NoSpaceError struct {
	Path       string
	Need, Free int64
}

func (e *NoSpaceError) Error() string {
	return fmt.Sprintf("Not enough disk space on %s: this download needs about %s free and only %s is available. Free some space, then Retry.",
		driveOf(e.Path), formatBytes(e.Need), formatBytes(e.Free))
}

func driveOf(p string) string {
	if v := filepath.VolumeName(p); v != "" {
		return v
	}
	return p
}

func formatBytes(n int64) string {
	if n >= 1<<30 {
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	}
	return fmt.Sprintf("%d MB", n>>20)
}

// within reports whether path lies under dir.
func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// freeAt is the free bytes on path's volume, measured at its nearest existing parent; ok is false when unknown.
func (s *Service) freeAt(path string) (int64, bool) {
	free := s.d.FreeBytes
	if free == nil {
		free = datadir.FreeBytes
	}
	for {
		n, err := free(path)
		if err == nil {
			return n, true
		}
		parent := filepath.Dir(path)
		if !errors.Is(err, fs.ErrNotExist) || parent == path {
			return 0, false
		}
		path = parent
	}
}

// checkSpace refuses a download of size bytes that cannot fit: the archive lands in the downloads folder and its
// extraction, which is taken as twice the archive when the real size is not yet known, in the store under Dir. A
// partial download already on disk counts as fetched. A volume whose free space cannot be read is not checked.
func (s *Service) checkSpace(it Item, size int64) error {
	extracted := 2 * size
	need := max(size, 0)
	if fi, err := os.Stat(s.dest(it.ID, it.FileName)); err == nil {
		need = max(need-fi.Size(), 0)
	}
	type volume struct {
		path string
		need int64
	}
	vols := []volume{{s.d.Dir, need + extracted + spaceMargin}}
	if root := s.downloadRoot(); !within(root, s.d.Dir) {
		vols = []volume{{root, need + spaceMargin}, {s.d.Dir, extracted + spaceMargin}}
	}
	for _, v := range vols {
		if free, ok := s.freeAt(v.path); ok && free < v.need {
			return &NoSpaceError{Path: v.path, Need: v.need, Free: free}
		}
	}
	return nil
}

// guardSize runs before a download's first byte. A download that cannot fit fails with NoSpaceError. One over
// LargeDownloadBytes parks in StateConfirmLarge until ConfirmLarge, and parked is then true.
func (s *Service) guardSize(it Item, size int64) (parked bool, err error) {
	if size <= 0 {
		return false, nil
	}
	if err := s.checkSpace(it, size); err != nil {
		return false, err
	}
	if size <= LargeDownloadBytes {
		return false, nil
	}
	s.mu.Lock()
	cur := s.find(it.ID)
	if cur == nil || cur.LargeOK || cur.State != StateDownloading {
		s.mu.Unlock()
		return false, nil
	}
	cur.State, cur.Progress, cur.Speed, cur.SizeKB = StateConfirmLarge, 0, 0, size>>10
	s.mu.Unlock()
	s.publish(true)
	return true, nil
}

// ConfirmLarge lets an item waiting in StateConfirmLarge download.
func (s *Service) ConfirmLarge(id string) {
	s.mu.Lock()
	if it := s.find(id); it != nil && it.State == StateConfirmLarge {
		it.State, it.LargeOK = StateQueued, true
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}
