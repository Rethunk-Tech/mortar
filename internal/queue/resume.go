package queue

import (
	"cmp"
	"context"
	"errors"
	"log"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/github"
)

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

func dropDownload(path string) {
	_ = os.Remove(path)
	_ = os.Remove(github.ResumeSidecar(path))
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
		id := strings.TrimSuffix(name, github.ResumeSuffix)
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
	if err != nil && errors.Is(err, github.ErrLinkExpired) {
		return err
	}
	if err == nil {
		// Only the host: a download address can carry a signed key.
		host := ""
		if u, perr := neturl.Parse(url); perr == nil {
			host = u.Host
		}
		log.Printf("download: %s from %s", cmp.Or(it.Name, it.Package, it.FileName), host)
	}
	return s.diskError(err, p.total)
}
