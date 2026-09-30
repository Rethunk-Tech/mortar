package queue

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Rethunk-AI/mortar/internal/archive"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/github"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/store"
)

type action int

const (
	resolve action = iota
	click
	fetch
	install
)

// usable is a download key that has not expired.
func (s *Service) usable(it *Item) bool {
	return it.key != "" && it.expires > s.d.Now().Add(keyValidMargin).Unix()
}

// next decides what to do now, under the lock; a nil item means nothing. An item that can download goes first (premium, or holding a key
// from a link); otherwise the head item waits for its click, unless one already does. Items for a profile the game
// runs stay queued, and held reports whether any did.
func (s *Service) next() (it *Item, act action, held bool) {
	if s.paused || s.until.After(s.d.Now()) {
		return nil, resolve, false
	}
	premium := s.d.Premium()
	var head *Item
	waiting := false
	for _, it := range s.items {
		switch {
		case it.State == StateWaitingClick:
			waiting = true
		case it.State != StateQueued:
		case s.d.Running != nil && s.d.Running(it.Game, it.Profile):
			held = true
		case it.Repo != "":
			return it, s.forAsset(it), held
		case premium || s.usable(it):
			return it, s.forFile(it, fetch), held
		case head == nil:
			head = it
		}
	}
	if head == nil || waiting {
		return nil, resolve, held
	}
	return head, s.forFile(head, click), held
}

// forFile is resolve while the item's file is not known yet.
func (s *Service) forFile(it *Item, then action) action {
	if it.FileID == 0 || it.FileName == "" {
		return resolve
	}
	return then
}

// forAsset is resolve while the release and asset are not chosen, and install once the download is staged.
func (s *Service) forAsset(it *Item) action {
	switch {
	case it.staged != "":
		return install
	case it.Asset == "" || it.Tag == "":
		return resolve
	}
	return fetch
}

// heldRecheck is how often items held for a running profile are looked at again: nothing tells the queue when a
// game stops.
const heldRecheck = 5 * time.Second

func (s *Service) run(ctx context.Context) {
	for {
		for s.step(ctx) {
		}
		var wake <-chan time.Time
		s.mu.Lock()
		_, _, held := s.next()
		if d := s.until.Sub(s.d.Now()); d > 0 {
			wake = time.After(d)
		} else if held {
			wake = time.After(heldRecheck)
		}
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-s.kick:
		case <-wake:
			s.publish(false)
		}
	}
}

func (s *Service) step(ctx context.Context) bool {
	s.mu.Lock()
	it, act, _ := s.next()
	if it == nil || ctx.Err() != nil {
		s.mu.Unlock()
		return false
	}
	snap := *it
	if act == click {
		it.State = StateWaitingClick
		it.key, it.expires = "", 0
	}
	switch act {
	case fetch:
		it.State, it.Progress, it.Speed = StateDownloading, 0, 0
	case install:
		// Nothing is left to download, and Cancel does not reach an install under way.
		it.State, it.Progress, it.Speed = StateInstalling, 100, 0
	}
	itemCtx, cancel := context.WithCancel(ctx)
	s.cancels[it.ID] = cancel
	s.mu.Unlock()
	defer func() {
		cancel()
		s.mu.Lock()
		delete(s.cancels, snap.ID)
		s.mu.Unlock()
	}()
	s.publish(true)
	switch act {
	case click:
		if err := s.OpenPage(snap.ID); err != nil {
			s.settle(snap.ID, err)
		}
	case resolve:
		s.settle(snap.ID, s.resolve(itemCtx, snap))
	case fetch:
		s.settle(snap.ID, s.download(itemCtx, snap))
	case install:
		s.settle(snap.ID, s.installStaged(snap))
	}
	return true
}

// settle records how a step ended: nil leaves the item as the step set it.
func (s *Service) settle(id string, err error) {
	s.mu.Lock()
	it := s.find(id)
	if it == nil || err == nil {
		s.mu.Unlock()
		return
	}
	var limit *nexus.RateLimitError
	var ghLimit *github.RateLimitError
	var full *store.DiskFullError
	switch {
	case errors.Is(err, context.Canceled):
		// Cancel already set the state; a shutdown leaves the item to start over next time.
	case errors.As(err, &limit):
		it.State, it.Progress, it.Speed = StateQueued, 0, 0
		s.until = limit.Reset
		if !s.until.After(s.d.Now()) {
			s.until = s.d.Now().Add(defaultBackoff)
		}
	case errors.As(err, &ghLimit):
		it.State, it.Progress, it.Speed = StateQueued, 0, 0
		s.until = ghLimit.Reset
		if !s.until.After(s.d.Now()) {
			s.until = s.d.Now().Add(defaultBackoff)
		}
	case errors.Is(err, nexus.ErrPremiumRequired):
		it.State, it.Progress, it.Speed, it.key = StateWaitingClick, 0, 0, ""
	case errors.Is(err, nexus.ErrUnauthorized):
		it.State, it.Error = StateFailed, "Nexus rejected your API key: sign in again in Settings"
	case errors.As(err, &full):
		it.State, it.Error = StateFailed, fmt.Sprintf("Not enough disk space: about %d MB is needed", full.NeedMB)
	default:
		it.State, it.Error = StateFailed, err.Error()
	}
	reopen := it.State == StateWaitingClick
	s.mu.Unlock()
	s.publish(true)
	if reopen {
		// An expired key, or a premium-only answer: the page gives a fresh one.
		if err := s.OpenPage(id); err != nil {
			s.settle(id, err)
		}
	}
}

