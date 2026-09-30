package queue

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/github"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/nxmsvc"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

type ghFixture struct {
	*fixture
	multi   atomic.Bool
	limited atomic.Bool
	verdict atomic.Value
	ok      atomic.Bool
	staged  []profile.Source
	final   []profile.Source
}

// newGitHubFixture serves two releases of me/mod: v2.0.0 with one archive (or two when multi is set) and v1.0.0.
func newGitHubFixture(t *testing.T) *ghFixture {
	t.Helper()
	g := &ghFixture{fixture: newFixture(t)}
	g.ok.Store(true)
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/me/mod/releases", func(w http.ResponseWriter, _ *http.Request) {
		if g.limited.Load() {
			w.Header().Set("X-Ratelimit-Remaining", "0")
			w.Header().Set("X-Ratelimit-Reset", strconv.FormatInt(g.now().Add(time.Hour).Unix(), 10))
			w.WriteHeader(http.StatusForbidden)
			return
		}
		second := ""
		if g.multi.Load() {
			second = fmt.Sprintf(`,{"name":"mod-2.0.0-alt.zip","browser_download_url":%q}`, srv.URL+"/dl/alt.zip")
		}
		fmt.Fprintf(w, `[
			{"tag_name":"v2.1.0-beta","prerelease":true,"assets":[]},
			{"tag_name":"v2.0.0","assets":[{"name":"mod-2.0.0.zip","size":13,"browser_download_url":%q},{"name":"notes.txt","browser_download_url":%q}%s]},
			{"tag_name":"v1.0.0","assets":[{"name":"mod-1.0.0.zip","size":13,"browser_download_url":%q}]}]`,
			srv.URL+"/dl/main.zip", srv.URL+"/dl/notes.txt", second, srv.URL+"/dl/old.zip")
	})
	mux.HandleFunc("/dl/", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, payload) })
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	g.s.d.Client = func() (*nexus.Client, error) { return nil, fmt.Errorf("signed out") }
	g.s.d.GitHub = &github.Client{HTTP: srv.Client(), CacheDir: t.TempDir(), APIBase: srv.URL, Now: g.now}
	g.s.d.Stage = func(_ string, src profile.Source, path string) (string, []string, error) {
		if b, err := fsx.ReadFile(path); err != nil || string(b) != payload {
			return "", nil, fmt.Errorf("stage read %q: %w", b, err)
		}
		g.mu.Lock()
		g.staged = append(g.staged, src)
		g.mu.Unlock()
		return "github-key", []string{"me.mod"}, nil
	}
	g.s.d.InstallStaged = func(_, _, key string, src profile.Source) (profile.InstallResult, error) {
		if key != "github-key" {
			return profile.InstallResult{}, fmt.Errorf("installed key %q", key)
		}
		g.mu.Lock()
		g.final = append(g.final, src)
		g.mu.Unlock()
		return profile.InstallResult{}, nil
	}
	g.s.d.Verify = func(_ context.Context, id, owner, repo string) (bool, error) {
		if id != "me.mod" || owner != "me" || repo != "mod" {
			return false, fmt.Errorf("verified %s %s/%s", id, owner, repo)
		}
		if err, _ := g.verdict.Load().(error); err != nil {
			return false, err
		}
		return g.ok.Load(), nil
	}
	return g
}

func ghReq(version string) Request {
	return Request{Kind: KindUpdate, Game: "stardew", Profile: "p1", Repo: "me/mod", Version: version}
}

func TestGitHubSingleAssetInstalls(t *testing.T) {
	g := newGitHubFixture(t)
	g.start()
	if _, err := g.s.Add([]Request{ghReq("2.0.0")}); err != nil {
		t.Fatal(err)
	}
	st := g.wait("done", g.item(StateDone))
	it := st.Items[0]
	want := profile.Source{Kind: profile.KindGitHub, Name: "mod-2.0.0.zip", Version: "2.0.0", Repo: "me/mod", Tag: "v2.0.0", Asset: "mod-2.0.0.zip"}
	if len(g.final) != 1 || g.final[0] != want || it.Unverified || it.Name != "me/mod" || it.FileName != "mod-2.0.0.zip" {
		t.Errorf("item %+v installed %+v", it, g.final)
	}
	if len(g.opened) != 0 {
		t.Errorf("a GitHub download opened pages: %v", g.opened)
	}
}

func TestGitHubSeveralAssetsWaitForAChoice(t *testing.T) {
	g := newGitHubFixture(t)
	g.multi.Store(true)
	g.start()
	if _, err := g.s.Add([]Request{ghReq("")}); err != nil {
		t.Fatal(err)
	}
	st := g.wait("the choice", g.item(StateNeedsChoice))
	if got := st.Items[0].Assets; len(got) != 2 || got[0] != "mod-2.0.0.zip" || got[1] != "mod-2.0.0-alt.zip" {
		t.Fatalf("choices %v", got)
	}
	g.s.Choose(st.Items[0].ID, "not-offered.zip")
	if got := g.s.State().Items[0].State; got != StateNeedsChoice {
		t.Fatalf("an asset that was not offered moved the item to %s", got)
	}
	g.s.Choose(st.Items[0].ID, "mod-2.0.0-alt.zip")
	g.wait("done", g.item(StateDone))
	if len(g.final) != 1 || g.final[0].Asset != "mod-2.0.0-alt.zip" || g.final[0].Tag != "v2.0.0" {
		t.Errorf("installed %+v", g.final)
	}
}

