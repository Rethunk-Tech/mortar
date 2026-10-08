package queue

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/ids"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nxm"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

const payload = "archive bytes"

type fixture struct {
	// installErr, when set, fails an install with the error it returns; allowed records AllowUnscanned calls.
	installErr   func() error
	allowed      []string
	t            *testing.T
	s            *Service
	dir          string
	premium      atomic.Bool
	keep         atomic.Bool
	cdn          http.HandlerFunc
	limitNow     atomic.Bool
	clock        atomic.Int64
	mu           sync.Mutex
	opened       []string
	installs     []profile.Source
	keys         []string
	samePage     func(game, profileID string, in profile.IncomingFile) (profile.MergeAsk, int, bool)
	installExtra func(game, profileID, entryKey, path string, src profile.Source) (profile.InstallResult, error)
	stored       map[string]profile.Source
	newest       atomic.Int32
	// calls counts Nexus API requests.
	calls     atomic.Int32
	fromStore []string
	// published is the last state publish finished writing; waiting on it rather than State keeps a test from
	// ending while queue.json is still being written.
	published atomic.Pointer[State]
}

func (f *fixture) now() time.Time { return time.Unix(f.clock.Load(), 0).UTC() }

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{t: t, dir: t.TempDir()}
	f.clock.Store(1_000_000)
	f.premium.Store(true)
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/games/stardewvalley/mods/1/files.json", func(w http.ResponseWriter, _ *http.Request) {
		f.headers(w)
		fmt.Fprint(w, `{"files":[
			{"file_id":10,"file_name":"a-1.0.zip","version":"1.0","category_name":"MAIN","size_kb":3,"is_primary":true,"md5_hash":"785d887b432c6e90f8ac1ac31ba3d845"},
			{"file_id":11,"file_name":"a-2.0.zip","version":"2.0","category_name":"MAIN","size_kb":4,"md5_hash":"785d887b432c6e90f8ac1ac31ba3d845"},
			{"file_id":12,"file_name":"a-2.0-alt.zip","version":"2.0","category_name":"OPTIONAL","size_kb":4,"md5_hash":"785d887b432c6e90f8ac1ac31ba3d845"}]}`)
	})
	mux.HandleFunc("/v1/games/stardewvalley/mods/1.json", func(w http.ResponseWriter, _ *http.Request) {
		f.headers(w)
		fmt.Fprint(w, `{"name":"Alpha","author":"me","picture_url":"https://img/a.png","endorsement_count":7}`)
	})
	mux.HandleFunc("/v1/games/stardewvalley/mods/1/files/", func(w http.ResponseWriter, r *http.Request) {
		if f.limitNow.Load() {
			reset := f.now().Add(time.Hour).Format(time.RFC3339)
			w.Header().Set("X-Rl-Daily-Remaining", "0")
			w.Header().Set("X-Rl-Hourly-Remaining", "0")
			w.Header().Set("X-Rl-Daily-Reset", reset)
			w.Header().Set("X-Rl-Hourly-Reset", reset)
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if key := r.URL.Query().Get("key"); key != "" {
			f.mu.Lock()
			f.keys = append(f.keys, key)
			f.mu.Unlock()
		} else if !f.premium.Load() {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		f.headers(w)
		fmt.Fprintf(w, `[{"name":"CDN","URI":%q}]`, srv.URL+"/cdn/file.zip")
	})
	mux.HandleFunc("/cdn/file.zip", func(w http.ResponseWriter, r *http.Request) {
		if f.cdn != nil {
			f.cdn(w, r)
			return
		}
		fmt.Fprint(w, payload)
	})
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			f.calls.Add(1)
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	c := nexus.New("test")
	c.BaseURL, c.CacheDir, c.HTTP, c.Now = srv.URL, t.TempDir(), srv.Client(), f.now
	client := c.WithKey("secret")
	s, err := New(Deps{
		Client:  func() (*nexus.Client, error) { return client, nil },
		Premium: f.premium.Load,
		Install: func(_ context.Context, _, _, path string, src profile.Source) (profile.InstallResult, error) {
			if f.installErr != nil {
				if err := f.installErr(); err != nil {
					return profile.InstallResult{}, err
				}
			}
			b, err := fsx.ReadFile(path)
			if err != nil || string(b) != payload {
				return profile.InstallResult{}, fmt.Errorf("installer read %q: %w", b, err)
			}
			f.mu.Lock()
			f.installs = append(f.installs, src)
			f.mu.Unlock()
			return profile.InstallResult{}, nil
		},
		AllowUnscanned: func(game, profileID, key, name, detection string) error {
			f.mu.Lock()
			f.allowed = append(f.allowed, game+"|"+profileID+"|"+key+"|"+name+"|"+detection)
			f.mu.Unlock()
			return nil
		},
		Newest:       func(_, _ string, _, _ int) int { return int(f.newest.Load()) },
		KeepArchives: f.keep.Load,
		SamePage: func(game, profileID string, in profile.IncomingFile) (profile.MergeAsk, int, bool) {
			if f.samePage == nil {
				return profile.MergeAsk{}, 0, false
			}
			return f.samePage(game, profileID, in)
		},
		InstallExtra: func(_ context.Context, game, profileID, entryKey, path string, src profile.Source) (profile.InstallResult, error) {
			if f.installExtra == nil {
				return profile.InstallResult{}, fmt.Errorf("unexpected extra install of %s", entryKey)
			}
			return f.installExtra(game, profileID, entryKey, path, src)
		},
		Stored: func(_, key string) bool {
			f.mu.Lock()
			defer f.mu.Unlock()
			_, ok := f.stored[key]
			return ok
		},
		SourcesOf: func(string) map[string]profile.Source {
			f.mu.Lock()
			defer f.mu.Unlock()
			return maps.Clone(f.stored)
		},
		InstallStaged: func(_, _, key string, src profile.Source) (profile.InstallResult, error) {
			f.mu.Lock()
			f.fromStore = append(f.fromStore, key)
			f.installs = append(f.installs, src)
			f.mu.Unlock()
			return profile.InstallResult{}, nil
		},
		OpenURL: func(u string) error {
			f.mu.Lock()
			f.opened = append(f.opened, u)
			f.mu.Unlock()
			return nil
		},
		HTTP: srv.Client(), Dir: f.dir, Now: f.now,
		Changed: func(st State) { f.published.Store(&st) },
	})
	if err != nil {
		t.Fatal(err)
	}
	f.s = s
	return f
}

func (f *fixture) headers(w http.ResponseWriter) {
	w.Header().Set("X-Rl-Daily-Remaining", "500")
	w.Header().Set("X-Rl-Hourly-Remaining", "100")
}

func (f *fixture) start() {
	ctx, cancel := context.WithCancel(context.Background())
	wait := Run(ctx, f.s, make(chan nxmsvc.Assignment))
	f.t.Cleanup(func() {
		cancel()
		wait()
	})
}

func (f *fixture) add(reqs ...Request) []Item {
	f.t.Helper()
	items, err := f.s.Add(f.t.Context(), reqs)
	if err != nil {
		f.t.Fatal(err)
	}
	return items
}

func (f *fixture) wait(what string, ok func(State) bool) State {
	f.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st := f.published.Load(); st != nil && ok(*st) {
			return *st
		}
		time.Sleep(5 * time.Millisecond)
	}
	f.t.Fatalf("timed out waiting for %s: %+v", what, f.s.State())
	return State{}
}