// resolve fills in the file an item names, or chooses it by version.
func (s *Service) resolve(ctx context.Context, it Item) error {
	if it.Repo != "" {
		return s.resolveGitHub(ctx, it)
	}
	c, err := s.d.Client()
	if err != nil {
		return err
	}
	files, err := c.Files(ctx, it.ModID)
	if err != nil {
		return err
	}
	var file nexus.File
	if it.FileID != 0 {
		for _, f := range files {
			if f.FileID == it.FileID {
				file = f
			}
		}
	} else {
		file, _ = ChooseFile(files, it.Version, it.Current)
	}
	if file.FileID == 0 {
		return errors.New("no suitable file for this mod is listed on Nexus")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if cur := s.find(it.ID); cur != nil {
		cur.FileID, cur.FileName, cur.SizeKB = file.FileID, cmp.Or(file.FileName, fmt.Sprintf("file-%d", file.FileID)), file.SizeKB
		cur.Version = cmp.Or(cur.Version, file.Version)
	}
	return nil
}

func (s *Service) download(ctx context.Context, it Item) error {
	if it.Repo != "" {
		return s.downloadGitHub(ctx, it)
	}
	c, err := s.d.Client()
	if err != nil {
		return err
	}
	var mod nexus.Mod
	// The picture and name are nice to have: a page that cannot be fetched only leaves the letter tile.
	if m, merr := c.Mod(ctx, it.ModID); merr == nil {
		mod = m
		s.mu.Lock()
		if cur := s.find(it.ID); cur != nil && cur.Name == "" {
			cur.Name = m.Name
		}
		s.mu.Unlock()
	}
	s.mu.Lock()
	key, expires := "", int64(0)
	if cur := s.find(it.ID); cur != nil && s.usable(cur) {
		key, expires = cur.key, cur.expires
	}
	s.mu.Unlock()
	links, err := c.DownloadLinks(ctx, it.ModID, it.FileID, key, expires)
	if err != nil {
		return err
	}
	if len(links) == 0 {
		return errors.New("the download has no link")
	}
	if err := os.MkdirAll(filepath.Join(s.d.Dir, downloadsDir), 0o700); err != nil {
		return err
	}
	path := filepath.Join(s.d.Dir, downloadsDir, it.ID+filepath.Ext(it.FileName))
	defer func() { _ = os.Remove(path) }()
	if err := s.fetch(ctx, it, links[0].URI, path); err != nil {
		return err
	}
	s.mu.Lock()
	cur := s.find(it.ID)
	if cur == nil || cur.State != StateDownloading {
		s.mu.Unlock()
		return context.Canceled
	}
	cur.State, cur.Progress, cur.Speed = StateInstalling, 100, 0
	s.mu.Unlock()
	s.publish(true)
	_, err = s.d.Install(it.Game, it.Profile, path, profile.Source{
		Kind: profile.KindNexus, Name: it.FileName, ModID: it.ModID, FileID: it.FileID, Version: it.Version,
		Picture: mod.PictureURL, EndorsementCount: mod.EndorsementCount,
	})
	return s.finish(it.ID, err, false)
}

// finish marks an item done after its install, which a mod already in the profile does not fail.
func (s *Service) finish(id string, err error, unverified bool) error {
	var dup *profile.DuplicateError
	if err != nil && !errors.As(err, &dup) {
		return err
	}
	s.mu.Lock()
	if cur := s.find(id); cur != nil {
		cur.State, cur.Progress, cur.key, cur.staged, cur.Unverified = StateDone, 100, "", "", unverified
	}
	s.mu.Unlock()
	s.publish(true)
	return nil
}

// fetch streams the file to path, reporting progress about every progressEvery.
func (s *Service) fetch(ctx context.Context, it Item, url, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := s.d.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the download server answered %s", resp.Status)
	}
	const limit = archive.DefaultMaxTotalBytes
	tooLarge := fmt.Errorf("the download is larger than %d MiB", limit>>20)
	if resp.ContentLength > limit {
		return tooLarge
	}
	total := resp.ContentLength
	if total <= 0 {
		total = it.SizeKB << 10
	}
	f, err := fsx.Create(path)
	if err != nil {
		return s.diskError(err, total)
	}
	defer func() { _ = f.Close() }()
	p := &progress{s: s, id: it.ID, total: total, last: s.d.Now()}
	n, err := io.Copy(f, io.TeeReader(io.LimitReader(resp.Body, limit+1), p))
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return s.diskError(err, total)
	}
	if n > limit {
		return tooLarge
	}
	return s.diskError(f.Close(), total)
}

func (s *Service) diskError(err error, total int64) error {
	if err != nil && store.IsDiskFull(err) {
		return &store.DiskFullError{NeedMB: total>>20 + 1, Err: err}
	}
	return err
}

type progress struct {
	s     *Service
	id    string
	total int64
	n     int64
	from  int64
	last  time.Time
}

func (p *progress) Write(b []byte) (int, error) {
	p.set(p.n + int64(len(b)))
	return len(b), nil
}

// set records n bytes received in all.
func (p *progress) set(n int64) {
	p.n = n
	now := p.s.d.Now()
	if now.Sub(p.last) < progressEvery {
		return
	}
	speed := int64(float64(p.n-p.from) / now.Sub(p.last).Seconds())
	p.from, p.last = p.n, now
	p.s.mu.Lock()
	if it := p.s.find(p.id); it != nil && it.State == StateDownloading {
		it.Speed = speed
		if p.total > 0 {
			it.Progress = float64(p.n) * 100 / float64(p.total)
		}
	}
	p.s.mu.Unlock()
	p.s.publish(false)
}
