package queue

import (
	"context" // #nosec G501 -- Content-MD5 is MD5 by RFC 1864
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/go-httpkit/download"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/ids"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// serveRange answers only a resume from byte from, as the CDN does for a partial it handed out.
func serveRange(w http.ResponseWriter, r *http.Request, body string, from int) {
	if r.Header.Get("Range") != fmt.Sprintf("bytes=%d-", from) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", from, len(body)-1, len(body)))
	w.WriteHeader(http.StatusPartialContent)
	_, _ = io.WriteString(w, body[from:])
}

// seedPartial leaves partial at path with the sidecar a stopped download writes.
func seedPartial(t *testing.T, path, partial string, resume map[string]any) {
	t.Helper()
	if err := os.WriteFile(path, []byte(partial), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = datadir.WriteJSON(path+".resume.json", resume)
}

func TestTruncatedPartialResumesAfterRestart(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	const body = payload
	half := len(body) / 2
	f.cdn = func(w http.ResponseWriter, r *http.Request) { serveRange(w, r, body, half) }
	f.s.Pause()
	id := f.add(req(10))[0].ID
	if err := os.MkdirAll(filepath.Join(f.dir, downloadsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.dir, downloadsDir, id+filepath.Ext("a-1.0.zip"))
	seedPartial(t, path, body[:half], map[string]any{"expectedSize": int64(len(body)), "url": "stale"})
	f.s.mu.Lock()
	f.s.items[0].State = StateDownloading
	f.s.items[0].FileName = "a-1.0.zip"
	f.s.mu.Unlock()
	f.s.publish(true)
	again, err := New(f.s.d)
	if err != nil {
		t.Fatal(err)
	}
	f.s = again
	f.start()
	f.s.Resume()
	f.wait("done", f.item(StateDone))
	if len(f.installs) != 1 {
		t.Fatalf("installs %d", len(f.installs))
	}
	f.leftovers()
}

func TestExpiredNexusLinkRefetchesThenRanges(t *testing.T) {
	t.Parallel()
	const body = payload
	half := len(body) / 2
	var links atomic.Int32
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/games/stardewvalley/mods/1/files.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Rl-Daily-Remaining", "500")
		w.Header().Set("X-Rl-Hourly-Remaining", "100")
		fmt.Fprint(w, `{"files":[{"file_id":10,"file_name":"a-1.0.zip","version":"1.0","category_name":"MAIN","size_kb":3,"is_primary":true}]}`)
	})
	mux.HandleFunc("/v1/games/stardewvalley/mods/1.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Rl-Daily-Remaining", "500")
		w.Header().Set("X-Rl-Hourly-Remaining", "100")
		fmt.Fprint(w, `{"name":"Alpha"}`)
	})
	mux.HandleFunc("/v1/games/stardewvalley/mods/1/files/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Rl-Daily-Remaining", "500")
		w.Header().Set("X-Rl-Hourly-Remaining", "100")
		n := links.Add(1)
		if n == 1 {
			fmt.Fprintf(w, `[{"name":"CDN","URI":%q}]`, srv.URL+"/cdn/old.zip")
			return
		}
		fmt.Fprintf(w, `[{"name":"CDN","URI":%q}]`, srv.URL+"/cdn/new.zip")
	})
	mux.HandleFunc("/cdn/old.zip", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("/cdn/new.zip", func(w http.ResponseWriter, r *http.Request) { serveRange(w, r, body, half) })
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, downloadsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	c := nexusmods.New("Mortar", "test")
	c.BaseURL, c.HTTP, c.Now = srv.URL, srv.Client(), func() time.Time { return time.Unix(1_000_000, 0).UTC() }
	client := c.WithKey("secret")
	var installs int
	s, err := New(Deps{
		Client:  func() (*nexusmods.Client, error) { return client, nil },
		Premium: func() bool { return true },
		Install: func(_ context.Context, _, _, path string, _ profile.Source) (profile.InstallResult, error) {
			b, err := safefs.ReadFile(path)
			if err != nil || string(b) != body {
				return profile.InstallResult{}, fmt.Errorf("installer read %q: %w", b, err)
			}
			installs++
			return profile.InstallResult{}, nil
		},
		HTTP: srv.Client(), Dir: dir, Now: func() time.Time { return time.Unix(1_000_000, 0).UTC() },
	})
	if err != nil {
		t.Fatal(err)
	}
	items, err := s.Add(t.Context(), []Request{req(10)})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, downloadsDir, items[0].ID+filepath.Ext("a-1.0.zip"))
	seedPartial(t, path, body[:half], map[string]any{"expectedSize": int64(len(body)), "url": srv.URL + "/cdn/old.zip"})

	ctx, cancel := context.WithCancel(context.Background())
	wait := Run(ctx, s, make(chan nxmsvc.Assignment))
	t.Cleanup(func() {
		cancel()
		wait()
	})
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st := s.State(); len(st.Items) > 0 && st.Items[0].State == StateDone && installs == 1 {
			if links.Load() < 2 {
				t.Fatalf("link fetch count %d", links.Load())
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out: %+v installs %d links %d", s.State(), installs, links.Load())
}

func TestSweepKeepsPartialsForQueuedItems(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	root := filepath.Join(f.dir, downloadsDir)
	if err := os.MkdirAll(filepath.Join(root, "My Folder"), 0o700); err != nil {
		t.Fatal(err)
	}
	live, gone := ids.New(), ids.New()
	keep := filepath.Join(root, live+".zip")
	drop := filepath.Join(root, gone+".zip")
	sidecar := download.Sidecar(drop)
	foreign := []string{filepath.Join(root, "holiday.jpg"), filepath.Join(root, "0123456789abcdef.pdf"), filepath.Join(root, "My Folder", "x.txt")}
	for _, p := range append([]string{keep, drop, sidecar}, foreign...) {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f.s.items = []*Item{{ID: live, State: StateQueued}}
	f.s.sweepDownloads()
	for _, p := range append([]string{keep}, foreign...) {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s was removed", p)
		}
	}
	for _, p := range []string{drop, sidecar} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s was kept", p)
		}
	}
}

