package lan

import (
	"archive/tar"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	// TransferProgressEvent carries file-transfer progress to the window.
	TransferProgressEvent = "lan:transfer"

	maxTarEntries = 10000
	maxTarBytes   = int64(2 << 30)
)

// TransferProgress reports one accepted share's store transfer.
type TransferProgress struct {
	ID      int    `json:"id"`
	Current int    `json:"current"`
	Total   int    `json:"total"`
	Bytes   int64  `json:"bytes"`
	Rate    int64  `json:"rate"`
	Done    bool   `json:"done"`
	Error   string `json:"error,omitempty"`
}

type transferGrant struct {
	Game    string
	Keys    map[string]bool
	Expires time.Time
}

type incomingTransfer struct {
	Peer    string
	Sender  string
	Game    string
	Token   string
	Items   []transferItem
	Expires time.Time
}

var errRemoteStoreEntryMissing = errors.New("peer does not have this store entry")

// Transfer copies missing store entries for an accepted paired share.
func (s *Service) Transfer(ctx context.Context, id int) (err error) {
	s.mu.Lock()
	incoming, ok := s.incoming[id]
	if !ok || time.Now().After(incoming.Expires) {
		s.mu.Unlock()
		return usererr.New(usererr.NotFound, "this share has expired; ask them to send it again")
	}
	if _, running := s.active[id]; running {
		s.mu.Unlock()
		return usererr.New(usererr.Busy, "the files for this share are already being copied")
	}
	ctx, cancel := context.WithCancel(ctx)
	s.active[id] = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.active, id)
		s.mu.Unlock()
	}()

	if s.deps.Store == nil {
		return errors.New("local store is unavailable")
	}
	// A cancelled transfer must still tell the sender it was cancelled.
	detached := context.WithoutCancel(ctx)
	missing := make([]string, 0, len(incoming.Items))
	for _, item := range incoming.Items {
		if has, err := s.storeHas(incoming.Game, item.Key); err == nil && !has {
			missing = append(missing, item.Key)
		}
	}
	s.reportOutgoing(detached, incoming, outgoingReport{State: outgoingPlan, Keys: missing})
	started := time.Now()
	var bytes int64
	log.Printf("lan: copying %d of %d files from %s", len(missing), len(incoming.Items), incoming.Sender)
	defer func() {
		switch {
		case err == nil:
			log.Printf("lan: copied %d files, %d bytes from %s in %s", len(missing), bytes, incoming.Sender, time.Since(started).Round(time.Millisecond))
			s.reportOutgoing(detached, incoming, outgoingReport{State: OutgoingDone})
		case errors.Is(err, context.Canceled):
			log.Printf("lan: copying files from %s cancelled", incoming.Sender)
			s.reportOutgoing(detached, incoming, outgoingReport{State: OutgoingCancelled})
		default:
			log.Printf("lan: copying files from %s failed: %v", incoming.Sender, err)
			s.reportOutgoing(detached, incoming, outgoingReport{State: OutgoingFailed, Reason: err.Error()})
		}
	}()
	total := len(incoming.Items)
	for current, item := range incoming.Items {
		if err := ctx.Err(); err != nil {
			s.emitTransfer(TransferProgress{ID: id, Current: current, Total: total, Bytes: bytes, Rate: transferRate(bytes, started), Error: err.Error()})
			return err
		}
		exists, err := s.storeHas(incoming.Game, item.Key)
		if err != nil {
			s.emitTransferError(id, current, total, bytes, started, err)
			return err
		}
		if !exists {
			n, err := s.fetchEntry(ctx, incoming, item, id, current, total, bytes, started)
			if errors.Is(err, errRemoteStoreEntryMissing) {
				n = 0
			} else if err != nil {
				s.emitTransferError(id, current, total, bytes, started, err)
				return err
			}
			bytes += n
		}
		s.emitTransfer(TransferProgress{
			ID: id, Current: current + 1, Total: total, Bytes: bytes, Rate: transferRate(bytes, started),
		})
	}
	s.emitTransfer(TransferProgress{
		ID: id, Current: total, Total: total, Bytes: bytes, Rate: transferRate(bytes, started), Done: true,
	})
	return nil
}

