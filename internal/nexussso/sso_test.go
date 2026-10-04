package nexussso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func wsURL(srv *httptest.Server) string { return "ws" + strings.TrimPrefix(srv.URL, "http") }

func send(ctx context.Context, t *testing.T, c *websocket.Conn, v any) {
	t.Helper()
	b, _ := json.Marshal(v)
	if err := c.Write(ctx, websocket.MessageText, b); err != nil {
		t.Error(err)
	}
}

func TestLegacyHappyPathAndReconnect(t *testing.T) {
	var conns atomic.Int32
	var hellos [2]map[string]any
	approve := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		n := conns.Add(1)
		_, raw, _ := c.Read(r.Context())
		var h map[string]any
		_ = json.Unmarshal(raw, &h)
		hellos[n-1] = h
		if n == 1 {
			send(r.Context(), t, c, map[string]any{"success": true, "data": map[string]any{"connection_token": "tok"}})
			_ = c.CloseNow()
			return
		}
		<-approve
		send(r.Context(), t, c, map[string]any{"success": true, "data": map[string]any{"api_key": "KEY"}})
		_ = c.Close(websocket.StatusNormalClosure, "")
	}))
	defer srv.Close()

	var opened []string
	var states []State
	l := Legacy{
		Slug: "mortar", SocketURL: wsURL(srv), PageURL: "https://example.test/sso", RetryDelay: time.Millisecond,
		OpenBrowser: func(u string) error { opened = append(opened, u); close(approve); return nil },
		OnState:     func(s State) { states = append(states, s) },
	}
	key, err := l.Run(context.Background())
	if err != nil || key != "KEY" {
		t.Fatalf("key %q err %v", key, err)
	}
	if hellos[0]["token"] != nil || hellos[0]["protocol"] != float64(2) || hellos[1]["token"] != "tok" || hellos[0]["id"] != hellos[1]["id"] {
		t.Fatalf("hellos %v", hellos)
	}
	if len(opened) != 1 {
		t.Fatalf("browser opened %d times", len(opened))
	}
	u, _ := url.Parse(opened[0])
	if u.Query().Get("application") != "mortar" || u.Query().Get("id") != hellos[0]["id"] {
		t.Fatalf("page url %s", opened[0])
	}
	if len(states) < 2 || states[0] != Connected || states[1] != Waiting {
		t.Fatalf("states %v", states)
	}
}

func holdOpen(t *testing.T, reply any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		_, _, _ = c.Read(r.Context())
		send(r.Context(), t, c, reply)
		_, _, _ = c.Read(r.Context())
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLegacyTimeoutCancelRefusal(t *testing.T) {
	ok := map[string]any{"success": true, "data": map[string]any{"connection_token": "t"}}
	base := Legacy{Slug: "s", OpenBrowser: func(string) error { return nil }}

	l := base
	l.SocketURL, l.Timeout = wsURL(holdOpen(t, ok)), 100*time.Millisecond
	if _, err := l.Run(context.Background()); !errors.Is(err, ErrTimeout) {
		t.Fatalf("timeout: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	l = base
	l.SocketURL = wsURL(holdOpen(t, ok))
	l.OnState = func(s State) {
		if s == Waiting {
			cancel()
		}
	}
	if _, err := l.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}

	l = base
	l.SocketURL = wsURL(holdOpen(t, map[string]any{"success": false, "error": "bad slug"}))
	if _, err := l.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "bad slug") {
		t.Fatalf("refusal: %v", err)
	}

	if _, err := (Legacy{}).Run(context.Background()); !errors.Is(err, ErrNoSlug) {
		t.Fatalf("no slug: %v", err)
	}
}
