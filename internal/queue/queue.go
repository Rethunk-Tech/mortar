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
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/desktopnotify"
	"github.com/Rethunk-AI/mortar/internal/fsx"
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
// for Choose (a release with several archives), NeedsConfirm for Confirm or Skip (a download SMAPI's update API
// does not tie to its repository), NeedsFomod for AnswerFomod (a store item with a FOMOD installer), and
// NeedsRoot for AnswerRoot (an extracted archive with no SMAPI manifest until the user picks a folder).
const (
	StateNeedsChoice  = "needs-choice"
	StateNeedsConfirm = "needs-confirm"
	StateNeedsFomod   = "needs-fomod"
	StateNeedsRoot    = "needs-root"
	StateNeedsMerge   = "needs-merge"
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
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	BatchID  string `json:"batchId,omitempty"`
	Game     string `json:"game"`
	Profile  string `json:"profileId"`
	ModID    int    `json:"modId"`
	FileID   int    `json:"fileId"`
	Current  int    `json:"currentFileId"`
	Name     string `json:"name"`
	FileName string `json:"fileName"`
	Version  string `json:"version"`
	SizeKB   int64  `json:"sizeKb"`
	// Picture is the mod page's picture URL, known once the download starts; empty for GitHub items.
	Picture  string  `json:"picture,omitempty"`
	State    string  `json:"state"`
	Progress float64 `json:"progress"`
	// Latest carries Request.Latest so a restart still resolves to the newest file.
	Latest   bool                           `json:"latest,omitempty"`
	Disabled []string                       `json:"disabled,omitempty"`
	Fomod    map[string]map[string][]string `json:"fomod,omitempty"`
	Speed    int64                          `json:"speed"`
	Error    string                         `json:"error"`

	Repo         string            `json:"repo"`
	Tag          string            `json:"tag"`
	Asset        string            `json:"asset"`
	Assets       []string          `json:"assets"`
	FallbackRepo string            `json:"fallbackRepo,omitempty"`
	FallbackID   string            `json:"fallbackId,omitempty"`
	Unverified   bool              `json:"unverified"`
	FomodKey     string            `json:"fomodKey,omitempty"`
	Remap        *profile.RemapAsk `json:"remap,omitempty"`
	Category     string            `json:"category,omitempty"`
	Merge        *profile.MergeAsk `json:"merge,omitempty"`
	MergeAdd     bool              `json:"mergeAdd,omitempty"`

	// fallbackTried stops a Nexus item from asking GitHub again once its fallback was looked up.
	fallbackTried bool
	// nexusFileName is the Nexus file name a GitHub fallback replaced, restored when the release is not this mod.
	nexusFileName string
	// The key and expiry of an nxm:// link supply a free account's download; they are never written to disk.
	key     string
	expires int64
	// staged is the store key of a downloaded GitHub asset that waits for Confirm, or that Confirm released,
	// or of a Nexus install waiting for FOMOD choices, or of a download waiting for a content root.
	staged string
	// readyZip means the Nexus archive is already on disk and the next step is install (or the merge choice).
	readyZip bool
	// fromStoreRefused means the stored copy cannot be used, because the profile asks to merge with an entry from
	// the same mod page, so a free account's item waits for its click; it is not persisted.
	fromStoreRefused bool
	endorsed         int
	fomod            map[string]map[string][]string
	// chosenRoot is the folder AnswerRoot picked for a staged item; it is not persisted.
	chosenRoot string
	// started is when this attempt left the queue for a fetch; it is not persisted.
	started time.Time
	fileMD5 string
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
	BatchID    string `json:"batchId,omitempty"`
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
	// FallbackRepo is the mod's GitHub repo (owner/name) for a Nexus update: when the account would have to click
	// Mod Manager Download, the same version's GitHub release is used instead, if there is exactly one archive.
	FallbackRepo string `json:"fallbackRepo,omitempty"`
	// FallbackID is the UniqueID the GitHub release must contain: a repo SMAPI's metadata names may publish
	// several mods, so a release at the same version is used only when it holds the mod being updated.
	FallbackID string `json:"fallbackId,omitempty"`
	// Latest asks for the newest file that updates FileID: a file found in the mod dataset, or that a save
	// recorded, may have been superseded since. Share imports leave it off to reproduce the shared files.
	Latest   bool                           `json:"latest"`
	Disabled []string                       `json:"disabled,omitempty"`
	Fomod    map[string]map[string][]string `json:"fomod,omitempty"`

	key     string
	expires int64
}

