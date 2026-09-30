// Package queue downloads Nexus files and GitHub release assets one at a time and installs each into its profile.
// A premium account downloads straight away; a free one is walked through Nexus's own download page, one click per
// file, and the nxm:// link that click produces supplies the download key. GitHub needs no account.
package queue

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/github"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/nxm"
	"github.com/Rethunk-AI/mortar/internal/nxmsvc"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/store"
)

// ChangedEvent is emitted with the new State after every change; progress ticks come at most every progressEvery.
const ChangedEvent = "queue:changed"

// What an item is for.
const (
	KindInstall    = "install"
	KindUpdate     = "update"
	KindDependency = "dependency"
)

// Where an item stands. Done, Skipped and Cancelled are final; Failed waits for Retry or Skip; NeedsChoice waits
// for Choose (a release with several archives) and NeedsConfirm for Confirm or Skip (a download SMAPI's update API
// does not tie to its repository).
const (
	StateNeedsChoice  = "needs-choice"
	StateNeedsConfirm = "needs-confirm"
	StateWaitingClick = "waiting-click"
	StateQueued       = "queued"
	StateDownloading  = "downloading"
	StateInstalling   = "installing"
	StateDone         = "done"
	StateFailed       = "failed"
	StateSkipped      = "skipped"
	StateCancelled    = "cancelled"
)

// repoPattern is a GitHub "owner/repo"; it also keeps anything but a name out of the API URL.
var repoPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

// validRepo also refuses "." and "..", which the pattern admits but would move the API URL's path.
func validRepo(repo string) bool {
	owner, name, _ := strings.Cut(repo, "/")
	return repoPattern.MatchString(repo) && strings.Trim(owner, ".") != "" && strings.Trim(name, ".") != ""
}

const (
	fileName        = "queue.json"
	downloadsDir    = "downloads"
	keptFinished    = 100
	progressEvery   = 250 * time.Millisecond
	defaultBackoff  = time.Minute
	keyValidMargin  = 5 * time.Second
	pageURLTemplate = "https://www.nexusmods.com/%s/mods/%d?tab=files&file_id=%d&nmm=1"
)

// Item is one file to download and install. FileID is 0 until the file is chosen from the mod's files (an update
// wants the file at a version; Current is the file the profile has now). A GitHub item has Repo ("owner/repo")
// instead of ModID; its Tag and Asset are filled in from the release, and Assets lists the choices while it
// waits in StateNeedsChoice. Unverified marks an installed asset whose source SMAPI's API could not check.
// Progress is a percentage and Speed bytes per second.
type Item struct {
	ID       string  `json:"id"`
	Kind     string  `json:"kind"`
	Game     string  `json:"game"`
	Profile  string  `json:"profileId"`
	ModID    int     `json:"modId"`
	FileID   int     `json:"fileId"`
	Current  int     `json:"currentFileId"`
	Name     string  `json:"name"`
	FileName string  `json:"fileName"`
	Version  string  `json:"version"`
	SizeKB   int64   `json:"sizeKb"`
	State    string  `json:"state"`
	Progress float64 `json:"progress"`
	Speed    int64   `json:"speed"`
	Error    string  `json:"error"`

	Repo       string   `json:"repo"`
	Tag        string   `json:"tag"`
	Asset      string   `json:"asset"`
	Assets     []string `json:"assets"`
	Unverified bool     `json:"unverified"`

	// The key and expiry of an nxm:// link supply a free account's download; they are never written to disk.
	key     string
	expires int64
	// staged is the store key of a downloaded GitHub asset that waits for Confirm, or that Confirm released.
	staged string
}

// saved is queue.json: the state, and the staged key of each item that has one, so a restart still asks for
// Confirm instead of downloading again. Staged keys stay out of State, which the window receives.
type saved struct {
	State
	Staged map[string]string `json:"staged,omitempty"`
}

// State is the whole queue. LimitedUntil is the Unix time Nexus's rate limit lifts, 0 when it is not limiting.
type State struct {
	Items        []Item `json:"items"`
	Paused       bool   `json:"paused"`
	LimitedUntil int64  `json:"limitedUntil"`
}

