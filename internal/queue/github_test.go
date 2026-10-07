package queue

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/github"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/smapi"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
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
	g.s.d.Stage = func(_ context.Context, _ string, src profile.Source, path string) (string, []mod.ID, error) {
		if b, err := fsx.ReadFile(path); err != nil || string(b) != payload {
			return "", nil, fmt.Errorf("stage read %q: %w", b, err)
		}
		g.mu.Lock()
		g.staged = append(g.staged, src)
		g.mu.Unlock()
		return "github-key", []mod.ID{"smapi:me.mod"}, nil
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
	g.s.d.Verify = func(_ context.Context, id mod.ID, owner, repo string) (bool, error) {
		if id != "smapi:me.mod" || owner != "me" || repo != "mod" {
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
	g.add(ghReq("2.0.0"))
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
	g.add(ghReq(""))
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
	// The choice is remembered for the repo, so the next download of it does not ask again.
	if _, err := g.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "stardew", Profile: "p2", Repo: "me/mod"}}); err != nil {
		t.Fatal(err)
	}
	g.wait("the second install", func(st State) bool { return len(st.Items) == 2 && st.Items[1].State == StateDone })
	if len(g.final) != 2 || g.final[1].Asset != "mod-2.0.0-alt.zip" {
		t.Errorf("installed %+v", g.final)
	}
}

// TestGitHubAssetPrefersTheLoadersShape serves a release with a source archive and two zips, only one holding a SMAPI
// manifest; that one is picked without asking.
func TestGitHubAssetPrefersTheLoadersShape(t *testing.T) {
	zipOf := func(names ...string) []byte {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		for _, n := range names {
			_, _ = zw.Create(n)
		}
		_ = zw.Close()
		return buf.Bytes()
	}
	files := map[string][]byte{
		"Mod-1.0.zip":        zipOf("Mod/manifest.json", "Mod/Mod.dll"),
		"Mod-1.0-extras.zip": zipOf("Extras/readme.txt"),
		"Mod-1.0-source.zip": zipOf("Mod/manifest.json", "Mod/Mod.cs"),
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if name := strings.TrimPrefix(r.URL.Path, "/dl/"); files[name] != nil {
			http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(files[name]))
			return
		}
		var assets []string
		for _, n := range []string{"Mod-1.0-source.zip", "Mod-1.0-extras.zip", "Mod-1.0.zip"} {
			assets = append(assets, fmt.Sprintf(`{"name":%q,"size":%d,"browser_download_url":%q}`, n, len(files[n]), srv.URL+"/dl/"+n))
		}
		fmt.Fprintf(w, `[{"tag_name":"v1.0","assets":[%s]}]`, strings.Join(assets, ","))
	}))
	t.Cleanup(srv.Close)
	f := newFixture(t)
	f.s.d.GitHub = &github.Client{HTTP: srv.Client(), CacheDir: t.TempDir(), APIBase: srv.URL, Now: f.now}
	rel, assets, err := f.s.GitHubAsset(t.Context(), "stardew", "me/mod", "")
	if err != nil || rel.Tag != "v1.0" || len(assets) != 1 || assets[0].Name != "Mod-1.0.zip" {
		t.Fatalf("picked %s %+v %v", rel.Tag, assets, err)
	}
}

// awaitConfirmation queues a release whose archive fails the check and returns the state once it waits.
func (g *ghFixture) awaitConfirmation() State {
	g.ok.Store(false)
	g.start()
	g.add(ghReq("2.0.0"))
	return g.wait("the confirmation", g.item(StateNeedsConfirm))
}

func TestGitHubMismatchWaitsForConfirmation(t *testing.T) {
	g := newGitHubFixture(t)
	st := g.awaitConfirmation()
	if len(g.final) != 0 {
		t.Fatal("installed before the user decided")
	}
	g.s.Confirm(st.Items[0].ID)
	st = g.wait("done", g.item(StateDone))
	if len(g.final) != 1 || len(g.staged) != 1 || st.Items[0].Unverified {
		t.Errorf("installed %+v staged %+v item %+v", g.final, g.staged, st.Items[0])
	}

	if _, err := g.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "stardew", Profile: "p2", Repo: "me/mod"}}); err != nil {
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
	g.add(ghReq("2.0.0"))
	st := g.wait("done", g.item(StateDone))
	if !st.Items[0].Unverified || len(g.final) != 1 {
		t.Errorf("item %+v installed %+v", st.Items[0], g.final)
	}
}

func TestGitHubRateLimitPausesUntilTheReset(t *testing.T) {
	g := newGitHubFixture(t)
	g.limited.Store(true)
	g.start()
	g.add(ghReq("2.0.0"))
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
		if _, err := g.s.Add(t.Context(), []Request{r}); err == nil {
			t.Errorf("repo %q was queued", repo)
		}
	}
	if _, err := g.s.Add(t.Context(), []Request{ghReq("")}); err != nil {
		t.Fatalf("signed out: %v", err)
	}
}

