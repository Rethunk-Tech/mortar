package queue

import (
	"context"
	"crypto/md5" // #nosec G501 -- Content-MD5 is MD5 by RFC 1864
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func rangeSvc(t *testing.T, client *http.Client) *Service {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, downloadsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	return &Service{
		d:     Deps{HTTP: client, Dir: dir, Now: func() time.Time { return time.Unix(1_000_000, 0).UTC() }},
		items: []*Item{{ID: "item", State: StateDownloading}},
	}
}

func TestFetchResumesWithRangeWhenTheServerSupportsIt(t *testing.T) {
	const body = "abcdefghij0123456789"
	var sawRange atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "bytes=10-" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		sawRange.Store(true)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 10-19/%d", len(body)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, body[10:])
	}))
	t.Cleanup(srv.Close)

	s := rangeSvc(t, srv.Client())
	path := s.dest("item", "m.zip")
	if err := os.WriteFile(path, []byte(body[:10]), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = datadir.WriteJSON(path+".resume.json", map[string]any{"expectedSize": int64(len(body)), "etag": `"v1"`, "url": srv.URL})

	if err := s.fetch(context.Background(), Item{ID: "item"}, srv.URL, path); err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(path)
	if err != nil || string(got) != body || !sawRange.Load() {
		t.Fatalf("got %q range %v err %v", got, sawRange.Load(), err)
	}
}

func TestFetchStartsOverWhenTheServerIgnoresRange(t *testing.T) {
	const body = "abcdefghij0123456789"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	s := rangeSvc(t, srv.Client())
	path := s.dest("item", "m.zip")
	if err := os.WriteFile(path, []byte("PARTIAL!!"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = datadir.WriteJSON(path+".resume.json", map[string]any{"expectedSize": int64(len(body)), "url": srv.URL})

	if err := s.fetch(context.Background(), Item{ID: "item"}, srv.URL, path); err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(path)
	if err != nil || string(got) != body {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestFetchVerifiesAChecksumFromTheSource(t *testing.T) {
	const body = "abcdefghij0123456789"
	sum := sha256.Sum256([]byte(body))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Checksum-Sha256", hex.EncodeToString(sum[:]))
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	s := rangeSvc(t, srv.Client())
	path := s.dest("item", "m.zip")
	if err := s.fetch(context.Background(), Item{ID: "item"}, srv.URL, path); err != nil {
		t.Fatal(err)
	}
}

func TestFetchContentMD5(t *testing.T) {
	const body = "abcdefghij0123456789"
	sum := md5.Sum([]byte(body)) // #nosec G401 -- Content-MD5 is MD5 by RFC 1864
	ok := base64.StdEncoding.EncodeToString(sum[:])

	t.Run("match", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-MD5", ok)
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			_, _ = io.WriteString(w, body)
		}))
		t.Cleanup(srv.Close)
		s := rangeSvc(t, srv.Client())
		path := s.dest("item", "m.zip")
		if err := s.fetch(context.Background(), Item{ID: "item"}, srv.URL, path); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-MD5", base64.StdEncoding.EncodeToString(make([]byte, md5.Size)))
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			_, _ = io.WriteString(w, body)
		}))
		t.Cleanup(srv.Close)
		s := rangeSvc(t, srv.Client())
		path := s.dest("item", "m.zip")
		err := s.fetch(context.Background(), Item{ID: "item"}, srv.URL, path)
		if err == nil || !strings.Contains(err.Error(), "Content-MD5") {
			t.Fatalf("err = %v", err)
		}
		if _, stat := os.Stat(path); !os.IsNotExist(stat) {
			t.Fatal("mismatch left the partial on disk")
		}
	})

	t.Run("absent", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			_, _ = io.WriteString(w, body)
		}))
		t.Cleanup(srv.Close)
		s := rangeSvc(t, srv.Client())
		path := s.dest("item", "m.zip")
		if err := s.fetch(context.Background(), Item{ID: "item"}, srv.URL, path); err != nil {
			t.Fatal(err)
		}
	})
}

func TestTruncatedPartialResumesAfterRestart(t *testing.T) {
	f := newFixture(t)
	const body = payload
	half := len(body) / 2
	f.cdn = func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != fmt.Sprintf("bytes=%d-", half) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", half, len(body)-1, len(body)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, body[half:])
	}
	f.s.Pause()
	items, err := f.s.Add(t.Context(), []Request{req(10)})
	if err != nil {
		t.Fatal(err)
	}
	id := items[0].ID
	if err := os.MkdirAll(filepath.Join(f.dir, downloadsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.dir, downloadsDir, id+filepath.Ext("a-1.0.zip"))
	if err := os.WriteFile(path, []byte(body[:half]), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = datadir.WriteJSON(path+".resume.json", map[string]any{"expectedSize": int64(len(body)), "url": "stale"})
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
	mux.HandleFunc("/cdn/new.zip", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != fmt.Sprintf("bytes=%d-", half) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", half, len(body)-1, len(body)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, body[half:])
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, downloadsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	c := nexus.New("test")
	c.BaseURL, c.CacheDir, c.HTTP, c.Now = srv.URL, t.TempDir(), srv.Client(), func() time.Time { return time.Unix(1_000_000, 0).UTC() }
	client := c.WithKey("secret")
	var installs int
	s, err := New(Deps{
		Client:  func() (*nexus.Client, error) { return client, nil },
		Premium: func() bool { return true },
		Install: func(_, _, path string, _ profile.Source) (profile.InstallResult, error) {
			b, err := fsx.ReadFile(path)
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
	if err := os.WriteFile(path, []byte(body[:half]), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = datadir.WriteJSON(path+".resume.json", map[string]any{"expectedSize": int64(len(body)), "url": srv.URL + "/cdn/old.zip"})

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
	f := newFixture(t)
	if err := os.MkdirAll(filepath.Join(f.dir, downloadsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(f.dir, downloadsDir, "keepme"+filepath.Ext("a.zip"))
	drop := filepath.Join(f.dir, downloadsDir, "gone"+filepath.Ext("a.zip"))
	if err := os.WriteFile(keep, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(drop, []byte("y"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.s.items = []*Item{{ID: "keepme", State: StateQueued}}
	f.s.sweepDownloads()
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("kept partial was removed")
	}
	if _, err := os.Stat(drop); !os.IsNotExist(err) {
		t.Fatal("orphan partial was kept")
	}
}

func TestFetchGivesUpOnALoopingPartialRange(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.Header().Set("Content-Range", "bytes 1-9/10")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, "0123456789")
	}))
	t.Cleanup(srv.Close)
	s := rangeSvc(t, srv.Client())
	path := s.dest("item", "m.zip")
	if err := os.WriteFile(path, []byte("PARTIAL!!"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = datadir.WriteJSON(path+".resume.json", map[string]any{"expectedSize": 10, "etag": `"v1"`, "url": srv.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	err := s.fetch(ctx, Item{ID: "item"}, srv.URL, path)
	if err == nil {
		t.Fatal("expected a resume error")
	}
	if n.Load() > int32(fetchRestarts+2) {
		t.Fatalf("requests = %d, livelock", n.Load())
	}
}

func TestDownloadRootUsesSeparateFolder(t *testing.T) {
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
