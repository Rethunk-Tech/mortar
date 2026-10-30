package contentpatcher

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/go-tilemask"
)

// The alpha mask of an overlay image is the slow part of a cold check (a recolour pack overlays hundreds of large
// sheets), so each mask is kept on disk, named by the image's path under the data folder, size and mtime: an
// unchanged image skips its decode, and a moved data folder keeps its masks.
const (
	alphaDiskDir  = "problems-overlay-alpha"
	alphaKeepFor  = 30 * 24 * time.Hour
	alphaTouchAge = 24 * time.Hour
)

var alphaPruned sync.Once

// alphaDiskPath names the cache file of the image behind fileKey (see pngFileKey), or "" when there is no data folder.
func alphaDiskPath(fileKey string) string {
	packDiskState.Lock()
	base := packDiskState.base
	packDiskState.Unlock()
	abs, version, _ := strings.Cut(fileKey, "\x00")
	if base == "" || abs == "" {
		return ""
	}
	dir := filepath.Join(base, "cache", alphaDiskDir)
	alphaPruned.Do(func() { pruneAlphaDir(dir) })
	sum := sha256.Sum256([]byte(packKey(base, abs) + "\x00" + version))
	return filepath.Join(dir, hex.EncodeToString(sum[:])+".gz")
}

func pruneAlphaDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-alphaKeepFor)
	for _, e := range entries {
		if info, err := e.Info(); err == nil && info.ModTime().Before(cutoff) {
			_ = safefs.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

func readAlphaDisk(path string) *tilemask.Mask {
	f, err := safefs.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	mask, err := tilemask.ReadMask(f, tilemask.Options{})
	if err != nil {
		return nil
	}
	if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) > alphaTouchAge {
		now := time.Now()
		_ = os.Chtimes(path, now, now)
	}
	return mask
}

func writeAlphaDisk(path string, mask *tilemask.Mask) {
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	_ = safefs.WriteStreamNoSync(path, 0o600, func(w io.Writer) error {
		_, err := mask.WriteTo(w)
		return err
	})
}