// Request asks for one file. FileID may be 0 to have Mortar choose by Version. CurrentKey is the store key of the
// entry an update replaces. A GitHub request names Repo instead of ModID, and Tag or Version selects the release
// (the newest stable one when both are empty); Asset may be empty when the release has just one archive.
type Request struct {
	Kind       string `json:"kind"`
	Game       string `json:"game"`
	Profile    string `json:"profileId"`
	ModID      int    `json:"modId"`
	FileID     int    `json:"fileId"`
	Name       string `json:"name"`
	FileName   string `json:"fileName"`
	Version    string `json:"version"`
	CurrentKey string `json:"currentKey"`
	Repo       string `json:"repo"`
	Tag        string `json:"tag"`
	Asset      string `json:"asset"`

	key     string
	expires int64
}

// Deps is what the queue needs from the rest of the app.
type Deps struct {
	// Client returns a client that carries the signed-in account's key, or an error when signed out.
	Client  func() (*nexus.Client, error)
	Premium func() bool
	Install func(game, profileID, path string, source profile.Source) (profile.InstallResult, error)
	// Stage unpacks a downloaded GitHub asset into the store and returns the UniqueIDs of its mods; InstallStaged
	// then adds it to the profile. Between the two, Verify checks the source.
	Stage         func(game string, source profile.Source, path string) (key string, uniqueIDs []string, err error)
	InstallStaged func(game, profileID, key string, source profile.Source) (profile.InstallResult, error)
	Verify        func(ctx context.Context, uniqueID, owner, repo string) (bool, error)
	GitHub        *github.Client
	OpenURL       func(url string) error
	// Running reports whether the game runs the profile; its items wait until it stops. Nil means never.
	Running func(game, profileID string) bool
	// Emit is nil in tests that do not watch events.
	Emit func(name string, data any)
	// Changed is called with the state after every change, from the goroutine that made it; nil means nothing.
	Changed func(State)
	HTTP    *http.Client
	// Dir is the data folder holding queue.json and the downloads folder; it must be on the store's volume.
	Dir string
	Now func() time.Time
}

// Service is the download queue.
type Service struct {
	d    Deps
	kick chan struct{}
	// pub orders publishes, so queue.json and the window always end on the latest state.
	pub   sync.Mutex
	mu    sync.Mutex
	items []*Item
	// paused stops new downloads from starting; one under way finishes.
	paused  bool
	until   time.Time
	cancels map[string]context.CancelFunc
}

// New reads the saved queue: what was under way starts over, and a file waiting for a click is clicked again.
func New(d Deps) (*Service, error) {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.HTTP == nil {
		d.HTTP = &http.Client{}
	}
	s := &Service{d: d, kick: make(chan struct{}, 1), cancels: map[string]context.CancelFunc{}}
	b, err := os.ReadFile(filepath.Join(d.Dir, fileName))
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var file saved
	if err := json.Unmarshal(b, &file); err != nil {
		return nil, fmt.Errorf("read the download queue: %w", err)
	}
	s.paused = file.Paused
	for _, it := range file.Items {
		if it.staged = file.Staged[it.ID]; it.State == StateNeedsConfirm && it.staged == "" {
			it.State = StateQueued
		}
		if slices.Contains([]string{StateDownloading, StateInstalling, StateWaitingClick}, it.State) {
			it.State, it.Progress, it.Speed = StateQueued, 0, 0
		}
		s.items = append(s.items, &it)
	}
	return s, nil
}

// Run works the queue until ctx ends, and feeds it the links the user assigned to a profile.
func Run(ctx context.Context, s *Service, assigned <-chan nxmsvc.Assignment) {
	_ = os.RemoveAll(filepath.Join(s.d.Dir, downloadsDir))
	go s.run(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case a := <-assigned:
				r := Request{
					Kind: KindInstall, Game: a.Game, Profile: a.Profile, ModID: a.Link.ModID, FileID: a.Link.FileID,
					key: a.Link.Key, expires: a.Link.Expires,
				}
				if _, err := s.add([]Request{r}); err != nil {
					s.reject(r, err)
				}
			}
		}
	}()
}

