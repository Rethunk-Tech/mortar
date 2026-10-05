package queue

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
func (s *Service) next(running map[string]bool) (it *Item, act action, held bool) {
	if s.paused || s.until.After(s.d.Now()) {
		return nil, resolve, false
	}
	pauseFetch := s.pauseFetch()
	premium := s.d.Premium()
	var head *Item
	waiting := false
	for _, it := range s.items {
		switch {
		case it.State == StateWaitingClick:
			waiting = true
		case it.State != StateQueued:
		case it.overlay && s.waitsForSameMod(it):
		case running[it.Game+"\n"+it.Profile]:
			held = true
		case it.Package != "":
			if pauseFetch {
				held = true
				continue
			}
			return it, fetch, held
		case it.Repo != "":
			act := s.forAsset(it)
			if act == fetch && pauseFetch {
				held = true
				continue
			}
			return it, act, held
		case premium || s.usable(it) || s.stored(it):
			act := s.forFile(it, fetch)
			if act == fetch && pauseFetch {
				held = true
				continue
			}
			return it, act, held
		case head == nil:
			head = it
		}
	}
	if head == nil || waiting {
		return nil, resolve, held
	}
	return head, s.forFile(head, click), held
}

func retryBackoff(n int) time.Duration {
	d := 200 * time.Millisecond
	for range n {
		d *= 2
	}
	return d
}

func (s *Service) checkNexusMD5(path, want string) error {
	if s.d.VerifyNexusMD5 == nil || !s.d.VerifyNexusMD5() {
		return nil
	}
	want = strings.ToLower(strings.TrimSpace(want))
	if want == "" {
		return nil
	}
	got, err := fsx.MD5(path)
	if err != nil {
		return err
	}
	if got != want {
		return usererr.New(usererr.Damaged, fmt.Sprintf("Nexus MD5 mismatch: got %s, wanted %s", got, want))
	}
	return nil
}

func (s *Service) pauseFetch() bool {
	if s.d.PauseWhilePlaying == nil || !s.d.PauseWhilePlaying() {
		return false
	}
	return s.d.GameBusy != nil && s.d.GameBusy()
}

func (s *Service) runningOf(items []*Item) map[string]bool {
	out := map[string]bool{}
	if s.d.Running == nil {
		return out
	}
	seen := map[string]struct{}{}
	for _, it := range items {
		k := it.Game + "\n" + it.Profile
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out[k] = s.d.Running(it.Game, it.Profile)
	}
	return out
}