func (f *fixture) item(state string) func(State) bool {
	return func(st State) bool { return len(st.Items) > 0 && st.Items[0].State == state }
}

// leftovers fails when temp downloads outlive their item; a cancelled download's goroutine removes its files after
// the item already reads Cancelled, so it gets a moment to finish.
func (f *fixture) leftovers() {
	f.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		entries, _ := os.ReadDir(filepath.Join(f.dir, downloadsDir))
		if len(entries) == 0 {
			return
		}
		if time.Now().After(deadline) {
			f.t.Errorf("temp downloads left behind: %v", entries)
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func req(fileID int) Request {
	return Request{Kind: KindInstall, Game: "stardew", Profile: "p1", ModID: 1, FileID: fileID}
}

func TestPremiumDownloadsAndInstallsWithoutClicks(t *testing.T) {
	f := newFixture(t)
	f.start()
	if _, err := f.s.Add(t.Context(), []Request{req(10), {Kind: KindUpdate, Game: "stardew", Profile: "p1", ModID: 1, Version: "2.0", CurrentKey: "nexus-1-10"}}); err != nil {
		t.Fatal(err)
	}
	st := f.wait("both done", func(st State) bool {
		return len(st.Items) == 2 && st.Items[0].State == StateDone && st.Items[1].State == StateDone
	})
	if st.Items[1].FileID != 11 || st.Items[1].FileName != "a-2.0.zip" || st.Items[0].Name != "Alpha" {
		t.Errorf("update resolved or named wrongly: %+v", st.Items)
	}
	src := f.installs[0]
	if src.Kind != "nexus" || src.ModID != 1 || src.FileID != 10 || src.Picture != "https://img/a.png" || src.EndorsementCount != 7 || src.Name != "a-1.0.zip" {
		t.Errorf("source %+v", src)
	}
	if len(f.opened) != 0 {
		t.Errorf("a premium download opened pages: %v", f.opened)
	}
	f.leftovers()
}

func TestSkipProfileHoldsDownloadsUntilRestore(t *testing.T) {
	f := newFixture(t)
	f.s.Pause()
	f.add(req(10))
	f.s.SkipProfile("stardew", "p1")
	st := f.s.State()
	if len(st.Items) != 1 || st.Items[0].State != StateSkipped || st.Items[0].Error != "profile deleted" {
		t.Fatalf("deleted profile queue state = %+v", st.Items)
	}
	f.s.RestoreProfile("stardew", "p1")
	st = f.s.State()
	if st.Items[0].State != StateQueued || st.Items[0].Error != "" {
		t.Fatalf("restored profile queue state = %+v", st.Items)
	}
}

func TestSkipCancelsAFetch(t *testing.T) {
	f := newFixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	f.cdn = func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		fmt.Fprint(w, payload)
	}
	f.start()
	f.add(req(10))
	<-entered
	f.s.Skip(f.s.State().Items[0].ID)
	close(release)
	st := f.wait("skipped fetch", f.item(StateSkipped))
	if st.Items[0].Error != "" {
		t.Errorf("skip error = %q", st.Items[0].Error)
	}
}

func TestFetchSlotsCapPremiumAndFreeSources(t *testing.T) {
	f := newFixture(t)
	for _, tc := range []struct {
		name    string
		premium bool
		want    int
	}{
		{name: "premium", premium: true, want: 3},
		{name: "free", premium: false, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f.premium.Store(tc.premium)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var active, maxActive atomic.Int32
			var wg sync.WaitGroup
			for range 6 {
				wg.Go(func() {
					release, err := f.s.fetchSlot(ctx, Item{})
					if err != nil {
						return
					}
					n := active.Add(1)
					for {
						old := maxActive.Load()
						if n <= old || maxActive.CompareAndSwap(old, n) {
							break
						}
					}
					time.Sleep(10 * time.Millisecond)
					active.Add(-1)
					release()
				})
			}
			wg.Wait()
			if got := int(maxActive.Load()); got > tc.want {
				t.Fatalf("max concurrent fetches = %d, want at most %d", got, tc.want)
			}
		})
	}
}

