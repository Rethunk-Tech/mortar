package queue

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/archive"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/github"
)

// errLinkExpired is a CDN or host refusing this URL; Nexus then needs a fresh DownloadLinks call.
var errLinkExpired = errors.New("the download link has expired")

type resumeMeta struct {
	ExpectedSize int64  `json:"expectedSize"`
	ETag         string `json:"etag"`
	URL          string `json:"url"`
	Hash         string `json:"hash"`
}

func destPath(dir, id, fileName string) string {
	return filepath.Join(dir, downloadsDir, id+filepath.Ext(fileName))
}

func (s *Service) downloadRoot() string {
	if s.d.DownloadDir != nil {
		if d := strings.TrimSpace(s.d.DownloadDir()); filepath.IsAbs(d) {
			return d
		}
	}
	return filepath.Join(s.d.Dir, downloadsDir)
}

func (s *Service) dest(id, fileName string) string {
	return filepath.Join(s.downloadRoot(), id+filepath.Ext(fileName))
}

func resumeSidecar(path string) string {
	return path + ".resume.json"
}

func saveResume(path string, m resumeMeta) {
	_ = datadir.WriteJSON(resumeSidecar(path), m)
}

func dropDownload(path string) {
	_ = os.Remove(path)
	_ = os.Remove(resumeSidecar(path))
}

func (s *Service) dropDownloadUnlessKept(path string) {
	if s.d.KeepArchives != nil && s.d.KeepArchives() {
		return
	}
	dropDownload(path)
}

func (s *Service) sweepDownloads() {
	if s.d.KeepArchives != nil && s.d.KeepArchives() {
		return
	}
	root := s.downloadRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	keep := map[string]bool{}
	s.mu.Lock()
	for _, it := range s.items {
		if !finished(it.State) {
			keep[it.ID] = true
		}
	}
	s.mu.Unlock()
	for _, e := range entries {
		name := e.Name()
		id := strings.TrimSuffix(name, ".resume.json")
		id = strings.TrimSuffix(id, filepath.Ext(id))
		if keep[id] {
			continue
		}
		_ = os.Remove(filepath.Join(root, name))
	}
}

const fetchRestarts = 3

// fetch streams url to path, appending with Range when a partial and the server allow it.
func (s *Service) fetch(ctx context.Context, it Item, url, path string) error {
	p := &progress{s: s, id: it.ID, total: it.SizeKB << 10, last: s.d.Now()}
	err := github.Download(ctx, s.d.HTTP, url, path, archive.DefaultMaxTotalBytes, func(done, total int64) {
		if total > 0 {
			p.total = total
		}
		p.set(done)
	})
	if err != nil && err.Error() == errLinkExpired.Error() {
		return errLinkExpired
	}
	return s.diskError(err, p.total)
}
