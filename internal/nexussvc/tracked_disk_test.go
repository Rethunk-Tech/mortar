package nexussvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
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
	var slow atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/users/validate.json":
			_, _ = w.Write([]byte(`{"user_id":7,"name":"Ada","is_premium":false}`))
		case "/v1/user/tracked_mods.json":
			if slow.Load() {
				time.Sleep(300 * time.Millisecond)
			}
			list, _ := body.Load().(string)
			_, _ = w.Write([]byte(list))
		}
	}))
	defer srv.Close()
	c := nexus.New("1")
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

	slow.Store(true)
	body.Store(`[{"mod_id":100,"domain_name":"stardewvalley"},{"mod_id":200,"domain_name":"stardewvalley"}]`)
	c2 := nexus.New("1") // no in-memory copy: only the disk can answer quickly
	c2.BaseURL = srv.URL
	s.client = c2
	start := time.Now()
	mods, err := s.trackedForCard(ctx)
	if err != nil || len(mods) != 1 || time.Since(start) > 100*time.Millisecond {
		t.Fatalf("kept list at once: %v %v after %v", mods, err, time.Since(start))
	}
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