func TestInstallsAreSerialized(t *testing.T) {
	f := newFixture(t)
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var active, maxActive atomic.Int32
	f.s.d.Install = func(context.Context, string, string, string, profile.Source) (profile.InstallResult, error) {
		n := active.Add(1)
		for {
			old := maxActive.Load()
			if n <= old || maxActive.CompareAndSwap(old, n) {
				break
			}
		}
		entered <- struct{}{}
		<-release
		active.Add(-1)
		return profile.InstallResult{}, nil
	}
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			_ = f.s.installNexusPath(t.Context(), Item{ID: ids.New(), Game: "stardew", Profile: "p1"}, "", nexus.Mod{})
		})
	}
	<-entered
	select {
	case <-entered:
		t.Fatal("second install started before the first finished")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	wg.Wait()
	if maxActive.Load() != 1 {
		t.Fatalf("max concurrent installs = %d, want 1", maxActive.Load())
	}
}

func TestAStoredFileInstallsFromTheStoreWithoutAClick(t *testing.T) {
	f := newFixture(t)
	f.premium.Store(false)
	f.stored = map[string]profile.Source{"nexus-1-10": {Kind: "nexus", Name: "a-1.0.zip", ModID: 1, FileID: 10, Version: "1.0", Picture: "https://img/a.png", EndorsementCount: 7}}
	f.start()
	f.add(req(10))
	st := f.wait("done", f.item(StateDone))
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.opened) != 0 || len(f.keys) != 0 || f.calls.Load() != 0 || len(f.fromStore) != 1 || f.fromStore[0] != "nexus-1-10" {
		t.Fatalf("opened %v, keys %v, from store %v", f.opened, f.keys, f.fromStore)
	}
	if src := f.installs[0]; src.Kind != "nexus" || src.ModID != 1 || src.FileID != 10 || src.Version != "1.0" || src.Picture != "https://img/a.png" || st.Items[0].FileName != "a-1.0.zip" {
		t.Errorf("source %+v, item %+v", src, st.Items[0])
	}
}