// reportOutgoing tells the sender how the pull stands, so its window can show it; the pull itself never depends on
// the answer.
func (s *Service) reportOutgoing(ctx context.Context, incoming incomingTransfer, report outgoingReport) {
	body, err := json.Marshal(report)
	if err != nil {
		return
	}
	endpoint, err := shareEndpoint(incoming.Peer, "/outgoing")
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	request.Header.Set("Authorization", "Bearer "+incoming.Token)
	request.Header.Set("Content-Type", "application/json")
	if response, err := (&http.Client{}).Do(request); err == nil {
		_ = response.Body.Close()
	}
}

// CancelTransfer stops an accepted share's active file transfer.
func (s *Service) CancelTransfer(id int) {
	s.mu.RLock()
	cancel := s.active[id]
	s.mu.RUnlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Service) storeHas(game, key string) (bool, error) {
	_, err := s.deps.Store.Path(game, key)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, store.ErrNotFound):
		return false, nil
	default:
		return false, err
	}
}

func (s *Service) fetchEntry(
	ctx context.Context,
	incoming incomingTransfer,
	item transferItem,
	id, current, total int,
	bytes int64,
	started time.Time,
) (int64, error) {
	key := item.Key
	endpoint, err := shareEndpoint(incoming.Peer, "/store/"+incoming.Game+"/"+key)
	if err != nil {
		return 0, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Authorization", "Bearer "+incoming.Token)
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		return 0, usererr.Wrap(usererr.Network, fmt.Errorf("%s went away before sending a file: %w", incoming.Sender, err))
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode == http.StatusNotFound {
		return 0, errRemoteStoreEntryMissing
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return 0, usererr.New(usererr.Network, fmt.Sprintf("%s refused to send a file (HTTP %d)", incoming.Sender, response.StatusCode))
	}
	temp, err := os.MkdirTemp("", "mortar-lan-transfer-")
	if err != nil {
		return 0, err
	}
	defer func() { _ = fsx.RemoveAll(temp) }()
	var received int64
	var last time.Time
	if err := extractTar(ctx, response.Body, temp, &received, func(n int64) {
		if now := time.Now(); now.Sub(last) >= progressEvery {
			last = now
		} else {
			return
		}
		s.emitTransfer(TransferProgress{
			ID: id, Current: current, Total: total, Bytes: bytes + n, Rate: transferRate(bytes+n, started),
		})
	}); err != nil {
		return 0, receiveError(incoming.Sender, err)
	}
	got, err := store.HashDir(temp)
	if err != nil {
		return 0, fmt.Errorf("check store entry %s: %w", key, err)
	}
	if item.Hash == "" || got != item.Hash {
		return 0, usererr.New(usererr.Damaged, fmt.Sprintf("a file from %s arrived damaged: it does not match what they vouched for", incoming.Sender))
	}
	// The hash the sender vouched for is the check: a local key names the archive it came from, not this folder.
	if err := s.deps.Store.AddDir(incoming.Game, key, temp); err != nil {
		return 0, fmt.Errorf("install store entry %s: %w", key, err)
	}
	if item.Source != "" {
		if err := s.deps.Store.Describe(incoming.Game, key, item.Source, item.Package, item.Version); err != nil {
			return 0, fmt.Errorf("record store entry %s: %w", key, err)
		}
	}
	log.Printf("lan: %s from %s", cmp.Or(item.Package, "a file"), incoming.Sender)
	return received, nil
}

// receiveError says what stopped a file's copy: the disk, the sender going away, or a file that cannot be unpacked.
func receiveError(sender string, err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return err
	case usererr.IsDiskFull(err):
		return usererr.Wrap(usererr.DiskFull, fmt.Errorf("there is no room left to copy files from %s: %w", sender, err))
	case errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, io.EOF), usererr.KindOf(err) == usererr.Network:
		return usererr.Wrap(usererr.Network, fmt.Errorf("%s went away while sending files: %w", sender, err))
	}
	return usererr.Wrap(usererr.Damaged, fmt.Errorf("a file from %s could not be unpacked: %w", sender, err))
}

