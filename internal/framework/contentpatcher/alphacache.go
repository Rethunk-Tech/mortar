package contentpatcher

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

func readAlphaDisk(path string) (pngAlpha, bool) {
	f, err := fsx.Open(path)
	if err != nil {
		return pngAlpha{}, false
	}
	defer func() { _ = f.Close() }()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return pngAlpha{}, false
	}
	br := bufio.NewReader(zr)
	var w, h int
	if _, err := fmt.Fscanf(br, "%d %d\n", &w, &h); err != nil {
		return pngAlpha{}, false
	}
	if w <= 0 || h <= 0 || w > 1<<16 || h > 1<<16 {
		return pngAlpha{}, false
	}
	mask := acquireAlpha((w*h + 7) / 8)
	if _, err := io.ReadFull(br, mask); err != nil {
		releaseAlpha(mask)
		return pngAlpha{}, false
	}
	if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) > alphaTouchAge {
		now := time.Now()
		_ = os.Chtimes(path, now, now)
	}
	return pngAlpha{w: w, h: h, a: mask}, true
}

func writeAlphaDisk(path string, pix pngAlpha) {
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	_ = datadir.WriteStream(path, 0o600, func(w io.Writer) error {
		zw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(zw, "%d %d\n", pix.w, pix.h); err != nil {
			return err
		}
		if _, err := zw.Write(pix.a); err != nil {
			return err
		}
		return zw.Close()
	})
}