func TestAQueuedItemIsNamedBeforeItsTurn(t *testing.T) {
	f := newFixture(t)
	f.s.Pause()
	assigned := make(chan nxmsvc.Assignment, 1)
	ctx, cancel := context.WithCancel(context.Background())
	wait := Run(ctx, f.s, assigned)
	t.Cleanup(func() {
		cancel()
		wait()
	})
	assigned <- nxmsvc.Assignment{Link: nxm.Link{ModID: 1, FileID: 10, Key: "k", Expires: f.now().Unix() + 600}, Game: "stardew", Profile: "p1"}
	f.wait("the name", func(st State) bool {
		return len(st.Items) == 1 && st.Items[0].State == StateQueued && st.Items[0].Name == "Alpha" && st.Items[0].Picture == "https://img/a.png"
	})
}

func TestFreeAccountWaitsForTheClickThenTakesTheLink(t *testing.T) {
	f := newFixture(t)
	f.premium.Store(false)
	f.start()
	f.add(req(10))
	f.wait("the click", f.item(StateWaitingClick))
	f.wait("the page", func(State) bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return len(f.opened) > 0
	})
	want := "https://www.nexusmods.com/stardewvalley/mods/1?tab=files&file_id=10&nmm=1"
	f.mu.Lock()
	opened := slices.Clone(f.opened)
	f.mu.Unlock()
	if len(opened) != 1 || opened[0] != want {
		t.Fatalf("opened %v", opened)
	}
	if f.s.Route(nxm.Link{ModID: 1, FileID: 99, Key: "k", Expires: f.now().Unix() + 600}) {
		t.Error("a link for another file was taken")
	}
	if !f.s.Route(nxm.Link{ModID: 1, FileID: 10, Key: "k", Expires: f.now().Unix() + 600}) {
		t.Fatal("the waiting item did not take its link")
	}
	f.wait("done", f.item(StateDone))
	if len(f.keys) != 1 || f.keys[0] != "k" {
		t.Errorf("download link was asked with keys %v", f.keys)
	}
}

func TestWaitingLatestItemIsSkippedOnceTheProfileHasANewerFile(t *testing.T) {
	f := newFixture(t)
	f.premium.Store(false)
	f.start()
	exact, latest := req(10), req(10)
	exact.Profile, latest.Latest = "p2", true
	f.add(latest, exact)
	f.wait("the click", f.item(StateWaitingClick))
	f.newest.Store(11)
	f.s.poke()
	f.wait("the skip", func(st State) bool {
		return len(st.Items) == 2 && st.Items[0].State == StateSkipped && st.Items[1].State == StateWaitingClick
	})
	if got := f.s.State().Items[0].Error; got != AlreadyNewest {
		t.Errorf("skip reason = %q, want %q", got, AlreadyNewest)
	}
}

