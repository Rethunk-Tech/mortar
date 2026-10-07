// Package storecheck verifies that stored mod files are intact: a slow background pass, an on-demand check from
// Settings, and Repair, which fetches a damaged item again from where it came from.
package storecheck

import (
	"cmp"
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/dlwatch"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// ProgressEvent carries Progress while a check from Settings runs.
const ProgressEvent = "storecheck:progress"

const (
	startDelay = 3 * time.Minute
	passEvery  = 6 * time.Hour
	busyWait   = 30 * time.Second
	// The background pass sleeps a multiple of each item's check time, so it uses about a fifth of one core.
	restFactor = 4
	minRest    = time.Second
)

// Deps is what the checks need from the rest of the app.
type Deps struct {
	Items *store.Store
	// Source is the source a profile recorded for a store key; zero when none did.
	Source func(game, key string) profile.Source
	// Add queues downloads; Repair uses it to fetch a damaged Nexus or GitHub item again.
	Add func(context.Context, []queue.Request) ([]queue.Item, error)
	// ArchiveDir is the downloads folder, where a local item's archive may still be.
	ArchiveDir func() string
	// NexusMD5 is Nexus's recorded MD5 of a mod file; nil, or an error, leaves an item without hashes to be baselined
	// from its current files.
	NexusMD5 func(ctx context.Context, gameID string, modID, fileID int) (string, error)
	// Busy reports a running game or an active download or install; the background pass waits it out.
	Busy func() bool
	Emit func(name string, data any)
}

// Service is the Settings › Storage integrity backend.
type Service struct {
	d  Deps
	mu sync.Mutex
	// running is the progress of a check from Settings; the background pass stays out of the way meanwhile.
	running  bool
	progress Progress
}

func New(d Deps) *Service { return &Service{d: d} }

// Progress is the state of a check from Settings.
type Progress struct {
	Running bool `json:"running"`
	Done    int  `json:"done"`
	Total   int  `json:"total"`
	Damaged int  `json:"damaged"`
}

// DamagedItem is a store item with damaged files.
type DamagedItem struct {
	Game    string `json:"game"`
	Key     string `json:"key"`
	Name    string `json:"name"`
	Missing int    `json:"missing"`
	Changed int    `json:"changed"`
	Extra   int    `json:"extra"`
}

// Summary is the outcome of a check from Settings. Failed counts items that could not be read.
type Summary struct {
	Checked int           `json:"checked"`
	Failed  int           `json:"failed"`
	Damaged []DamagedItem `json:"damaged"`
}

// CheckProgress is the check from Settings that is running, if any, so a reopened page can pick it up.
func (s *Service) CheckProgress() Progress {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress
}

func (s *Service) setProgress(p Progress) {
	s.mu.Lock()
	s.progress = p
	s.mu.Unlock()
	if s.d.Emit != nil {
		s.d.Emit(ProgressEvent, p)
	}
}

// Check verifies every store item now, whatever its last verification.
func (s *Service) Check(ctx context.Context) (Summary, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return Summary{}, usererr.New(usererr.Busy, "a check is already running")
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	refs, err := s.d.Items.Refs(false, time.Now())
	if err != nil {
		return Summary{}, err
	}
	sum := Summary{Damaged: []DamagedItem{}}
	p := Progress{Running: true, Total: len(refs)}
	s.setProgress(p)
	for _, r := range refs {
		s.baseline(ctx, r)
		d, err := s.d.Items.Verify(ctx, r.Game, r.Key)
		switch {
		case ctx.Err() != nil:
			s.setProgress(Progress{})
			return sum, ctx.Err()
		case err != nil:
			sum.Failed++
			log.Printf("store check %s/%s: %v", r.Game, r.Key, err)
		case !d.Empty():
			sum.Damaged = append(sum.Damaged, DamagedItem{
				Game: r.Game, Key: r.Key, Name: cmp.Or(s.d.Source(r.Game, r.Key).Name, r.Key),
				Missing: len(d.Missing), Changed: len(d.Changed), Extra: len(d.Extra),
			})
		}
		sum.Checked++
		p.Done, p.Damaged = sum.Checked, len(sum.Damaged)
		s.setProgress(p)
	}
	s.setProgress(Progress{})
	return sum, nil
}

// Run verifies due items one at a time until ctx ends, waiting while the game or a download is active and resting
// between items. A check from Settings pauses it.
func Run(ctx context.Context, s *Service) {
	if !sleep(ctx, startDelay) {
		return
	}
	for {
		s.pass(ctx)
		if !sleep(ctx, passEvery) {
			return
		}
	}
}

func (s *Service) pass(ctx context.Context) {
	refs, err := s.d.Items.Refs(true, time.Now())
	if err != nil {
		log.Printf("store verify: %v", err)
		return
	}
	for _, r := range refs {
		for s.busy() {
			if !sleep(ctx, busyWait) {
				return
			}
		}
		start := time.Now()
		s.baseline(ctx, r)
		d, err := s.d.Items.Verify(ctx, r.Game, r.Key)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("store verify %s/%s: %v", r.Game, r.Key, err)
		} else if !d.Empty() {
			log.Printf("store verify %s/%s: %d missing, %d changed, %d extra files", r.Game, r.Key, len(d.Missing), len(d.Changed), len(d.Extra))
		}
		if !sleep(ctx, max(minRest, restFactor*time.Since(start))) {
			return
		}
	}
}