func (s *Service) handleStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/store/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.Contains(parts[1], `\`) {
		http.Error(w, "invalid store path", http.StatusBadRequest)
		return
	}
	game, key := parts[0], parts[1]
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" || !s.grantAllows(token, game, key) {
		http.Error(w, "store transfer is not authorized", http.StatusForbidden)
		return
	}
	if s.deps.Store == nil {
		http.Error(w, "local store is unavailable", http.StatusInternalServerError)
		return
	}
	root, err := s.deps.Store.Path(game, key)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "store entry not found", http.StatusNotFound)
			return
		}
		http.Error(w, "could not read store entry", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/x-tar")
	counted := countingWriter{ResponseWriter: w, add: func(n int64) {
		s.updateOutgoing(token, func(o *OutgoingTransfer) { o.Bytes += n })
	}}
	if err := writeTar(r.Context(), counted, root); err != nil {
		return
	}
	s.updateOutgoing(token, func(o *OutgoingTransfer) { o.Current++ })
}

func (s *Service) grantAllows(token, game, key string) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for value, grant := range s.grants {
		if now.After(grant.Expires) {
			delete(s.grants, value)
		}
	}
	grant, ok := s.grants[token]
	return ok && grant.Game == game && grant.Keys[key] && now.Before(grant.Expires)
}

func (s *Service) rememberGrant(token, game string, keys []string) {
	if token == "" {
		return
	}
	allowed := make(map[string]bool, len(keys))
	for _, key := range keys {
		allowed[key] = true
	}
	s.mu.Lock()
	s.grants[token] = transferGrant{Game: game, Keys: allowed, Expires: time.Now().Add(transferTTL)}
	s.mu.Unlock()
}

// progressEvery is the cadence of per-file byte progress inside one store entry, the same as the queue's.
const progressEvery = 250 * time.Millisecond

func (s *Service) emitTransfer(progress TransferProgress) {
	if s.deps.Emit != nil {
		s.deps.Emit(TransferProgressEvent, progress)
	}
}

func (s *Service) emitTransferError(id, current, total int, bytes int64, started time.Time, err error) {
	s.emitTransfer(TransferProgress{
		ID: id, Current: current, Total: total, Bytes: bytes, Rate: transferRate(bytes, started), Error: err.Error(),
	})
}

func transferRate(bytes int64, started time.Time) int64 {
	seconds := time.Since(started).Seconds()
	if seconds < 1 {
		seconds = 1
	}
	return int64(float64(bytes) / seconds)
}

type contextReader struct {
	done <-chan struct{}
	r    io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	select {
	case <-r.done:
		return 0, context.Canceled
	default:
		return r.r.Read(p)
	}
}

func extractTar(ctx context.Context, source io.Reader, root string, total *int64, progress func(int64)) error {
	reader := tar.NewReader(io.LimitReader(source, maxTarBytes+1))
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer func() { _ = rootHandle.Close() }()
	entries := 0
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		entries++
		if entries > maxTarEntries || header.Size < 0 {
			return errors.New("store transfer has too many entries")
		}
		relative := path.Clean(strings.ReplaceAll(header.Name, `\`, "/"))
		if relative == "." || path.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, "../") {
			return errors.New("store transfer contains an invalid path")
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := rootHandle.MkdirAll(relative, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := rootHandle.MkdirAll(path.Dir(relative), 0o700); err != nil {
				return err
			}
			file, err := rootHandle.OpenFile(relative, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			if err != nil {
				return err
			}
			n, copyErr := io.CopyN(file, contextReader{done: ctx.Done(), r: reader}, header.Size)
			closeErr := file.Close()
			if err := errors.Join(copyErr, closeErr); err != nil {
				return err
			}
			if n != header.Size {
				return errors.New("store transfer file ended early")
			}
			*total += n
			if *total > maxTarBytes {
				return errors.New("store transfer is too large")
			}
			progress(*total)
		default:
			return errors.New("store transfer contains a link or special file")
		}
	}
}

func writeTar(ctx context.Context, output io.Writer, root string) error {
	writer := tar.NewWriter(output)
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer func() { _ = rootHandle.Close() }()
	err = fs.WalkDir(rootHandle.FS(), ".", func(relative string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if relative == "." || relative == store.CompleteMarker {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("store entry contains a symlink")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return errors.New("store entry contains a special file")
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relative
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if !info.IsDir() {
			file, err := rootHandle.Open(relative)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(writer, contextReader{done: ctx.Done(), r: file})
			closeErr := file.Close()
			if err := errors.Join(copyErr, closeErr); err != nil {
				return err
			}
		}
		return nil
	})
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	return err
}