func TestExpiredKeyReopensThePage(t *testing.T) {
	f := newFixture(t)
	f.premium.Store(false)
	f.start()
	f.add(req(10))
	f.wait("the click", f.item(StateWaitingClick))
	f.s.Route(nxm.Link{ModID: 1, FileID: 10, Key: "old", Expires: f.now().Unix() + 1})
	f.wait("the page again", func(State) bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return len(f.opened) == 2
	})
	if got := f.s.State().Items[0].State; got != StateWaitingClick {
		t.Errorf("state %s", got)
	}
}

func TestRateLimitPausesUntilTheReset(t *testing.T) {
	f := newFixture(t)
	f.limitNow.Store(true)
	f.start()
	f.add(req(10))
	st := f.wait("the limit", func(st State) bool { return st.LimitedUntil != 0 })
	if st.LimitedUntil != f.now().Add(time.Hour).Unix() || st.Items[0].State != StateQueued {
		t.Fatalf("state %+v", st)
	}
	f.limitNow.Store(false)
	f.clock.Add(3601)
	f.s.poke()
	f.wait("done after the reset", f.item(StateDone))
}

func TestFailedRetrySkipAndCancel(t *testing.T) {
	f := newFixture(t)
	var fail atomic.Bool
	fail.Store(true)
	block := make(chan struct{})
	var hold atomic.Bool
	f.cdn = func(w http.ResponseWriter, r *http.Request) {
		if hold.Load() {
			w.WriteHeader(http.StatusOK)
			_ = http.NewResponseController(w).Flush()

			select {
			case <-r.Context().Done():
			case <-block:
			}
			return
		}
		if fail.Load() {
			http.Error(w, "no", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, payload)
	}
	t.Cleanup(func() { close(block) })
	f.start()
	f.add(req(10))
	st := f.wait("failure", f.item(StateFailed))
	if !strings.Contains(st.Items[0].Error, "500") {
		t.Errorf("error %q", st.Items[0].Error)
	}
	fail.Store(false)
	f.s.Retry(st.Items[0].ID)
	f.wait("done after retry", f.item(StateDone))

	fail.Store(true)
	f.add(req(11))
	st = f.wait("second failure", func(st State) bool { return len(st.Items) == 2 && st.Items[1].State == StateFailed })
	f.s.Skip(st.Items[1].ID)
	f.wait("skipped", func(st State) bool { return st.Items[1].State == StateSkipped })

	hold.Store(true)
	f.add(req(12))
	st = f.wait("downloading", func(st State) bool { return len(st.Items) == 3 && st.Items[2].State == StateDownloading })
	f.s.Cancel(st.Items[2].ID)
	f.wait("cancelled", func(st State) bool { return st.Items[2].State == StateCancelled })
	f.leftovers()
	if len(f.installs) != 1 {
		t.Errorf("installs %d", len(f.installs))
	}
}

func TestPauseHoldsBackNewDownloads(t *testing.T) {
	f := newFixture(t)
	f.start()
	f.s.Pause()
	f.add(req(10))
	time.Sleep(50 * time.Millisecond)
	if st := f.s.State(); !st.Paused || st.Items[0].State != StateQueued {
		t.Fatalf("state %+v", st)
	}
	f.s.Resume()
	f.wait("done", f.item(StateDone))
}

func TestQueueSurvivesARestart(t *testing.T) {
	f := newFixture(t)
	f.s.Pause()
	f.add(req(10), req(11))
	f.s.mu.Lock()
	f.s.items[0].State = StateDownloading
	f.s.mu.Unlock()
	f.s.publish(true)
	f.s.mu.Lock()
	f.s.items[0].key = "never-saved"
	f.s.mu.Unlock()
	again, err := New(f.s.d)
	if err != nil {
		t.Fatal(err)
	}
	st := again.State()
	if !st.Paused || len(st.Items) != 2 || st.Items[0].State != StateQueued || st.Items[0].FileID != 10 {
		t.Fatalf("restored %+v", st)
	}
	if raw, _ := os.ReadFile(filepath.Join(f.dir, fileName)); strings.Contains(string(raw), "never-saved") {
		t.Error("a download key was written to disk")
	}
}

func TestRestartPausesAQueueHoldingNexusFiles(t *testing.T) {
	f := newFixture(t)
	f.add(req(10))
	again, err := New(f.s.d)
	if err != nil {
		t.Fatal(err)
	}
	if !again.State().Paused {
		t.Fatal("a Nexus file resumed on its own after a restart")
	}
	other := newFixture(t)
	if _, err := other.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "stardew", Profile: "p1", Repo: "o/r"}}); err != nil {
		t.Fatal(err)
	}
	again, err = New(other.s.d)
	if err != nil {
		t.Fatal(err)
	}
	if again.State().Paused {
		t.Fatal("a GitHub-only queue was paused")
	}
}