// baseline gives a Nexus item that has no recorded hashes a baseline from its original archive, when that archive is
// still in the downloads folder and its MD5 is the one Nexus lists for the file. Otherwise Verify baselines the item
// from its current files, which cannot see damage done before.
func (s *Service) baseline(ctx context.Context, r store.Ref) {
	modID, fileID, ok := store.NexusFile(r.Key)
	if !ok || s.d.NexusMD5 == nil || s.d.ArchiveDir == nil || s.d.Items.HasBaseline(r.Game, r.Key) {
		return
	}
	want, err := s.d.NexusMD5(ctx, r.Game, modID, fileID)
	if err != nil || want == "" {
		return
	}
	path, ok := s.findNexusArchive(modID, want)
	if !ok {
		return
	}
	if err := s.d.Items.BaselineFromArchive(ctx, r.Game, r.Key, path); err != nil {
		log.Printf("store check %s/%s: baseline from %s: %v", r.Game, r.Key, path, err)
	}
}

// findNexusArchive looks in the downloads folder for an archive of the mod whose MD5 is want.
func (s *Service) findNexusArchive(modID int, want string) (string, bool) {
	dir := s.d.ArchiveDir()
	ents, err := os.ReadDir(dir)
	if dir == "" || err != nil {
		return "", false
	}
	want = strings.ToLower(strings.TrimSpace(want))
	for _, e := range ents {
		if info, ok := dlwatch.ParseNexusFilename(e.Name()); !ok || info.ModID != modID {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if sum, err := fsx.MD5(path); err == nil && sum == want {
			return path, true
		}
	}
	return "", false
}

func (s *Service) busy() bool {
	s.mu.Lock()
	manual := s.running
	s.mu.Unlock()
	return manual || (s.d.Busy != nil && s.d.Busy())
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// RepairResult says what Repair did: "queued" when the item is downloading again, "restored" when it was
// extracted again from its archive.
type RepairResult struct {
	Status string `json:"status"`
}

// Repair replaces a damaged item with a fresh copy from its recorded source. A Nexus or GitHub item goes through
// the download queue for profileID; a local item is extracted again from its archive if that is still in the
// downloads folder. The damaged copy is set aside meanwhile and put back if the repair cannot start.
func (s *Service) Repair(ctx context.Context, gameID, profileID, key string) (RepairResult, error) {
	switch {
	case isNexus(key):
		return s.refetch(ctx, gameID, key, nexusRequest(gameID, profileID, key, s.d.Source(gameID, key)))
	case strings.HasPrefix(key, "github-"):
		src := s.d.Source(gameID, key)
		if src.Repo == "" {
			return RepairResult{}, usererr.New(usererr.NotFound, "Mortar no longer knows which GitHub release this came from, so it cannot download it again.")
		}
		return s.refetch(ctx, gameID, key, queue.Request{
			Kind: queue.KindInstall, Game: gameID, Profile: profileID, Repo: src.Repo, Tag: src.Tag, Asset: src.Asset,
			Name: src.Repo, FileName: src.Asset, Version: src.Version,
		})
	case strings.HasPrefix(key, "local-"):
		return s.reextract(ctx, gameID, key)
	}
	return RepairResult{}, usererr.New(usererr.Invalid, "This is part of the mod loader. Reinstall the loader from the game's page to repair it.")
}

func isNexus(key string) bool {
	_, _, ok := store.NexusFile(key)
	return ok
}

func nexusRequest(gameID, profileID, key string, src profile.Source) queue.Request {
	modID, fileID, _ := store.NexusFile(key)
	return queue.Request{
		Kind: queue.KindInstall, Game: gameID, Profile: profileID, ModID: modID, FileID: fileID,
		Name: src.Name, FileName: src.Name, Version: src.Version,
	}
}

func (s *Service) refetch(ctx context.Context, gameID, key string, req queue.Request) (RepairResult, error) {
	restore, err := s.d.Items.Quarantine(gameID, key)
	if err != nil {
		return RepairResult{}, err
	}
	if _, err := s.d.Add(ctx, []queue.Request{req}); err != nil {
		return RepairResult{}, joinRestore(err, restore)
	}
	return RepairResult{Status: "queued"}, nil
}

func (s *Service) reextract(ctx context.Context, gameID, key string) (RepairResult, error) {
	path, ok := s.findArchive(key)
	if !ok {
		return RepairResult{}, usererr.New(usererr.NotFound, "The archive this mod was installed from is no longer in the downloads folder, so it cannot be repaired. Download it again and reinstall it.")
	}
	restore, err := s.d.Items.Quarantine(gameID, key)
	if err != nil {
		return RepairResult{}, err
	}
	if err := s.d.Items.AddArchiveKey(ctx, gameID, key, path); err != nil {
		return RepairResult{}, joinRestore(err, restore)
	}
	return RepairResult{Status: "restored"}, nil
}

func joinRestore(err error, restore func() error) error {
	if rerr := restore(); rerr != nil {
		log.Printf("store repair: put the damaged item back: %v", rerr)
	}
	return err
}

// findArchive looks in the downloads folder for the archive whose hash is the local key.
func (s *Service) findArchive(key string) (string, bool) {
	if s.d.ArchiveDir == nil || s.d.ArchiveDir() == "" {
		return "", false
	}
	dir := s.d.ArchiveDir()
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, e := range ents {
		if !archive.HasExtension(e.Name()) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if sum, err := fsx.SHA256(path); err == nil && store.LocalKey(sum) == key {
			return path, true
		}
	}
	return "", false
}