// Deps is what the queue needs from the rest of the app.
type Deps struct {
	// Client returns a client that carries the signed-in account's key, or an error when signed out.
	Client  func() (*nexus.Client, error)
	Premium func() bool
	Install func(game, profileID, path string, source profile.Source) (profile.InstallResult, error)
	// Stored reports whether the game's store already holds key, as a Nexus file installed into another profile
	// does, with the source a profile recorded for it (zero when none does). Such a file installs from the store
	// without downloading or a click, and the recorded source spares the Nexus lookups.
	Stored func(game, key string) (profile.Source, bool)
	// Stage unpacks a downloaded GitHub asset into the store and returns the UniqueIDs of its mods; InstallStaged
	// then adds it to the profile. Between the two, Verify checks the source.
	Stage         func(game string, source profile.Source, path string) (key string, uniqueIDs []string, err error)
	InstallStaged func(game, profileID, key string, source profile.Source) (profile.InstallResult, error)
	InstallRemap  func(game, profileID, key, root string, source profile.Source) (profile.InstallResult, error)
	// Newest is the highest Nexus file id the profile holds from modID's page, 0 when none; nil means never.
	Newest func(game, profileID string, modID int) int
	// SamePage reports an existing profile entry from this Nexus mod page when the incoming file is another file on it.
	SamePage func(game, profileID string, modID, fileID int, category string) (profile.MergeAsk, bool)
	// InstallExtra adds a downloaded Nexus file to an existing same-page entry.
	InstallExtra func(game, profileID, entryKey, path string, source profile.Source) (profile.InstallResult, error)
	Verify       func(ctx context.Context, uniqueID, owner, repo string) (bool, error)
	GitHub       *github.Client
	OpenURL      func(url string) error
	// Running reports whether the game runs the profile; its items wait until it stops. Nil means never.
	Running func(game, profileID string) bool
	// Emit is nil in tests that do not watch events.
	Emit func(name string, data any)
	// Changed is called with the state after every change, from the goroutine that made it; nil means nothing.
	Changed func(State)
	// HistoryBatch records a completed install in the profile's bulk history event; an empty batch closes it.
	HistoryBatch func(game, profileID, batchID string) error
	HTTP         *http.Client
	// Dir is the data folder holding queue.json and the downloads folder; it must be on the store's volume.
	Dir string
	Now func() time.Time
	// Parallel is the Premium/GitHub fetch pool size; nil means 3.
	Parallel func() int
	// KeepArchives leaves downloaded zips after they are extracted; nil means delete.
	KeepArchives func() bool
	// DownloadDir is an absolute folder for archives; empty or nil uses <data>/downloads.
	DownloadDir func() string
	// Track records a Nexus mod as tracked after a successful install; nil means never.
	Track func(ctx context.Context, modID int)
	// RetryFetches is extra fetch attempts after a failed download; nil or 0 is off.
	RetryFetches func() int
	// PauseWhilePlaying pauses new fetches while GameBusy; installs still wait per profile.
	PauseWhilePlaying func() bool
	// GameBusy reports whether any game is launching or running; nil means never.
	GameBusy func() bool
	// VerifyNexusMD5 compares a finished Nexus download with the file's API md5 when set.
	VerifyNexusMD5 func() bool
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
	paused       bool
	until        time.Time
	cancels      map[string]context.CancelFunc
	installMu    sync.Mutex
	premiumFetch chan struct{}
	freeFetch    chan struct{}
	premiumInUse int
}

// New reads the saved queue: what was under way is queued again so a partial file can resume, and a file
// waiting for a click is clicked again.
func New(d Deps) (*Service, error) {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.HTTP == nil {
		tr := &http.Transport{Proxy: http.ProxyFromEnvironment, ResponseHeaderTimeout: 20 * time.Second, TLSHandshakeTimeout: 10 * time.Second}
		if dt, ok := http.DefaultTransport.(*http.Transport); ok {
			tr = dt.Clone()
			tr.ResponseHeaderTimeout = 20 * time.Second
		}
		d.HTTP = &http.Client{Transport: tr, Timeout: 30 * time.Minute}
	}
	s := &Service{
		d: d, kick: make(chan struct{}, 1), cancels: map[string]context.CancelFunc{},
		premiumFetch: make(chan struct{}, 3), freeFetch: make(chan struct{}, 1),
	}
	b, err := fsx.ReadFile(filepath.Join(d.Dir, fileName))
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
		if it.State == StateNeedsFomod {
			if it.staged == "" {
				it.State = StateQueued
			} else {
				it.FomodKey = it.staged
			}
		}
		if it.State == StateNeedsRoot && it.staged == "" {
			it.State = StateQueued
		}
		if it.State == StateNeedsMerge {
			it.readyZip = true
		}
		if slices.Contains([]string{StateDownloading, StateInstalling, StateWaitingClick}, it.State) {
			it.State, it.Progress, it.Speed = StateQueued, 0, 0
		}
		s.items = append(s.items, &it)
	}
	return s, nil
}