func TestSignedOutRefusesToQueue(t *testing.T) {
	f := newFixture(t)
	f.s.d.Client = func() (*nexus.Client, error) { return nil, fmt.Errorf("sign in") }
	if _, err := f.s.Add(t.Context(), []Request{req(10)}); err == nil {
		t.Fatal("queued while signed out")
	}
}

func TestNexusDownloadOverTheCapFails(t *testing.T) {
	f := newFixture(t)
	f.cdn = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(archive.DefaultMaxTotalBytes+1))
		fmt.Fprint(w, payload)
	}
	f.start()
	f.add(req(10))
	st := f.wait("failed", f.item(StateFailed))
	if !strings.Contains(st.Items[0].Error, "larger than") || len(f.installs) != 0 {
		t.Fatalf("item %+v installs %v", st.Items[0], f.installs)
	}
	f.leftovers()
}

func TestItemsForARunningProfileWait(t *testing.T) {
	f := newFixture(t)
	var running atomic.Bool
	running.Store(true)
	f.s.d.Running = func(_, id string) bool { return id == "p1" && running.Load() }
	f.start()
	f.add(req(10))
	time.Sleep(100 * time.Millisecond)
	if st := f.s.State(); st.Items[0].State != StateQueued || len(f.installs) != 0 {
		t.Fatalf("item for a running profile moved on: %+v", st.Items[0])
	}
	running.Store(false)
	f.s.Resume()
	f.wait("done", f.item(StateDone))
}

func TestARejectedLinkCanBeRetried(t *testing.T) {
	f := newFixture(t)
	client := f.s.d.Client
	f.s.d.Client = func() (*nexus.Client, error) { return nil, fmt.Errorf("sign in") }
	ctx, cancel := context.WithCancel(context.Background())
	assigned := make(chan nxmsvc.Assignment)
	wait := Run(ctx, f.s, assigned)
	t.Cleanup(func() {
		cancel()
		wait()
	})
	assigned <- nxmsvc.Assignment{Link: nxm.Link{ModID: 1, FileID: 10}, Game: "stardew", Profile: "p1"}
	it := f.wait("the rejection", f.item(StateFailed)).Items[0]
	if it.Game != "stardew" || it.Profile != "p1" || it.FileID != 10 {
		t.Fatalf("rejected item %+v", it)
	}
	f.s.d.Client = client
	f.s.Retry(it.ID)
	f.wait("done", f.item(StateDone))
}

func TestAddValidatesTheWholeBatch(t *testing.T) {
	f := newFixture(t)
	if _, err := f.s.Add(t.Context(), []Request{req(10), {Kind: KindInstall, Game: "stardew", ModID: 1, FileID: 11}}); err == nil {
		t.Fatal("expected error")
	}
	if st := f.s.State(); len(st.Items) != 0 {
		t.Fatalf("partial add: %+v", st.Items)
	}
}

