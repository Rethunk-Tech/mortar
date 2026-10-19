package queue

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

const cutAt = 5

// cutThen sends the first cutAt bytes of payload, then runs rest; later requests are answered by tail.
func cutThen(hits *atomic.Int32, rest func(r *http.Request, w http.ResponseWriter), tail func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) > 1 {
			tail(w, r)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		fmt.Fprint(w, payload[:cutAt])
		_ = http.NewResponseController(w).Flush()
		rest(r, w)
	}
}

func serveTail(rng *atomic.Value) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		rng.Store(r.Header.Get("Range"))
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", cutAt, len(payload)-1, len(payload)))
		w.WriteHeader(http.StatusPartialContent)
		fmt.Fprint(w, payload[cutAt:])
	}
}

func (f *fixture) partialSize(id string) int64 {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	it := f.s.find(id)
	return github.Partial(f.s.dest(it.ID, it.FileName))
}

func (f *fixture) waitPartial(id string) {
	f.t.Helper()
	f.wait("a partial on disk", func(State) bool {
		f.s.mu.Lock()
		it := f.s.find(id)
		ok := it != nil && it.State == StateDownloading && it.FileName != ""
		f.s.mu.Unlock()
		return ok && f.partialSize(id) == cutAt
	})
}

func TestPauseKeepsThePartialAndResumeContinuesIt(t *testing.T) {
	f := newFixture(t)
	var hits atomic.Int32
	var rng atomic.Value
	f.cdn = cutThen(&hits, func(r *http.Request, _ http.ResponseWriter) { <-r.Context().Done() }, serveTail(&rng))
	f.start()
	id := f.add(req(10))[0].ID
	f.waitPartial(id)

	f.s.PauseItem(id)
	st := f.wait("paused", f.item(StatePaused))
	if got := f.partialSize(id); got != cutAt || st.Items[0].Partial != cutAt {
		t.Fatalf("partial %d, item says %d", got, st.Items[0].Partial)
	}
	f.s.ResumeItem(id)
	f.wait("done", f.item(StateDone))
	if got, _ := rng.Load().(string); got != fmt.Sprintf("bytes=%d-", cutAt) {
		t.Fatalf("resume asked for %q", got)
	}
}

func TestRetryAfterANetworkErrorResumesThePartial(t *testing.T) {
	f := newFixture(t)
	var hits atomic.Int32
	var rng atomic.Value
	f.cdn = cutThen(&hits, func(*http.Request, http.ResponseWriter) { panic(http.ErrAbortHandler) }, serveTail(&rng))
	f.start()
	id := f.add(req(10))[0].ID
	st := f.wait("failed", f.item(StateFailed))
	if st.Items[0].ErrorKind != FailNetwork || st.Items[0].Partial != cutAt {
		t.Fatalf("failed item %+v", st.Items[0])
	}
	f.s.Retry(id)
	f.wait("done", f.item(StateDone))
	if got, _ := rng.Load().(string); got != fmt.Sprintf("bytes=%d-", cutAt) {
		t.Fatalf("retry asked for %q", got)
	}
}

// blockingCDN serves each Package's archive only after release closes, and records the order requests arrive in.
type blockingCDN struct {
	release chan struct{}
	mu      sync.Mutex
	started []string
}

func (b *blockingCDN) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	b.started = append(b.started, strings.TrimPrefix(r.URL.Path, "/"))
	b.mu.Unlock()
	select {
	case <-b.release:
		fmt.Fprint(w, payload)
	case <-r.Context().Done():
	}
}

func (b *blockingCDN) has(name string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.Contains(strings.Join(b.started, ","), name)
}

// packageFixture queues Me-Mod1..n from one server.
func packageFixture(t *testing.T, srvURL string) *fixture {
	t.Helper()
	f := newFixture(t)
	f.s.d.Closure = func(_ context.Context, _ string, roots []thunderstore.Ref, _ bool) ([]thunderstore.Resolved, error) {
		var out []thunderstore.Resolved
		for _, r := range roots {
			out = append(out, thunderstore.Resolved{Namespace: r.Namespace, Name: r.Name, Version: "1.0.0", URL: srvURL + "/" + r.Name})
		}
		return out, nil
	}
	f.s.d.InstallPackage = f.s.d.Install
	return f
}

func addPackages(t *testing.T, f *fixture, names ...string) {
	t.Helper()
	for _, n := range names {
		if _, err := f.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "Me-" + n}}); err != nil {
			t.Fatal(err)
		}
	}
}

func (f *fixture) stateOf(pkg string) Item {
	for _, it := range f.s.State().Items {
		if it.Package == pkg {
			return it
		}
	}
	return Item{}
}

func TestPausingFreesTheSlotForTheNextItem(t *testing.T) {
	cdn := &blockingCDN{release: make(chan struct{})}
	srv := httptest.NewServer(cdn)
	defer srv.Close()
	f := packageFixture(t, srv.URL)
	f.start()
	addPackages(t, f, "A", "B", "C", "D")
	f.wait("three downloading", func(st State) bool { return cdn.has("A") && cdn.has("B") && cdn.has("C") })
	if cdn.has("D") {
		t.Fatal("a fourth download started beside three")
	}
	f.s.PauseItem(f.stateOf("Me-B").ID)
	f.wait("D started", func(State) bool { return cdn.has("D") })
	if f.stateOf("Me-B").State != StatePaused {
		t.Fatalf("B is %s", f.stateOf("Me-B").State)
	}
	close(cdn.release)
}

func TestPausedItemsSurviveAReload(t *testing.T) {
	f := packageFixture(t, "https://x")
	f.s.Pause()
	addPackages(t, f, "A", "B")
	f.s.PauseItem(f.stateOf("Me-A").ID)
	b := f.stateOf("Me-B").ID
	f.s.mu.Lock()
	f.s.find(b).State, f.s.find(b).AutoResume = StatePaused, true
	f.s.mu.Unlock()
	f.s.publish(true)

	again, err := New(f.s.d)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, it := range again.State().Items {
		got = append(got, fmt.Sprintf("%s:%s:%v", it.Package, it.State, it.AutoResume))
	}
	if want := "Me-A:paused:false Me-B:paused:true"; strings.Join(got, " ") != want {
		t.Fatalf("got %v want %s", got, want)
	}
}

func TestHoldForPlayPausesRunningDownloadsAndSparesTheUsersOwnPauses(t *testing.T) {
	cdn := &blockingCDN{release: make(chan struct{})}
	srv := httptest.NewServer(cdn)
	defer srv.Close()
	f := packageFixture(t, srv.URL)
	f.start()
	addPackages(t, f, "A", "B")
	f.wait("both downloading", func(State) bool { return cdn.has("A") && cdn.has("B") })
	f.s.PauseItem(f.stateOf("Me-B").ID)

	f.s.HoldForPlay(true)
	f.wait("A held", func(State) bool { return f.stateOf("Me-A").State == StatePaused })
	if a, b := f.stateOf("Me-A"), f.stateOf("Me-B"); !a.AutoResume || b.AutoResume {
		t.Fatalf("A auto %v, B auto %v", a.AutoResume, b.AutoResume)
	}
	addPackages(t, f, "C")
	time.Sleep(30 * time.Millisecond)
	if cdn.has("C") {
		t.Fatal("a download started while the game ran")
	}

	close(cdn.release)
	f.s.HoldForPlay(false)
	f.wait("A and C done, B still paused", func(State) bool {
		return f.stateOf("Me-A").State == StateDone && f.stateOf("Me-C").State == StateDone && f.stateOf("Me-B").State == StatePaused
	})
}
