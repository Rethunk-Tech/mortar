package queue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/archive"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
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

func resumeSidecar(path string) string {
	return path + ".resume.json"
}

func loadResume(path string) resumeMeta {
	b, err := os.ReadFile(resumeSidecar(path))
	if err != nil {
		return resumeMeta{}
	}
	var m resumeMeta
	if json.Unmarshal(b, &m) != nil {
		return resumeMeta{}
	}
	return m
}

func saveResume(path string, m resumeMeta) {
	_ = datadir.WriteJSON(resumeSidecar(path), m)
}

func dropDownload(path string) {
	_ = os.Remove(path)
	_ = os.Remove(resumeSidecar(path))
}

func (s *Service) sweepDownloads() {
	root := filepath.Join(s.d.Dir, downloadsDir)
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
	for n := 0; n <= fetchRestarts; n++ {
		err, retry := s.fetchOnce(ctx, it, url, path)
		if retry {
			dropDownload(path)
			continue
		}
		return err
	}
	return fmt.Errorf("the download server sent a range Mortar could not resume")
}

func (s *Service) fetchOnce(ctx context.Context, it Item, url, path string) (err error, retry bool) {
	meta := loadResume(path)
	if meta.URL == "" {
		meta.URL = url
	}
	offset := int64(0)
	if st, err := os.Stat(path); err == nil {
		offset = st.Size()
	}
	if meta.ExpectedSize > 0 && offset > meta.ExpectedSize {
		dropDownload(path)
		offset = 0
		meta = resumeMeta{URL: url}
	}
	if meta.ExpectedSize > 0 && offset == meta.ExpectedSize && offset > 0 {
		return verifyDownload(path, meta), false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err, false
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := s.d.HTTP.Do(req)
	if err != nil {
		return err, false
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		if offset > 0 {
			dropDownload(path)
			offset = 0
		}
	case http.StatusPartialContent:
		start, total, ok := parseContentRange(resp.Header.Get("Content-Range"))
		if !ok || start != offset {
			return nil, true
		}
		if total > 0 {
			meta.ExpectedSize = total
		}
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusGone:
		return errLinkExpired, false
	case http.StatusRequestedRangeNotSatisfiable:
		return nil, true
	default:
		return fmt.Errorf("the download server answered %s", resp.Status), false
	}

	const limit = archive.DefaultMaxTotalBytes
	tooLarge := fmt.Errorf("the download is larger than %d MiB", limit>>20)
	if etag := resp.Header.Get("ETag"); etag != "" {
		if meta.ETag != "" && meta.ETag != etag && offset > 0 && resp.StatusCode == http.StatusPartialContent {
			return nil, true
		}
		meta.ETag = etag
	}
	if h := hashFrom(resp.Header); h != "" {
		meta.Hash = h
	}
	meta.URL = url
	if resp.StatusCode == http.StatusOK && resp.ContentLength > 0 {
		meta.ExpectedSize = resp.ContentLength
	} else if resp.StatusCode == http.StatusPartialContent && meta.ExpectedSize == 0 && resp.ContentLength > 0 {
		meta.ExpectedSize = offset + resp.ContentLength
	}
	if meta.ExpectedSize > limit {
		return tooLarge, false
	}
	if resp.ContentLength > limit {
		return tooLarge, false
	}

	var f *os.File
	if offset > 0 {
		f, err = fsx.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	} else {
		f, err = fsx.Create(path)
	}
	if err != nil {
		return s.diskError(err, meta.ExpectedSize), false
	}
	defer func() { _ = f.Close() }()
	saveResume(path, meta)

	total := meta.ExpectedSize
	if total <= 0 {
		total = it.SizeKB << 10
	}
	p := &progress{s: s, id: it.ID, total: total, n: offset, from: offset, last: s.d.Now()}
	n, err := io.Copy(f, io.TeeReader(io.LimitReader(resp.Body, limit+1-offset), p))
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err(), false
		}
		return s.diskError(err, total), false
	}
	if offset+n > limit {
		return tooLarge, false
	}
	if err := s.diskError(f.Close(), total); err != nil {
		return err, false
	}
	st, err := os.Stat(path)
	if err != nil {
		return err, false
	}
	if meta.ExpectedSize > 0 && st.Size() != meta.ExpectedSize {
		dropDownload(path)
		return fmt.Errorf("the download size is %d bytes, not %d", st.Size(), meta.ExpectedSize), false
	}
	return verifyDownload(path, meta), false
}

func parseContentRange(h string) (start, total int64, ok bool) {
	h = strings.TrimSpace(strings.TrimPrefix(h, "bytes"))
	rangePart, totalPart, found := strings.Cut(strings.TrimSpace(h), "/")
	if !found {
		return 0, 0, false
	}
	from, _, ok := strings.Cut(rangePart, "-")
	if !ok {
		return 0, 0, false
	}
	start, err := strconv.ParseInt(from, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	if totalPart != "*" {
		total, err = strconv.ParseInt(totalPart, 10, 64)
		if err != nil {
			return 0, 0, false
		}
	}
	return start, total, true
}

func hashFrom(h http.Header) string {
	if v := strings.TrimSpace(h.Get("X-Checksum-Sha256")); v != "" {
		return "sha256:" + strings.ToLower(strings.TrimPrefix(v, "sha256:"))
	}
	return ""
}

func verifyDownload(path string, meta resumeMeta) error {
	if meta.Hash == "" {
		return nil
	}
	kind, want, _ := strings.Cut(meta.Hash, ":")
	if kind != "sha256" {
		return nil
	}
	f, err := fsx.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if !strings.EqualFold(want, hex.EncodeToString(h.Sum(nil))) {
		dropDownload(path)
		return errors.New("the download did not match its checksum")
	}
	return nil
}