func TestGitHubMismatchWaitsForConfirmation(t *testing.T) {
	g := newGitHubFixture(t)
	g.ok.Store(false)
	g.start()
	if _, err := g.s.Add([]Request{ghReq("2.0.0")}); err != nil {
		t.Fatal(err)
	}
	st := g.wait("the confirmation", g.item(StateNeedsConfirm))
	if len(g.final) != 0 {
		t.Fatal("installed before the user decided")
	}
	g.s.Confirm(st.Items[0].ID)
	st = g.wait("done", g.item(StateDone))
	if len(g.final) != 1 || len(g.staged) != 1 || st.Items[0].Unverified {
		t.Errorf("installed %+v staged %+v item %+v", g.final, g.staged, st.Items[0])
	}

	if _, err := g.s.Add([]Request{{Kind: KindInstall, Game: "stardew", Profile: "p2", Repo: "me/mod"}}); err != nil {
		t.Fatal(err)
	}
	st = g.wait("the second confirmation", func(st State) bool { return len(st.Items) == 2 && st.Items[1].State == StateNeedsConfirm })
	g.s.Skip(st.Items[1].ID)
	g.wait("skipped", func(st State) bool { return st.Items[1].State == StateSkipped })
	if len(g.final) != 1 {
		t.Errorf("a skipped download was installed: %+v", g.final)
	}
}

func TestGitHubUnknownSourceInstallsWithANote(t *testing.T) {
	g := newGitHubFixture(t)
	g.verdict.Store(github.ErrUnknown)
	g.start()
	if _, err := g.s.Add([]Request{ghReq("2.0.0")}); err != nil {
		t.Fatal(err)
	}
	st := g.wait("done", g.item(StateDone))
	if !st.Items[0].Unverified || len(g.final) != 1 {
		t.Errorf("item %+v installed %+v", st.Items[0], g.final)
	}
}

func TestGitHubRateLimitPausesUntilTheReset(t *testing.T) {
	g := newGitHubFixture(t)
	g.limited.Store(true)
	g.start()
	if _, err := g.s.Add([]Request{ghReq("2.0.0")}); err != nil {
		t.Fatal(err)
	}
	st := g.wait("the limit", func(st State) bool { return st.LimitedUntil != 0 })
	if st.LimitedUntil != g.now().Add(time.Hour).Unix() || st.Items[0].State != StateQueued {
		t.Fatalf("state %+v", st)
	}
	g.limited.Store(false)
	g.clock.Add(3601)
	g.s.poke()
	g.wait("done after the reset", g.item(StateDone))
}

func TestGitHubRequestsNeedNoSignInButAValidRepo(t *testing.T) {
	g := newGitHubFixture(t)
	for _, repo := range []string{"", "me", "me/mod/extra", "../etc/passwd", "me/mo d"} {
		r := ghReq("")
		r.Repo = repo
		if _, err := g.s.Add([]Request{r}); err == nil {
			t.Errorf("repo %q was queued", repo)
		}
	}
	if _, err := g.s.Add([]Request{ghReq("")}); err != nil {
		t.Fatalf("signed out: %v", err)
	}
}

func TestAddRefusesDotRepos(t *testing.T) {
	for _, repo := range []string{"me/..", "./mod", "../..", "me/."} {
		if validRepo(repo) {
			t.Errorf("validRepo(%q) = true", repo)
		}
	}
	if !validRepo("me/mod.cfg") {
		t.Error("a dotted repo name was refused")
	}
}

func TestConfirmSurvivesARestartAndItsInstallIgnoresCancel(t *testing.T) {
	g := newGitHubFixture(t)
	g.ok.Store(false)
	g.start()
	if _, err := g.s.Add([]Request{ghReq("2.0.0")}); err != nil {
		t.Fatal(err)
	}
	id := g.wait("the confirmation", g.item(StateNeedsConfirm)).Items[0].ID

	again, err := New(g.s.d)
	if err != nil {
		t.Fatal(err)
	}
	if st := again.State().Items[0].State; st != StateNeedsConfirm {
		t.Fatalf("after a restart the item is %s", st)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	again.d.InstallStaged = func(_, _, key string, _ profile.Source) (profile.InstallResult, error) {
		close(entered)
		<-release
		if key != "github-key" {
			return profile.InstallResult{}, fmt.Errorf("installed key %q", key)
		}
		return profile.InstallResult{}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	Run(ctx, again, make(chan nxmsvc.Assignment))
	again.Confirm(id)
	<-entered
	again.Cancel(id)
	if st := again.State().Items[0].State; st != StateInstalling {
		t.Fatalf("during the install the item is %s", st)
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for again.State().Items[0].State != StateDone {
		if time.Now().After(deadline) {
			t.Fatalf("not done: %+v", again.State().Items[0])
		}
		time.Sleep(5 * time.Millisecond)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.staged) != 1 {
		t.Fatalf("downloaded %d times", len(g.staged))
	}
}