// Run works the queue until ctx ends, and feeds it the links the user assigned to a profile. The returned wait
// blocks until both workers have stopped after ctx ends, so nothing writes the queue's files after it returns.
func Run(ctx context.Context, s *Service, assigned <-chan nxmsvc.Assignment) (wait func()) {
	s.sweepDownloads()
	var wg sync.WaitGroup
	wg.Go(func() { s.run(ctx) })
	wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case a := <-assigned:
				r := Request{
					Kind: KindInstall, Game: a.Game, Profile: a.Profile, ModID: a.Link.ModID, FileID: a.Link.FileID,
					key: a.Link.Key, expires: a.Link.Expires,
				}
				items, err := s.add([]Request{r})
				if err != nil {
					s.reject(r, err)
				}
				for _, it := range items {
					if it.Name == "" {
						go s.describe(ctx, it.ID, it.ModID)
					}
				}
			}
		}
	})
	return wg.Wait
}

// reject shows a link that could not be queued as a failed entry, since there is no item to blame. It keeps the
// request so that Retry can download it once the cause is fixed.
func (s *Service) reject(r Request, err error) {
	log.Printf("queue: mod %d file %d could not be queued: %v", r.ModID, r.FileID, err)
	it := &Item{
		ID: newID(), Kind: r.Kind, Game: r.Game, Profile: r.Profile, ModID: r.ModID, FileID: r.FileID,
		State: StateFailed, Error: err.Error(), key: r.key, expires: r.expires,
	}
	s.mu.Lock()
	s.items = append(s.items, it)
	name, game, profile := it.Name, it.Game, it.Profile
	s.mu.Unlock()
	s.publish(true)
	notifyDesktopDownload(name, game, profile, false)
}

// describe fills a queued Nexus item's name and picture from its mod page, so it shows them while it waits.
func (s *Service) describe(ctx context.Context, id string, modID int) {
	c, err := s.d.Client()
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	m, err := c.Mod(ctx, modID)
	if err != nil {
		return
	}
	s.mu.Lock()
	cur := s.find(id)
	if cur != nil {
		cur.Name = cmp.Or(cur.Name, m.Name)
		cur.Picture, cur.endorsed = m.PictureURL, m.EndorsementCount
	}
	s.mu.Unlock()
	if cur != nil {
		s.publish(true)
	}
}

// stored reports whether the item's Nexus file is already in the store.
func (s *Service) stored(it *Item) bool {
	return it.Repo == "" && it.FileID != 0 && !it.fromStoreRefused && s.d.Stored != nil && storedOK(s.d.Stored(it.Game, store.NexusKey(it.ModID, it.FileID)))
}

func storedOK(_ profile.Source, ok bool) bool { return ok }

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

// Busy reports whether the queue still has work or an open user decision.
func (s *Service) Busy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.ContainsFunc(s.items, func(it *Item) bool { return !finished(it.State) })
}

func finished(state string) bool {
	return state == StateDone || state == StateSkipped || state == StateCancelled
}

// sameDownload is an in-flight queue item for the same file. GitHub items match on repo, tag and asset; an
// unresolved empty asset is not the same as a chosen one.
func sameDownload(it *Item, r Request) bool {
	if finished(it.State) || it.Game != r.Game || it.Profile != r.Profile {
		return false
	}
	if it.Repo != "" || r.Repo != "" {
		return it.Repo == r.Repo && it.Tag == r.Tag && it.Asset == r.Asset
	}
	return it.ModID == r.ModID && it.FileID == r.FileID
}