func TestAddCarriesHistoryBatchID(t *testing.T) {
	f := newFixture(t)
	request := req(10)
	request.BatchID = "batch-1"
	items, err := f.s.Add(t.Context(), []Request{request})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].BatchID != request.BatchID {
		t.Fatalf("queued item = %+v", items)
	}
}

func TestDismissAndClearFinishedLeaveActiveItems(t *testing.T) {
	f := newFixture(t)
	f.s.mu.Lock()
	f.s.items = []*Item{
		{ID: "queued", Name: "q", State: StateQueued, Game: "stardew"},
		{ID: "active", Name: "a", State: StateDownloading, Game: "stardew"},
		{ID: "done", Name: "d", State: StateDone, Game: "stardew", staged: "staged-key"},
		{ID: "fail", Name: "f", State: StateFailed, Game: "stardew"},
		{ID: "skip", Name: "s", State: StateSkipped, Game: "stardew"},
		{ID: "cancel", Name: "c", State: StateCancelled, Game: "stardew"},
		{ID: "choice", Name: "ch", State: StateNeedsChoice, Game: "stardew"},
	}
	f.s.mu.Unlock()
	f.s.Dismiss("queued")
	f.s.Dismiss("active")
	f.s.Dismiss("choice")
	if ids := itemIDs(f.s.State()); len(ids) != 7 {
		t.Fatalf("dismissed a live item: %v", ids)
	}
	f.s.Dismiss("done")
	if slices.Contains(itemIDs(f.s.State()), "done") {
		t.Fatal("done still present")
	}
	if keys := f.s.StagedKeys(); len(keys) != 0 {
		t.Fatalf("staged keys %v", keys)
	}
	f.s.ClearFinished()
	got := itemIDs(f.s.State())
	if !slices.Equal(got, []string{"queued", "active", "choice"}) {
		t.Fatalf("after clear: %v", got)
	}
	b, err := os.ReadFile(filepath.Join(f.dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{"queued", "active", "choice"} {
		if !strings.Contains(text, want) {
			t.Errorf("queue.json missing %s", want)
		}
	}
	for _, drop := range []string{`"id": "fail"`, `"id": "skip"`, `"id": "cancel"`, `"id": "done"`} {
		if strings.Contains(text, drop) {
			t.Errorf("queue.json still has %s", drop)
		}
	}
}

func TestNewQueueClientTimesOutSlowHeaders(t *testing.T) {
	s, err := New(Deps{Dir: t.TempDir(), Now: func() time.Time { return time.Unix(1, 0).UTC() }})
	if err != nil {
		t.Fatal(err)
	}
	tr, ok := s.d.HTTP.Transport.(*http.Transport)
	if !ok || tr.ResponseHeaderTimeout != 20*time.Second {
		t.Fatalf("queue HTTP client missing ResponseHeaderTimeout: %#v", s.d.HTTP)
	}
	if s.d.HTTP.Timeout != 30*time.Minute {
		t.Fatalf("queue HTTP client missing Timeout: %#v", s.d.HTTP)
	}
}

func itemIDs(st State) []string {
	ids := make([]string, len(st.Items))
	for i, it := range st.Items {
		ids[i] = it.ID
	}
	return ids
}

func TestAutoRetryFetchesBeforeFailing(t *testing.T) {
	f := newFixture(t)
	var hits atomic.Int32
	f.cdn = func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) < 3 {
			http.Error(w, "no", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, payload)
	}
	f.s.d.RetryFetches = func() int { return 2 }
	f.start()
	f.add(req(10))
	f.wait("done after retries", f.item(StateDone))
	if hits.Load() != 3 {
		t.Fatalf("fetches = %d", hits.Load())
	}
}

func TestPauseDownloadsWhileGameRuns(t *testing.T) {
	f := newFixture(t)
	var busy atomic.Bool
	busy.Store(true)
	f.s.d.PauseWhilePlaying = func() bool { return true }
	f.s.d.GameBusy = busy.Load
	f.start()
	f.add(req(10))
	time.Sleep(80 * time.Millisecond)
	if st := f.s.State(); len(st.Items) != 1 || st.Items[0].State != StateQueued {
		t.Fatalf("while busy %+v", f.s.State())
	}
	busy.Store(false)
	NotifyUnlocked(f.s)
	f.wait("resumed", f.item(StateDone))
}

func TestNexusMD5MismatchFailsDownload(t *testing.T) {
	f := newFixture(t)
	f.s.d.VerifyNexusMD5 = func() bool { return true }
	f.cdn = func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "not the archive")
	}
	f.start()
	f.add(req(10))
	st := f.wait("md5 failed", f.item(StateFailed))
	if !strings.Contains(st.Items[0].Error, "MD5") {
		t.Fatalf("error %q", st.Items[0].Error)
	}
}