// forFile is resolve while the item's file is not known yet.
func (s *Service) forFile(it *Item, then action) action {
	if it.staged != "" || it.readyZip {
		return install
	}
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

// heldRecheck is how often items held for a running profile are looked at again. Launch also republishes the
// queue when a profile unlocks.
const heldRecheck = 5 * time.Second

func (s *Service) run(ctx context.Context) {
	var workers sync.WaitGroup
	for range 3 {
		workers.Go(func() {
			for ctx.Err() == nil {
				if s.step(ctx) {
					continue
				}
				select {
				case <-ctx.Done():
				case <-s.kick:
				case <-time.After(heldRecheck):
				}
			}
		})
	}
	<-ctx.Done()
	workers.Wait()
}

// obsolete lists queued Nexus items the profile already holds at that file or newer, read under the lock. Only an
// update or a Latest request counts: a plain request asks for that exact file, as a share import does.
func (s *Service) obsolete() []Item {
	var out []Item
	for _, it := range s.items {
		if (it.State == StateQueued || it.State == StateWaitingClick) && it.Repo == "" && it.FileID > 0 &&
			it.staged == "" && !it.readyZip && (it.Latest || it.Kind == KindUpdate) {
			out = append(out, *it)
		}
	}
	return out
}

// skipHeld skips the items obsolete found once the profile is checked outside the lock.
func (s *Service) skipHeld() {
	if s.d.Newest == nil {
		return
	}
	s.mu.Lock()
	cands := s.obsolete()
	s.mu.Unlock()
	for _, it := range cands {
		if have := s.d.Newest(it.Game, it.Profile, it.ModID, it.Current); have >= it.FileID {
			log.Printf("queue: mod %d file %d skipped as %s, profile %s already has file %d", it.ModID, it.FileID, it.ID, it.Profile, have)
			s.end(it.ID, StateSkipped, StateQueued, StateWaitingClick)
		}
	}
}

func (s *Service) step(ctx context.Context) bool {
	s.skipHeld()
	s.mu.Lock()
	var queued []*Item
	for _, it := range s.items {
		if it.State == StateQueued || it.State == StateWaitingClick {
			queued = append(queued, it)
		}
	}
	s.mu.Unlock()
	running := s.runningOf(queued)
	s.mu.Lock()
	it, act, _ := s.next(running)
	// Resolving leaves an item queued, so next can hand out one another worker is still resolving; a second
	// request would race the first, and a stale rate-limit answer could pause the queue again after its reset.
	if it == nil || ctx.Err() != nil || s.cancels[it.ID] != nil {
		s.mu.Unlock()
		return false
	}
	snap := *it
	if act == click {
		it.State = StateWaitingClick
		it.key, it.expires = "", 0
	}
	if act == fetch {
		it.State, it.Progress, it.Speed = StateDownloading, 0, 0
		if it.started.IsZero() {
			it.started = s.d.Now()
		}
	}
	if act == install {
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
		if s.useGitHubFallback(itemCtx, snap) {
			return true
		}
		if err := s.OpenPage(snap.ID); err != nil {
			s.settle(snap.ID, err)
		}
	case resolve:
		s.settle(snap.ID, s.resolve(itemCtx, snap))
	case fetch:
		s.settle(snap.ID, s.download(itemCtx, snap))
		s.dropIfEnded(snap)
	case install:
		s.settle(snap.ID, s.installStaged(itemCtx, snap))
	}
	return true
}

// dropIfEnded removes a download that was cancelled or skipped while it ran: end removes the files at once, but the
// download may still write them until it has stopped, so they are removed again once it has.
func (s *Service) dropIfEnded(snap Item) {
	s.mu.Lock()
	it := s.find(snap.ID)
	ended := it != nil && (it.State == StateCancelled || it.State == StateSkipped)
	s.mu.Unlock()
	if ended {
		dropDownload(s.dest(snap.ID, snap.FileName))
	}
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
	var busy *source.BusyError
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
	case errors.As(err, &busy):
		it.State, it.Progress, it.Speed = StateQueued, 0, 0
		s.until = busy.Reset
		if !s.until.After(s.d.Now()) {
			s.until = s.d.Now().Add(defaultBackoff)
		}
	case errors.Is(err, nexus.ErrPremiumRequired):
		it.State, it.Progress, it.Speed, it.key = StateWaitingClick, 0, 0, ""
	case errors.Is(err, nexus.ErrUnauthorized):
		it.State, it.Error = StateFailed, "Nexus rejected your API key: sign in again in Settings"
	case errors.Is(err, nexus.ErrQuarantined):
		it.State, it.Error = StateFailed, "Nexus has quarantined this file; Mortar will not download it"
	case errors.As(err, &full):
		it.State, it.Error = StateFailed, fmt.Sprintf("Not enough disk space: about %d MB is needed", full.NeedMB)
	default:
		it.State, it.Error = StateFailed, err.Error()
	}
	if it.State == StateFailed {
		it.ErrorKind = failureKind(err)
	}
	reopen := it.State == StateWaitingClick
	var rec *Item
	if it.State == StateFailed {
		snap := *it
		rec = &snap
	}
	s.mu.Unlock()
	s.publish(true)
	if rec != nil {
		s.recordHistory(rec, StateFailed)
		notifyDesktopDownload(rec.Name, rec.Game, rec.Profile, false)
	}
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
	t, err := game.NexusTitle(it.Game)
	if err != nil {
		return err
	}
	files, err := c.Files(ctx, t, it.ModID)
	if err != nil {
		return usererr.Wrap(usererr.Network, err)
	}
	statuses, _ := c.ScanStatuses(ctx, t, it.ModID)
	var file nexus.File
	if it.FileID != 0 && statuses[it.FileID] == "QUARANTINED" {
		return nexus.ErrQuarantined
	}
	safe := files[:0]
	for _, candidate := range files {
		if statuses[candidate.FileID] != "QUARANTINED" {
			safe = append(safe, candidate)
		}
	}
	files = safe
	if it.FileID != 0 {
		file = fileByID(files, it.FileID)
		if it.Latest {
			file = newestUpdate(files, file)
		}
	} else {
		file, _ = ChooseFile(files, it.Version, it.Current)
	}
	if file.FileID == 0 {
		return usererr.New(usererr.NotFound, "no suitable file for this mod is listed on Nexus")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if cur := s.find(it.ID); cur != nil {
		cur.FileID, cur.FileName, cur.SizeKB = file.FileID, cmp.Or(file.FileName, fmt.Sprintf("file-%d", file.FileID)), file.SizeKB
		cur.fileMD5 = file.MD5
		cur.Version = cmp.Or(cur.Version, file.Version)
		if it.Latest {
			cur.Version = cmp.Or(file.Version, cur.Version)
		}
		cur.Category = cmp.Or(cur.Category, file.Category)
	}
	return nil
}

func (s *Service) fetchSlot(ctx context.Context, it Item) (func(), error) {
	if it.Repo == "" && it.Package == "" && (s.d.Premium == nil || !s.d.Premium()) {
		select {
		case s.freeFetch <- struct{}{}:
			return func() { <-s.freeFetch }, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	limit := 3
	if s.d.Parallel != nil {
		if n := s.d.Parallel(); n >= 1 {
			limit = n
		}
	}
	for {
		s.mu.Lock()
		if s.premiumInUse < limit {
			s.premiumInUse++
			s.mu.Unlock()
			return func() {
				s.mu.Lock()
				s.premiumInUse--
				s.mu.Unlock()
			}, nil
		}
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (s *Service) download(ctx context.Context, it Item) error {
	release, err := s.fetchSlot(ctx, it)
	if err != nil {
		return err
	}
	defer release()
	releaseSource, err := s.sourceSlot(ctx, it)
	if err != nil {
		return err
	}
	defer releaseSource()
	if it.Package != "" {
		return s.downloadPackage(ctx, it)
	}
	if it.Repo != "" {
		return s.downloadGitHub(ctx, it)
	}
	c, err := s.d.Client()
	if err != nil {
		return err
	}
	t, err := game.NexusTitle(it.Game)
	if err != nil {
		return err
	}
	var im nexus.Mod
	// The picture and name are nice to have: a page that cannot be fetched only leaves the letter tile. describe
	// usually fetched them while the item waited.
	if it.Picture != "" {
		im = nexus.Mod{Name: it.Name, PictureURL: it.Picture, EndorsementCount: it.endorsed}
	} else if m, merr := c.Mod(ctx, t, it.ModID); merr == nil {
		im = m
		s.mu.Lock()
		if cur := s.find(it.ID); cur != nil {
			if cur.Name == "" {
				cur.Name = m.Name
			}
			cur.Picture, cur.endorsed = m.PictureURL, m.EndorsementCount
		}
		s.mu.Unlock()
	}
	if s.stored(&it) {
		it.overlay = it.overlay || (s.d.StoredOverlay != nil && s.d.StoredOverlay(it.Game, store.NexusKey(it.ModID, it.FileID)))
		if s.parkOverlay(it.ID, it.overlay, false) {
			return nil
		}
		if done, err := s.installStored(ctx, c, it, im); done {
			return err
		}
	}
	s.mu.Lock()
	key, expires := "", int64(0)
	if cur := s.find(it.ID); cur != nil && s.usable(cur) {
		key, expires = cur.key, cur.expires
	}
	s.mu.Unlock()
	links, err := c.DownloadLinks(ctx, t, it.ModID, it.FileID, key, expires)
	if err != nil {
		return usererr.Wrap(usererr.Network, err)
	}
	if len(links) == 0 {
		return usererr.New(usererr.Network, "the download has no link")
	}
	if err := os.MkdirAll(s.downloadRoot(), 0o700); err != nil {
		return err
	}
	path := s.dest(it.ID, it.FileName)
	fetchOnce := func() error {
		uri := links[0].URI
		var ferr error
		for attempt := range 2 {
			ferr = s.fetch(ctx, it, uri, path)
			if ferr == nil || !errors.Is(ferr, github.ErrLinkExpired) || attempt == 1 {
				break
			}
			next, lerr := c.DownloadLinks(ctx, t, it.ModID, it.FileID, key, expires)
			if lerr != nil {
				return usererr.Wrap(usererr.Network, lerr)
			}
			if len(next) == 0 {
				return usererr.New(usererr.Network, "the download has no link")
			}
			links = next
			uri = links[0].URI
		}
		return ferr
	}
	retries := 0
	if s.d.RetryFetches != nil {
		retries = s.d.RetryFetches()
	}
	var fetchErr error
	for n := 0; n <= retries; n++ {
		fetchErr = fetchOnce()
		if fetchErr == nil || errors.Is(fetchErr, context.Canceled) {
			break
		}
		if n == retries {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryBackoff(n)):
		}
	}
	if fetchErr != nil {
		return fetchErr
	}
	wantMD5 := ""
	s.mu.Lock()
	if cur := s.find(it.ID); cur != nil {
		wantMD5 = cur.fileMD5
	}
	s.mu.Unlock()
	if err := s.checkNexusMD5(path, wantMD5); err != nil {
		return err
	}
	optional := manifestLess(path)
	if s.parkOverlay(it.ID, optional, true) {
		return nil
	}
	var files []nexus.File
	category := it.Category
	if category == "" {
		files = pageFiles(ctx, c, it)
		category = fileCategoryOf(files, it.FileID)
	}
	var ask profile.MergeAsk
	updates, merge := 0, false
	if it.Kind != KindUpdate && s.d.SamePage != nil && !optional {
		ids := archiveModIDs(path)
		ask, updates, merge = s.d.SamePage(it.Game, it.Profile, it.incoming(category, files, ids))
		if merge && files == nil {
			// Only a file from a page the profile already holds needs the page's file list to tell an update.
			ask, updates, merge = s.d.SamePage(it.Game, it.Profile, it.incoming(category, pageFiles(ctx, c, it), ids))
		}
	}
	s.mu.Lock()
	cur := s.find(it.ID)
	if cur == nil || cur.State != StateDownloading {
		s.mu.Unlock()
		return context.Canceled
	}
	cur.State, cur.Progress, cur.Speed, cur.readyZip, cur.Category = StateInstalling, 100, 0, true, category
	if merge {
		cur.State, cur.Merge, cur.MergeAdd = StateNeedsMerge, &ask, false
		s.mu.Unlock()
		s.publish(true)
		return nil
	}
	if updates > 0 {
		cur.Current, it.Current = updates, updates
	}
	s.mu.Unlock()
	s.publish(true)
	return s.contentPatcherHint(ctx, it, s.installNexusPath(ctx, it, path, im))
}

func (s *Service) pauseFomod(id, key string) {
	s.mu.Lock()
	if cur := s.find(id); cur != nil {
		cur.State, cur.Progress, cur.staged, cur.FomodKey = StateNeedsFomod, 0, key, key
	}
	s.mu.Unlock()
	s.publish(true)
}

func (s *Service) pauseRoot(id string, ask *profile.RemapAsk) {
	if ask == nil {
		return
	}
	s.mu.Lock()
	if cur := s.find(id); cur != nil {
		cur.State, cur.Progress, cur.staged, cur.Remap = StateNeedsRoot, 0, ask.Key, ask
	}
	s.mu.Unlock()
	s.publish(true)
}

func (s *Service) afterInstall(id string, res profile.InstallResult, err error, unverified bool) error {
	if err == nil && res.Fomod != nil {
		s.pauseFomod(id, res.Fomod.Key)
		return nil
	}
	if err == nil && res.Remap != nil {
		s.pauseRoot(id, res.Remap)
		return nil
	}
	return s.finish(id, err, unverified)
}

// finish marks an item done after its install, which a mod already in the profile does not fail.
func (s *Service) finish(id string, err error, unverified bool) error {
	var dup *profile.DuplicateError
	if err != nil && !errors.As(err, &dup) {
		return err
	}
	s.mu.Lock()
	var rec *Item
	if cur := s.find(id); cur != nil {
		snap := *cur
		rec = &snap
		cur.State, cur.Progress, cur.key, cur.staged, cur.Unverified = StateDone, 100, "", "", unverified
	}
	s.mu.Unlock()
	s.publish(true)
	if rec != nil {
		s.recordHistory(rec, StateDone)
		notifyDesktopDownload(rec.Name, rec.Game, rec.Profile, true)
		if s.d.HistoryBatch != nil {
			if batchErr := s.d.HistoryBatch(rec.Game, rec.Profile, rec.BatchID); batchErr != nil {
				log.Printf("queue: record profile history batch %s: %v", rec.BatchID, batchErr)
			}
			if rec.BatchID != "" && s.batchFinished(rec) {
				if batchErr := s.d.HistoryBatch(rec.Game, rec.Profile, ""); batchErr != nil {
					log.Printf("queue: close profile history batch %s: %v", rec.BatchID, batchErr)
				}
			}
		}
	}
	return nil
}

func (s *Service) batchFinished(rec *Item) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.items {
		if it == nil || it.ID == rec.ID || it.Game != rec.Game || it.Profile != rec.Profile || it.BatchID != rec.BatchID {
			continue
		}
		if !dismissable(it.State) {
			return false
		}
	}
	return true
}

func (s *Service) diskError(err error, total int64) error {
	if err != nil && usererr.IsDiskFull(err) {
		return usererr.Wrap(usererr.DiskFull, &store.DiskFullError{NeedMB: total>>20 + 1, Err: err})
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

// set records n bytes received in all.
func (p *progress) set(n int64) {
	p.n = n
	now := p.s.d.Now()
	if now.Sub(p.last) < progressEvery {
		return
	}
	speed := int64(float64(p.n-p.from) / now.Sub(p.last).Seconds())
	p.from, p.last = p.n, now
	var tick Progress
	p.s.mu.Lock()
	it := p.s.find(p.id)
	if it != nil && it.State == StateDownloading {
		it.Speed = speed
		if p.total > 0 {
			it.Progress = float64(p.n) * 100 / float64(p.total)
			// The file list can omit a size; the transfer's own length is the real one.
			it.SizeKB = p.total >> 10
		}
		tick = Progress{ID: it.ID, Progress: it.Progress, Speed: it.Speed, SizeKB: it.SizeKB}
	}
	p.s.mu.Unlock()
	if tick.ID != "" {
		p.s.publishProgress(tick)
	}
}

// installStored adds a Nexus file the store already holds to the item's profile. It reports false, so the file is
// downloaded as usual, when the profile has an entry from the same mod page: that choice installs from the archive.
func (s *Service) installStored(ctx context.Context, c *nexus.Client, it Item, im nexus.Mod) (bool, error) {
	if it.Kind != KindUpdate && !it.overlay && s.d.SamePage != nil {
		_, updates, ok := s.d.SamePage(it.Game, it.Profile, it.incoming(it.Category, nil, nil))
		if ok {
			// Only a file from a page the profile already holds needs the page's file list to tell an update.
			_, updates, ok = s.d.SamePage(it.Game, it.Profile, it.incoming(it.Category, pageFiles(ctx, c, it), nil))
		}
		if updates > 0 {
			s.mu.Lock()
			if cur := s.find(it.ID); cur != nil {
				cur.Current = updates
			}
			s.mu.Unlock()
			it.Current = updates
		}
		if ok {
			s.mu.Lock()
			defer s.mu.Unlock()
			cur := s.find(it.ID)
			if cur == nil || s.d.Premium() || s.usable(cur) {
				return false, nil
			}
			// A free account needs the file's link to download it, so the item waits for its click.
			cur.fromStoreRefused, cur.State = true, StateQueued
			return true, nil
		}
	}
	s.mu.Lock()
	if cur := s.find(it.ID); cur != nil {
		cur.State, cur.Progress, cur.Speed = StateInstalling, 100, 0
	}
	s.mu.Unlock()
	s.publish(true)
	key := store.NexusKey(it.ModID, it.FileID)
	log.Printf("queue: mod %d file %d installs from the store (%s)", it.ModID, it.FileID, key)
	s.installMu.Lock()
	res, err := s.d.InstallStaged(it.Game, it.Profile, key, nexusSource(it, im))
	s.installMu.Unlock()
	return true, s.afterInstall(it.ID, res, err, false)
}