// NotifyUnlocked republishes the queue after a profile is no longer running, so downloads and pending share
// configs that were waiting can continue.
func NotifyUnlocked(s *Service) {
	s.publish(false)
	s.poke()
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
	for _, r := range reqs {
		if r.Game == "" || r.Profile == "" || (r.ModID <= 0 && !validRepo(r.Repo)) {
			return nil, errors.New("choose a mod and a profile for the download")
		}
	}
	// A file already in the store takes its name, version and picture from the profile that installed it, so it
	// needs no Nexus lookup; a Latest request still resolves, since it looks for a newer file.
	known := make([]profile.Source, len(reqs))
	if s.d.Stored != nil {
		for i, r := range reqs {
			if r.Repo == "" && r.FileID != 0 && !r.Latest {
				known[i], _ = s.d.Stored(r.Game, store.NexusKey(r.ModID, r.FileID))
			}
		}
	}
	out := make([]Item, 0, len(reqs))
	s.mu.Lock()
	for i, r := range reqs {
		if i := slices.IndexFunc(s.items, func(it *Item) bool { return sameDownload(it, r) }); i >= 0 {
			it := s.items[i]
			if it.State == StateFailed {
				it.State, it.Error = StateQueued, ""
			}
			if r.key != "" {
				it.key, it.expires = r.key, r.expires
			}
			if len(r.Disabled) > 0 {
				it.Disabled = slices.Clone(r.Disabled)
			}
			if len(r.Fomod) > 0 {
				it.Fomod, it.fomod = r.Fomod, r.Fomod
			}
			log.Printf("queue: mod %d file %d %s joined existing item %s (%s)", r.ModID, r.FileID, r.Repo, it.ID, it.State)
			out = append(out, *it)
			continue
		}
		it := &Item{
			ID: newID(), Kind: r.Kind, BatchID: r.BatchID, Game: r.Game, Profile: r.Profile, ModID: r.ModID, FileID: r.FileID,
			Name: r.Name, FileName: r.FileName, Version: r.Version, State: StateQueued, key: r.key, expires: r.expires,
			Repo: r.Repo, Tag: r.Tag, Asset: r.Asset, FallbackRepo: r.FallbackRepo, FallbackID: r.FallbackID, Latest: r.Latest, Disabled: slices.Clone(r.Disabled), Fomod: r.Fomod, fomod: r.Fomod,
		}
		if it.Repo != "" {
			it.Name = cmp.Or(it.Name, it.Repo)
		}
		if src := known[i]; src.Name != "" {
			it.FileName, it.Version = cmp.Or(it.FileName, src.Name), cmp.Or(it.Version, src.Version)
			it.Picture, it.endorsed = src.Picture, src.EndorsementCount
		}
		if id, file, ok := store.NexusFile(r.CurrentKey); ok && id == r.ModID {
			it.Current = file
		}
		s.items = append(s.items, it)
		log.Printf("queue: mod %d file %d %s queued as %s for profile %s", r.ModID, r.FileID, r.Repo, it.ID, r.Profile)
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
		// An update that has not settled on a file yet (or could not) takes the file the user clicked on its page.
		i = slices.IndexFunc(s.items, func(it *Item) bool {
			return (it.Kind == KindUpdate || it.Latest) && it.ModID == link.ModID && it.Repo == "" &&
				(it.State == StateWaitingClick || it.State == StateQueued || it.State == StateFailed)
		})
		if i >= 0 {
			s.items[i].FileID, s.items[i].Error = link.FileID, ""
		}
	}
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
		if (it.State == StateFailed || it.State == StateSkipped) && match(it) {
			it.State, it.Error = StateQueued, ""
		}
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}

func dismissable(state string) bool {
	return state == StateDone || state == StateFailed || state == StateSkipped || state == StateCancelled
}

// Dismiss removes one finished item. Queued and in-progress items are left alone.
func (s *Service) Dismiss(id string) {
	s.drop(func(it *Item) bool { return it.ID == id && dismissable(it.State) })
}

// ClearFinished removes every done, failed, cancelled or skipped item.
func (s *Service) ClearFinished() {
	s.drop(func(it *Item) bool { return dismissable(it.State) })
}

func (s *Service) drop(match func(*Item) bool) {
	s.mu.Lock()
	s.items = slices.DeleteFunc(s.items, match)
	s.mu.Unlock()
	s.publish(true)
}

// Skip drops an item that has not started, that failed, or that waits for the user.
func (s *Service) Skip(id string) {
	s.end(id, StateSkipped, StateFailed, StateQueued, StateWaitingClick, StateDownloading, StateNeedsChoice, StateNeedsConfirm, StateNeedsFomod, StateNeedsRoot, StateNeedsMerge)
}