// reject shows a link that could not be queued as a failed entry, since there is no item to blame. It keeps the
// request so that Retry can download it once the cause is fixed.
func (s *Service) reject(r Request, err error) {
	s.mu.Lock()
	s.items = append(s.items, &Item{
		ID: newID(), Kind: r.Kind, Game: r.Game, Profile: r.Profile, ModID: r.ModID, FileID: r.FileID,
		State: StateFailed, Error: err.Error(), key: r.key, expires: r.expires,
	})
	s.mu.Unlock()
	s.publish(true)
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Service) poke() {
	select {
	case s.kick <- struct{}{}:
	default:
	}
}

func (s *Service) snapshot() State {
	st := State{Items: make([]Item, len(s.items)), Paused: s.paused}
	for i, it := range s.items {
		st.Items[i] = *it
	}
	if s.until.After(s.d.Now()) {
		st.LimitedUntil = s.until.Unix()
	}
	return st
}

// State returns the queue as it stands.
func (s *Service) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshot()
}

func finished(state string) bool {
	return state == StateDone || state == StateSkipped || state == StateCancelled
}

// publish tells the window; persist also writes queue.json, which progress ticks skip.
func (s *Service) publish(persist bool) {
	s.pub.Lock()
	defer s.pub.Unlock()
	s.mu.Lock()
	if n := len(s.items); n > 0 {
		var kept []*Item
		over := 0
		for _, it := range s.items {
			if finished(it.State) {
				over++
			}
		}
		over -= keptFinished
		for _, it := range s.items {
			if over > 0 && finished(it.State) {
				over--
				continue
			}
			kept = append(kept, it)
		}
		s.items = kept
	}
	st := s.snapshot()
	s.mu.Unlock()
	if persist {
		// A queue that cannot be saved still runs; it only starts over after a restart.
		out := saved{State: st, Staged: map[string]string{}}
		for _, it := range st.Items {
			if it.staged != "" {
				out.Staged[it.ID] = it.staged
			}
		}
		_ = datadir.WriteJSON(filepath.Join(s.d.Dir, fileName), out)
	}
	if s.d.Emit != nil {
		s.d.Emit(ChangedEvent, st)
	}
	if s.d.Changed != nil {
		s.d.Changed(st)
	}
}

func (s *Service) find(id string) *Item {
	for _, it := range s.items {
		if it.ID == id {
			return it
		}
	}
	return nil
}

// Add queues files. It refuses a Nexus file while signed out, since nothing could be downloaded. A file already
// waiting in the same profile is not queued twice, and a failed one is retried.
func (s *Service) Add(reqs []Request) ([]Item, error) {
	for i := range reqs {
		reqs[i].key, reqs[i].expires = "", 0
	}
	return s.add(reqs)
}