func TestDownloadRootUsesSeparateFolder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	custom := t.TempDir()
	s := &Service{d: Deps{Dir: dir, DownloadDir: func() string { return custom }}}
	if s.downloadRoot() != custom {
		t.Fatalf("root = %s", s.downloadRoot())
	}
	got := s.dest("item", "m.zip")
	want := filepath.Join(custom, "item.zip")
	if got != want {
		t.Fatalf("dest = %s want %s", got, want)
	}
	plain := &Service{d: Deps{Dir: dir}}
	if plain.downloadRoot() != filepath.Join(dir, downloadsDir) {
		t.Fatalf("default root = %s", plain.downloadRoot())
	}
}

func TestDestKeepsABareCompressedDownloadsName(t *testing.T) {
	t.Parallel()
	s := &Service{d: Deps{Dir: t.TempDir()}}
	got := s.dest("item", "Cool Mod.dll.gz")
	if filepath.Base(got) != "item~~Cool Mod.dll.gz" {
		t.Fatalf("dest = %s", got)
	}
	if err := os.MkdirAll(filepath.Dir(got), 0o700); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join("testdata", "bare.txt.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(got, src, 0o600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := archive.Extract(got, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "Cool Mod.dll")); err != nil {
		t.Fatalf("extracted under the queue id, not the file's name: %v", err)
	}
}

func TestSweepRemovesASidecarTempLeftByACrash(t *testing.T) {
	s := &Service{d: Deps{Dir: t.TempDir()}}
	root := s.downloadRoot()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"0123456789abcdef.zip.resume.json.123456.tmp", "0123456789abcdef~~a.gz.resume.json.99.tmp"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// The temp file safefs.WriteFileAtomic leaves beside the sidecar download.File writes: "<base>.<random>.tmp".
	crashed, err := os.CreateTemp(root, filepath.Base(download.Sidecar("0123456789abcdef.zip"))+".*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	_ = crashed.Close()
	s.sweepDownloads()
	left, _ := os.ReadDir(root)
	if len(left) != 0 {
		t.Fatalf("sweep left %d files", len(left))
	}
}