// SkipAll skips every item that has not started.
func (s *Service) SkipAll() {
	s.mu.Lock()
	var ids []string
	for _, it := range s.items {
		if slices.Contains([]string{StateQueued, StateWaitingClick, StateNeedsChoice, StateNeedsConfirm, StateNeedsFomod, StateNeedsRoot, StateNeedsMerge}, it.State) {
			ids = append(ids, it.ID)
		}
	}
	s.mu.Unlock()
	for _, id := range ids {
		s.Skip(id)
	}
}

// SkipProfile skips every not-yet-started item for a profile.
func (s *Service) SkipProfile(game, profileID string) {
	s.mu.Lock()
	for _, it := range s.items {
		if it.Game == game && it.Profile == profileID &&
			slices.Contains([]string{StateQueued, StateWaitingClick, StateNeedsChoice, StateNeedsConfirm, StateNeedsFomod, StateNeedsRoot, StateNeedsMerge}, it.State) {
			it.State, it.Progress, it.Speed, it.key, it.staged, it.Error = StateSkipped, 0, 0, "", "", "profile deleted"
		}
	}
	s.mu.Unlock()
	s.publish(true)
}

// RestoreProfile requeues items held because their profile was deleted.
func (s *Service) RestoreProfile(game, profileID string) {
	s.mu.Lock()
	for _, it := range s.items {
		if it.Game == game && it.Profile == profileID && it.State == StateSkipped && it.Error == "profile deleted" {
			it.State, it.Error = StateQueued, ""
		}
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
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

// AnswerFomod installs an item waiting in StateNeedsFomod with the chosen plugins.
func (s *Service) AnswerFomod(id string, choices map[string]map[string][]string) {
	s.mu.Lock()
	if it := s.find(id); it != nil && it.State == StateNeedsFomod && it.staged != "" {
		it.State, it.fomod, it.Fomod, it.FomodKey = StateQueued, choices, choices, it.staged
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}

// AnswerRoot installs an item waiting in StateNeedsRoot with the chosen content folder.
func (s *Service) AnswerRoot(id, root string) {
	s.mu.Lock()
	if it := s.find(id); it != nil && it.State == StateNeedsRoot && it.staged != "" {
		it.State, it.chosenRoot = StateQueued, root
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}

// AnswerMerge installs an item waiting in StateNeedsMerge, either into the existing same-page entry or as its own.
func (s *Service) AnswerMerge(id string, add bool) {
	s.mu.Lock()
	if it := s.find(id); it != nil && it.State == StateNeedsMerge && it.readyZip {
		it.State, it.MergeAdd = StateQueued, add
	}
	s.mu.Unlock()
	s.publish(true)
	s.poke()
}

// FailRoot marks an item waiting in StateNeedsRoot as failed when the user cancels the folder picker.
func (s *Service) FailRoot(id string) {
	const msg = "No mod folder was chosen for this archive"
	s.mu.Lock()
	var rec *Item
	if it := s.find(id); it != nil && it.State == StateNeedsRoot {
		snap := *it
		rec = &snap
		it.State, it.Error = StateFailed, msg
		it.staged, it.Remap, it.chosenRoot = "", nil, ""
	}
	s.mu.Unlock()
	s.publish(true)
	if rec != nil {
		s.recordHistory(rec, StateFailed)
		notifyDesktopDownload(rec.Name, rec.Game, rec.Profile, false)
	}
}

func notifyDesktopDownload(name, game, profile string, ok bool) {
	title, key := "Download failed", "desktopDownloadFailed"
	if ok {
		title, key = "Download finished", "desktopDownloadFinished"
	}
	desktopnotify.SendIf(desktopnotify.Pref(key), title, name, map[string]any{"game": game, "profile": profile})
}

// Cancel stops an item, including a download under way. An install cannot be interrupted.
func (s *Service) Cancel(id string) {
	s.end(id, StateCancelled, StateQueued, StateWaitingClick, StateDownloading)
}

func (s *Service) end(id, to string, from ...string) {
	var rec *Item
	s.mu.Lock()
	if it := s.find(id); it != nil && slices.Contains(from, it.State) {
		snap := *it
		rec = &snap
		it.State, it.Progress, it.Speed, it.key, it.staged = to, 0, 0, "", ""
		if cancel := s.cancels[id]; cancel != nil {
			cancel()
		}
	}
	s.mu.Unlock()
	s.publish(true)
	if rec != nil {
		if to == StateCancelled || to == StateSkipped {
			dropDownload(s.dest(rec.ID, rec.FileName))
		}
		s.recordHistory(rec, to)
	}
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