func TestNexusMD5MatchAllowsInstall(t *testing.T) {
	f := newFixture(t)
	f.s.d.VerifyNexusMD5 = func() bool { return true }
	f.start()
	f.add(req(10))
	f.wait("md5 ok", f.item(StateDone))
}

// A file named up front (a link, a share) skips the list the queue would choose from, so its md5 is looked up; a
// mismatch drops the download, so Retry fetches it again instead of rechecking the bad copy.
func TestNexusMD5OfANamedFileIsLookedUpAndRetryRefetches(t *testing.T) {
	f := newFixture(t)
	f.s.d.VerifyNexusMD5 = func() bool { return true }
	var bad atomic.Bool
	bad.Store(true)
	f.cdn = func(w http.ResponseWriter, _ *http.Request) {
		if bad.Load() {
			fmt.Fprint(w, "not the archive")
			return
		}
		fmt.Fprint(w, payload)
	}
	f.start()
	named := req(10)
	named.FileName = "a-1.0.zip"
	f.add(named)
	st := f.wait("md5 failed", f.item(StateFailed))
	if !strings.Contains(st.Items[0].Error, "MD5") {
		t.Fatalf("error %q", st.Items[0].Error)
	}
	bad.Store(false)
	f.s.Retry(st.Items[0].ID)
	f.wait("retried", f.item(StateDone))
}

func TestRouteGivesAClickedFileToAPendingUpdate(t *testing.T) {
	f := newFixture(t)
	f.premium.Store(false)
	f.s.mu.Lock()
	f.s.items = append(f.s.items, &Item{ID: "u", Kind: KindUpdate, ModID: 41150, State: StateFailed, Error: "That item could not be found."})
	f.s.mu.Unlock()
	if !f.s.Route(nxm.Link{ModID: 41150, FileID: 185334, Key: "k", Expires: f.now().Unix() + 600}) {
		t.Fatal("the link should complete the pending update")
	}
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	it := f.s.items[len(f.s.items)-1]
	if it.FileID != 185334 || it.State != StateQueued || it.Error != "" {
		t.Fatalf("update item %+v", *it)
	}
}

func TestFinishedItemCarriesTheHistoryEventItRecorded(t *testing.T) {
	f := newFixture(t)
	var calls []string
	f.s.d.HistoryBatch = func(_, _, batchID string) (string, error) {
		calls = append(calls, batchID)
		if batchID == "" {
			return "", nil
		}
		return "bulk-ev", nil
	}
	f.s.mu.Lock()
	f.s.items = []*Item{
		{ID: "one", Name: "a", State: StateInstalling, Game: "stardew", Profile: "p1"},
		{ID: "batched", Name: "b", State: StateInstalling, Game: "stardew", Profile: "p1", BatchID: "batch-1"},
	}
	f.s.mu.Unlock()
	if err := f.s.finish("one", nil, false, "own-ev"); err != nil {
		t.Fatal(err)
	}
	if err := f.s.finish("batched", nil, false, "own-ev"); err != nil {
		t.Fatal(err)
	}
	st := f.s.State()
	if st.Items[0].Change != "own-ev" || st.Items[1].Change != "bulk-ev" {
		t.Fatalf("changes %q %q, want own-ev bulk-ev", st.Items[0].Change, st.Items[1].Change)
	}
	if !slices.Equal(calls, []string{"", "batch-1", ""}) {
		t.Fatalf("history batch calls %q", calls)
	}
}