func (s *Service) add(reqs []Request) ([]Item, error) {
	if slices.ContainsFunc(reqs, func(r Request) bool { return r.Repo == "" }) {
		if _, err := s.d.Client(); err != nil {
			return nil, err
		}
	}
	out := make([]Item, 0, len(reqs))
	s.mu.Lock()
	for _, r := range reqs {
		if r.Game == "" || r.Profile == "" || (r.ModID <= 0 && !validRepo(r.Repo)) {
			s.mu.Unlock()
			return nil, errors.New("choose a mod and a profile for the download")
		}
		if i := slices.IndexFunc(s.items, func(it *Item) bool {
			return !finished(it.State) && it.Game == r.Game && it.Profile == r.Profile && it.ModID == r.ModID && it.FileID == r.FileID && it.Repo == r.Repo
		}); i >= 0 {
			it := s.items[i]
			if it.State == StateFailed {
				it.State, it.Error = StateQueued, ""
			}
			if r.key != "" {
				it.key, it.expires = r.key, r.expires
			}
			out = append(out, *it)
			continue
		}
		it := &Item{
			ID: newID(), Kind: r.Kind, Game: r.Game, Profile: r.Profile, ModID: r.ModID, FileID: r.FileID,
			Name: r.Name, FileName: r.FileName, Version: r.Version, State: StateQueued, key: r.key, expires: r.expires,
			Repo: r.Repo, Tag: r.Tag, Asset: r.Asset,
		}
		if it.Repo != "" {
			it.Name = cmp.Or(it.Name, it.Repo)
		}
		if id, file, ok := store.NexusFile(r.CurrentKey); ok && id == r.ModID {
			it.Current = file
		}
		s.items = append(s.items, it)
		out = append(out, *it)
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
	return out, nil
}

// Route gives a link's download key to the waiting item for the same file. It reports whether one took it; when
// none did, the link is a download the user started on their own.
func (s *Service) Route(link nxm.Link) bool {
	s.mu.Lock()
	i := slices.IndexFunc(s.items, func(it *Item) bool {
		return (it.State == StateWaitingClick || it.State == StateQueued) && it.ModID == link.ModID && it.FileID == link.FileID && it.FileID != 0
	})
	if i < 0 {
		s.mu.Unlock()
		return false
	}
	it := s.items[i]
	it.key, it.expires, it.State = link.Key, link.Expires, StateQueued
	s.mu.Unlock()
	s.publish(true)
	s.poke()
	return true
}

// StagedKeys lists, per game, the store keys of downloads staged for Confirm, which the store must keep.
func (s *Service) StagedKeys() map[string][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string][]string{}
	for _, it := range s.items {
		if it.staged != "" {
			out[it.Game] = append(out[it.Game], it.staged)
		}
	}
	return out
}

// Retry queues a failed item again.
func (s *Service) Retry(id string) { s.retry(func(it *Item) bool { return it.ID == id }) }

// RetryFailed queues every failed item again.
func (s *Service) RetryFailed() { s.retry(func(*Item) bool { return true }) }

func (s *Service) retry(match func(*Item) bool) {
	s.mu.Lock()
	for _, it := range s.items {
		if it.State == StateFailed && match(it) {
			it.State, it.Error = StateQueued, ""
		}
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}

// Skip drops an item that has not started, that failed, or that waits for the user.
func (s *Service) Skip(id string) {
	s.end(id, StateSkipped, StateFailed, StateQueued, StateWaitingClick, StateNeedsChoice, StateNeedsConfirm)
}

// Choose picks the asset of an item waiting in StateNeedsChoice.
func (s *Service) Choose(id, asset string) {
	s.mu.Lock()
	if it := s.find(id); it != nil && it.State == StateNeedsChoice && slices.Contains(it.Assets, asset) {
		it.State, it.Asset, it.Assets = StateQueued, asset, nil
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}

// Confirm installs an item waiting in StateNeedsConfirm although its source could not be tied to its repository.
func (s *Service) Confirm(id string) {
	s.mu.Lock()
	if it := s.find(id); it != nil && it.State == StateNeedsConfirm {
		it.State = StateQueued
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}

// Cancel stops an item, including a download under way. An install cannot be interrupted.
func (s *Service) Cancel(id string) {
	s.end(id, StateCancelled, StateQueued, StateWaitingClick, StateDownloading)
}

func (s *Service) end(id, to string, from ...string) {
	s.mu.Lock()
	if it := s.find(id); it != nil && slices.Contains(from, it.State) {
		it.State, it.Progress, it.Speed, it.key = to, 0, 0, ""
		if cancel := s.cancels[id]; cancel != nil {
			cancel()
		}
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}

// Pause stops new downloads from starting, and Resume lets them go on.
func (s *Service) Pause() { s.setPaused(true) }

func (s *Service) Resume() { s.setPaused(false) }

func (s *Service) setPaused(p bool) {
	s.mu.Lock()
	s.paused = p
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}

// OpenPage opens the Nexus download page of a waiting item again.
func (s *Service) OpenPage(id string) error {
	s.mu.Lock()
	it := s.find(id)
	var url string
	if it != nil && it.FileID != 0 {
		url = fmt.Sprintf(pageURLTemplate, nexus.Game, it.ModID, it.FileID)
	}
	s.mu.Unlock()
	if url == "" {
		return errors.New("this download has no page to open yet")
	}
	return s.d.OpenURL(url)
}
