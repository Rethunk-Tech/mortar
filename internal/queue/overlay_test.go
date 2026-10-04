package queue

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/nxmsvc"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/store"
)

func TestOptionalFileWaitsForItsMainFile(t *testing.T) {
	main := &Item{ID: "a", Game: "stardew", Profile: "p", ModID: 7, State: StateDownloading}
	opt := &Item{ID: "b", Game: "stardew", Profile: "p", ModID: 7, State: StateDownloading}
	other := &Item{ID: "c", Game: "stardew", Profile: "p", ModID: 8, State: StateQueued}
	s, err := New(Deps{Dir: t.TempDir(), Premium: func() bool { return true }})
	if err != nil {
		t.Fatal(err)
	}
	s.items = []*Item{main, opt, other}
	if !s.waitsForSameMod(opt) || s.waitsForSameMod(main) || s.waitsForSameMod(other) {
		t.Fatal("only the later file of the same mod waits")
	}
	opt.State, opt.FileID, opt.FileName = StateQueued, 2, "b.zip"
	if got, act, held := s.next(nil); got != opt || act != fetch || held {
		t.Fatal("a second file of the same mod waited to download")
	}
	opt.State = StateDownloading
	if s.parkOverlay("b", false, true) || opt.State != StateDownloading {
		t.Fatal("a file with a manifest of the same mod was held")
	}
	if !s.parkOverlay("b", true, true) || opt.State != StateQueued || !opt.readyZip || !opt.overlay {
		t.Fatalf("optional file not parked: %+v", opt)
	}
	other.FileID, other.FileName = 1, "o.zip"
	if got, _, _ := s.next(nil); got != other {
		t.Fatalf("next = %+v, want the other mod's file while the main file downloads", got)
	}
	main.State = StateFailed
	if s.waitsForSameMod(opt) {
		t.Fatal("a finished main file still holds its optional file")
	}
	if got, act, _ := s.next(nil); got != opt || act != install {
		t.Fatalf("next = %+v %v, want the parked optional file to install", got, act)
	}
	if got := failureKind(&profile.InstallError{Msg: "x", Err: &profile.NoBaseError{Archive: "x.zip"}}); got != FailNoBase {
		t.Fatalf("kind = %q", got)
	}
}

func TestManifestLess(t *testing.T) {
	write := func(name string, files ...string) string {
		p := filepath.Join(t.TempDir(), name)
		f, err := fsx.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		zw := zip.NewWriter(f)
		for _, n := range files {
			w, _ := zw.Create(n)
			_, _ = w.Write([]byte(`{"Name":"A","UniqueID":"a.b","Version":"1.0.0"}`))
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
		return p
	}
	if !manifestLess(write("opt.zip", "Mod/assets/a.png")) || manifestLess(write("main.zip", "Mod/manifest.json")) {
		t.Fatal("manifestLess")
	}
}

// A profile received over the local network installs its files from the store; an optional file there waits for its
// main file and is laid over it.
func TestStoredOptionalFileInstallsOverItsMainFile(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_DATA_HOME", base)
	t.Setenv("XDG_CONFIG_HOME", base)
	t.Setenv("LOCALAPPDATA", base)
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	p, err := profiles.Create("stardew", "P")
	if err != nil {
		t.Fatal(err)
	}
	add := func(key string, files map[string]string) {
		dir := t.TempDir()
		for rel, body := range files {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if err := items.AddDir("stardew", key, dir); err != nil {
			t.Fatal(err)
		}
	}
	mainKey, optKey := store.NexusKey(5, 1), store.NexusKey(5, 2)
	add(mainKey, map[string]string{
		"Mod/manifest.json": `{"Name":"M","Author":"a","Version":"1.0.0","UniqueID":"a.m"}`, "Mod/a.png": "main",
	})
	add(optKey, map[string]string{"Mod/a.png": "opt"})
	c := nexus.New("test")
	c.BaseURL, c.CacheDir = "http://127.0.0.1:1", t.TempDir()
	client := c.WithKey("k")
	s, err := New(Deps{
		Client:  func() (*nexus.Client, error) { return client, nil },
		Premium: func() bool { return true },
		Stored: func(game, key string) (profile.Source, bool) {
			_, err := items.Path(game, key)
			return profile.Source{}, err == nil
		},
		StoredOverlay: profiles.StoredOverlay,
		InstallStaged: profiles.InstallStaged,
		Dir:           t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add([]Request{
		{Kind: KindInstall, Game: "stardew", Profile: p.ID, ModID: 5, FileID: 1, FileName: "main.zip"},
		{Kind: KindInstall, Game: "stardew", Profile: p.ID, ModID: 5, FileID: 2, FileName: "opt.zip"},
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	wait := Run(ctx, s, make(chan nxmsvc.Assignment))
	defer func() {
		cancel()
		wait()
	}()
	deadline := time.Now().Add(5 * time.Second)
	for st := s.State(); st.Items[0].State != StateDone || st.Items[1].State != StateDone; st = s.State() {
		if time.Now().After(deadline) || st.Items[0].State == StateFailed || st.Items[1].State == StateFailed {
			t.Fatalf("queue = %+v", st.Items)
		}
		time.Sleep(5 * time.Millisecond)
	}
	all, err := profiles.List("stardew")
	if err != nil || len(all) != 1 || len(all[0].Entries) != 2 || all[0].Entries[1].OverlayOf != mainKey {
		t.Fatalf("profiles = %+v, %v", all, err)
	}
	modsDir, err := profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := fsx.ReadFile(filepath.Join(modsDir, mainKey, "Mod", "a.png")); err != nil || string(b) != "opt" {
		t.Fatalf("laid file = %q, %v", b, err)
	}
}
