package nexussvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

type emitHost struct {
	winhost.Host
	events chan string
}

func (h emitHost) Emit(name string, _ any) { h.events <- name }

func TestTrackedForCardAnswersFromDiskAndRefreshesInBackground(t *testing.T) {
	store := testStore(t)
	var body atomic.Value
	body.Store(`[{"mod_id":100,"domain_name":"stardewvalley"}]`)
	// While held, Nexus does not answer the tracked list: a read that returns meanwhile came from disk.
	held := make(chan struct{})
	var hold atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/users/validate.json":
			_, _ = w.Write([]byte(`{"user_id":7,"name":"Ada","is_premium":false}`))
		case "/v1/user/tracked_mods.json":
			if hold.Load() {
				<-held
			}
			list, _ := body.Load().(string)
			_, _ = w.Write([]byte(list))
		}
	}))
	defer srv.Close()
	// Runs before srv.Close, which waits for the held request.
	release := sync.OnceFunc(func() { close(held) })
	defer release()
	c := nexusmods.New("Mortar", "1")
	c.BaseURL = srv.URL
	s := NewService(store, c, &meta.Client{})
	events := make(chan string, 4)
	s.App = emitHost{events: events}
	if err := s.UseDataDir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := s.SignIn(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	<-events // sign-in's account event
	if mods, err := s.trackedForCard(ctx); err != nil || len(mods) != 1 {
		t.Fatalf("first read waits for Nexus: %v %v", mods, err)
	}

	hold.Store(true)
	body.Store(`[{"mod_id":100,"domain_name":"stardewvalley"},{"mod_id":200,"domain_name":"stardewvalley"}]`)
	c2 := nexusmods.New("Mortar", "1") // no in-memory copy: only the disk can answer quickly
	c2.BaseURL = srv.URL
	s.client = c2
	read := make(chan []nexusmods.TrackedMod, 1)
	go func() {
		mods, _ := s.trackedForCard(ctx)
		read <- mods
	}()
	select {
	case mods := <-read:
		if len(mods) != 1 {
			t.Fatalf("kept list: %v", mods)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the read waited for Nexus instead of answering from disk")
	}
	release()
	select {
	case name := <-events:
		if name != TrackedEvent {
			t.Fatalf("event %q", name)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no refresh event")
	}
	if mods, _ := s.trackedForCard(ctx); len(mods) != 2 {
		t.Fatalf("refreshed list kept: %v", mods)
	}

	for s.refreshing.Load() {
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := s.SignOut(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.trackedPath); !os.IsNotExist(err) {
		t.Fatalf("sign-out leaves the list: %v", err)
	}
}