func TestConfirmSurvivesARestartAndItsInstallIgnoresCancel(t *testing.T) {
	g := newGitHubFixture(t)
	id := g.awaitConfirmation().Items[0].ID

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
	wait := Run(ctx, again, make(chan nxmsvc.Assignment))
	t.Cleanup(func() {
		cancel()
		wait()
	})
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

func TestAStagedKeyIsKeptAndALostOneDownloadsAgain(t *testing.T) {
	g := newGitHubFixture(t)
	id := g.awaitConfirmation().Items[0].ID
	if keys := g.s.StagedKeys()["stardew"]; len(keys) != 1 || keys[0] != "github-key" {
		t.Fatalf("staged keys %v", keys)
	}
	install := g.s.d.InstallStaged
	g.s.d.InstallStaged = func(game, _, key string, _ profile.Source) (profile.InstallResult, error) {
		return profile.InstallResult{}, &store.Error{Game: game, Key: key, Err: store.ErrNotFound}
	}
	g.s.Confirm(id)
	g.wait("the failure", g.item(StateFailed))
	if keys := g.s.StagedKeys(); len(keys) != 0 {
		t.Fatalf("a lost staged key is still kept: %v", keys)
	}
	g.s.d.InstallStaged = install
	g.ok.Store(true)
	g.s.Retry(id)
	g.wait("done", g.item(StateDone))
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.staged) != 2 {
		t.Fatalf("downloaded %d times, want 2", len(g.staged))
	}
}

func TestSkippingAConfirmationReleasesItsStagedKey(t *testing.T) {
	g := newGitHubFixture(t)
	id := g.awaitConfirmation().Items[0].ID
	g.s.Skip(id)
	if keys := g.s.StagedKeys(); len(keys) != 0 {
		t.Fatalf("a skipped item keeps its staged key: %v", keys)
	}
}

func TestAddKeepsGitHubTagAndAssetDistinct(t *testing.T) {
	g := newGitHubFixture(t)
	first, err := g.s.Add(t.Context(), []Request{
		{Kind: KindInstall, Game: "stardew", Profile: "p1", Repo: "me/mod", Tag: "v1.0.0", Asset: "a.zip"},
		{Kind: KindInstall, Game: "stardew", Profile: "p1", Repo: "me/mod", Tag: "v1.0.0", Asset: "b.zip"},
		{Kind: KindInstall, Game: "stardew", Profile: "p1", Repo: "me/mod", Tag: "v2.0.0", Asset: "a.zip"},
		{Kind: KindInstall, Game: "stardew", Profile: "p1", Repo: "me/mod", Tag: "v1.0.0"},
	})
	if err != nil || len(first) != 4 {
		t.Fatalf("queued %d, %v, want 4 distinct GitHub rows", len(first), err)
	}
	again, err := g.s.Add(t.Context(), []Request{
		{Kind: KindInstall, Game: "stardew", Profile: "p1", Repo: "me/mod", Tag: "v1.0.0", Asset: "a.zip"},
		{Kind: KindInstall, Game: "stardew", Profile: "p1", Repo: "me/mod", Tag: "v1.0.0"},
	})
	if err != nil || len(again) != 2 || again[0].ID != first[0].ID || again[1].ID != first[3].ID {
		t.Fatalf("dedup = %+v, %v", again, err)
	}
}

func TestAClickBoundUpdateUsesTheSameGitHubVersionInstead(t *testing.T) {
	g := newGitHubFixture(t)
	add := func(id, version string) Item {
		it := &Item{ID: id, Kind: KindUpdate, ModID: 6304, Version: version, FallbackRepo: "me/mod", State: StateWaitingClick}
		g.s.mu.Lock()
		g.s.items = append(g.s.items, it)
		g.s.mu.Unlock()
		return *it
	}
	if !g.s.useGitHubFallback(t.Context(), add("hit", "2.0.0")) {
		t.Fatal("v2.0.0 has one archive; the update should switch to GitHub")
	}
	if g.s.useGitHubFallback(t.Context(), add("miss", "3.0.0")) {
		t.Fatal("no 3.0.0 release; the update should keep waiting for the click")
	}
	g.multi.Store(true)
	g.s.d.GitHub.CacheDir = t.TempDir()
	if g.s.useGitHubFallback(t.Context(), add("two", "2.0.0")) {
		t.Fatal("two archives cannot be chosen for the user")
	}
	g.s.mu.Lock()
	defer g.s.mu.Unlock()
	hit, miss := g.s.find("hit"), g.s.find("miss")
	if hit.Repo != "me/mod" || hit.Tag != "v2.0.0" || hit.Asset != "mod-2.0.0.zip" || hit.State != StateQueued {
		t.Fatalf("switched item %+v", *hit)
	}
	if miss.Repo != "" || miss.State != StateWaitingClick || !miss.fallbackTried {
		t.Fatalf("missed item %+v", *miss)
	}
}

func TestAGitHubFallbackWithoutTheModGoesBackToNexus(t *testing.T) {
	g := newGitHubFixture(t)
	run := func(id, fallbackID string) Item {
		it := &Item{
			ID: id, Kind: KindUpdate, ModID: 6304, Version: "2.0.0", State: StateDownloading,
			Repo: "me/mod", Tag: "v2.0.0", Asset: "mod-2.0.0.zip", FileName: "mod-2.0.0.zip",
			FallbackRepo: "me/mod", FallbackID: mod.SMAPI(fallbackID), nexusFileName: "Mod-6304-2-0-0.zip", fallbackTried: true,
		}
		g.s.mu.Lock()
		g.s.items = append(g.s.items, it)
		g.s.mu.Unlock()
		if err := g.s.downloadGitHub(t.Context(), *it); err != nil {
			t.Fatal(err)
		}
		g.s.mu.Lock()
		defer g.s.mu.Unlock()
		return *g.s.find(id)
	}
	g.ok.Store(true)
	other := run("other", "someone.else")
	if other.Repo != "" || other.State != StateQueued || other.FileName != "Mod-6304-2-0-0.zip" || len(g.final) != 0 {
		t.Fatalf("a release without the mod must go back to Nexus: %+v, installed %+v", other, g.final)
	}
	if same := run("same", "ME.MOD"); same.Repo != "me/mod" || len(g.final) != 1 {
		t.Fatalf("a release holding the mod installs from GitHub: %+v, installed %+v", same, g.final)
	}
}
