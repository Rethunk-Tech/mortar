package sharehost

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/andybalholm/brotli"
)

type shareResp struct {
	Code    string `json:"code"`
	Token   string `json:"token"`
	Version int    `json:"version"`
	Game    string `json:"game"`
	Payload string `json:"payload"`
}

func TestShares(t *testing.T) {
	payload := mustPayload(t, "stardew-valley")
	payload2 := mustPayload(t, "lethal-company")

	t.Run("create/get/versions", func(t *testing.T) {
		h := New(NewMemStore(), WithTrustedProxyIP())
		code, token := postShare(t, h, payload, "10.0.0.1")
		got := getShare(t, h, "/api/shares/"+code)
		if got.Code != code || got.Game != "stardew-valley" || got.Payload != payload || got.Version != 1 {
			t.Fatalf("latest: %#v", got)
		}
		rec := do(t, h, http.MethodPut, "/api/shares/"+code, `{"payload":`+jsonStr(payload2)+`}`, token, "10.0.0.1")
		if rec.Code != http.StatusOK {
			t.Fatalf("put %d %s", rec.Code, rec.Body.String())
		}
		var put struct {
			Version int `json:"version"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &put); err != nil || put.Version != 2 {
			t.Fatalf("put body %s", rec.Body.String())
		}
		latest := getShare(t, h, "/api/shares/"+code)
		if latest.Payload != payload2 || latest.Version != 2 || latest.Game != "lethal-company" {
			t.Fatalf("latest after put %#v", latest)
		}
		v1 := getShare(t, h, "/api/shares/"+code+"/1")
		if v1.Payload != payload || v1.Version != 1 {
			t.Fatalf("v1 %#v", v1)
		}
		v2 := getShare(t, h, "/api/shares/"+code+"/2")
		if v2.Payload != payload2 {
			t.Fatalf("v2 %#v", v2)
		}
	})

	t.Run("wrong token 403", func(t *testing.T) {
		h := New(NewMemStore(), WithTrustedProxyIP())
		code, _ := postShare(t, h, payload, "10.0.0.2")
		rec := do(t, h, http.MethodPut, "/api/shares/"+code, `{"payload":`+jsonStr(payload)+`}`, "not-the-token-value-at-all-0123456789abc", "10.0.0.2")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d", rec.Code)
		}
		missing := do(t, h, http.MethodPut, "/api/shares/zzzzzzzzzz", `{"payload":`+jsonStr(payload)+`}`, "any", "10.0.0.2")
		if missing.Code != http.StatusNotFound {
			t.Fatalf("unknown code %d", missing.Code)
		}
	})

	t.Run("oversize 413", func(t *testing.T) {
		h := New(NewMemStore(), WithTrustedProxyIP())
		big := strings.Repeat("a", maxPayload+1)
		rec := do(t, h, http.MethodPost, "/api/shares", `{"payload":"`+big+`"}`, "", "10.0.0.3")
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("got %d", rec.Code)
		}
	})

	t.Run("rate limit 429", func(t *testing.T) {
		h := New(NewMemStore(), WithTrustedProxyIP())
		for range createsPerHour {
			rec := do(t, h, http.MethodPost, "/api/shares", `{"payload":`+jsonStr(payload)+`}`, "", "10.0.0.4")
			if rec.Code != http.StatusCreated {
				t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
			}
		}
		rec := do(t, h, http.MethodPost, "/api/shares", `{"payload":`+jsonStr(payload)+`}`, "", "10.0.0.4")
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("want 429 got %d", rec.Code)
		}
		other := do(t, h, http.MethodPost, "/api/shares", `{"payload":`+jsonStr(payload)+`}`, "", "10.0.0.5")
		if other.Code != http.StatusCreated {
			t.Fatalf("other ip %d", other.Code)
		}
	})

	t.Run("bad payload 400", func(t *testing.T) {
		h := New(NewMemStore(), WithTrustedProxyIP())
		rec := do(t, h, http.MethodPost, "/api/shares", `{"payload":"not-a-share"}`, "", "10.0.0.6")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got %d", rec.Code)
		}
	})

	t.Run("sweep deletes stale and keeps fresh", func(t *testing.T) {
		st := NewMemStore()
		ctx := t.Context()
		now := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
		cutoff := now.Add(-retainFor)
		mustCreate(t, st, "stale00001", Meta{TokenHash: "aa", Latest: 1, Created: now.Add(-400 * 24 * time.Hour)}, payload)
		mustCreate(t, st, "fresh00001", Meta{TokenHash: "bb", Latest: 1, Created: now.Add(-10 * 24 * time.Hour)}, payload)
		mustCreate(t, st, "readold001", Meta{
			TokenHash: "cc", Latest: 1,
			Created:  now.Add(-400 * 24 * time.Hour),
			LastRead: now.Add(-10 * 24 * time.Hour),
		}, payload)
		deleted, kept, err := st.Sweep(ctx, cutoff)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 1 || kept != 2 {
			t.Fatalf("deleted %d kept %d", deleted, kept)
		}
		if _, err := st.Meta(ctx, "stale00001"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("stale still present: %v", err)
		}
		if _, err := st.Meta(ctx, "fresh00001"); err != nil {
			t.Fatalf("fresh gone: %v", err)
		}
		if _, err := st.Meta(ctx, "readold001"); err != nil {
			t.Fatalf("read-old gone: %v", err)
		}
	})

	t.Run("lastRead throttling", func(t *testing.T) {
		st := NewMemStore()
		h := New(st, WithTrustedProxyIP())
		t0 := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
		h.now = func() time.Time { return t0 }
		code, _ := postShare(t, h, payload, "10.0.0.7")
		getShare(t, h, "/api/shares/"+code)
		m, err := st.Meta(t.Context(), code)
		if err != nil {
			t.Fatal(err)
		}
		if !m.LastRead.Equal(t0) {
			t.Fatalf("first lastRead %v", m.LastRead)
		}
		h.now = func() time.Time { return t0.Add(time.Hour) }
		getShare(t, h, "/api/shares/"+code)
		m, err = st.Meta(t.Context(), code)
		if err != nil {
			t.Fatal(err)
		}
		if !m.LastRead.Equal(t0) {
			t.Fatalf("throttled lastRead %v", m.LastRead)
		}
		h.now = func() time.Time { return t0.Add(25 * time.Hour) }
		getShare(t, h, "/api/shares/"+code)
		m, err = st.Meta(t.Context(), code)
		if err != nil {
			t.Fatal(err)
		}
		if !m.LastRead.Equal(t0.Add(25 * time.Hour)) {
			t.Fatalf("refreshed lastRead %v", m.LastRead)
		}
	})
}

func mustCreate(t *testing.T, st *MemStore, code string, meta Meta, payload string) {
	t.Helper()
	if err := st.Create(t.Context(), code, meta, Version{Payload: payload, Game: "stardew-valley"}); err != nil {
		t.Fatal(err)
	}
}

func postShare(t *testing.T, h http.Handler, payload, ip string) (code, token string) {
	t.Helper()
	rec := do(t, h, http.MethodPost, "/api/shares", `{"payload":`+jsonStr(payload)+`}`, "", ip)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	var got shareResp
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Code) != codeLen || len(got.Token) != 43 || got.Version != 1 {
		t.Fatalf("create body %#v", got)
	}
	return got.Code, got.Token
}

func getShare(t *testing.T, h http.Handler, path string) shareResp {
	t.Helper()
	rec := do(t, h, http.MethodGet, path, "", "", "10.0.0.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s %d %s", path, rec.Code, rec.Body.String())
	}
	var got shareResp
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func do(t *testing.T, h http.Handler, method, path, body, token, ip string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if ip != "" {
		req.Header.Set("Do-Connecting-Ip", ip)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func mustPayload(t *testing.T, game string) string {
	t.Helper()
	raw := fmt.Sprintf(`[3,"Test Profile","%s",{},[]]`, game)
	var buf bytes.Buffer
	w := brotli.NewWriter(&buf)
	if _, err := w.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	p := base64.RawURLEncoding.EncodeToString(buf.Bytes())
	s, err := share.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if s.Game != game {
		t.Fatalf("parsed game %q", s.Game)
	}
	return p
}

func TestReadsNeverRollBackLatest(t *testing.T) {
	payload := mustPayload(t, "stardew-valley")
	h := New(NewMemStore())
	var tick atomic.Int64
	h.now = func() time.Time { return time.Unix(0, 0).Add(time.Duration(tick.Add(1)) * 24 * time.Hour) }
	code, token := postShare(t, h, payload, "")
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					do(t, h, http.MethodGet, "/api/shares/"+code, "", "", "")
				}
			}
		})
	}
	const puts = 30
	for range puts {
		if rec := do(t, h, http.MethodPut, "/api/shares/"+code, `{"payload":`+jsonStr(payload)+`}`, token, ""); rec.Code != http.StatusOK {
			t.Fatalf("put %d", rec.Code)
		}
	}
	close(stop)
	wg.Wait()
	if got := getShare(t, h, "/api/shares/"+code); got.Version != puts+1 {
		t.Fatalf("latest rolled back to %d, want %d", got.Version, puts+1)
	}
}

func TestForgedClientIPHeaderIgnoredByDefault(t *testing.T) {
	payload := mustPayload(t, "stardew-valley")
	h := New(NewMemStore())
	limited := false
	for i := range createsPerHour + 1 {
		if do(t, h, http.MethodPost, "/api/shares", `{"payload":`+jsonStr(payload)+`}`, "", fmt.Sprintf("10.0.%d.1", i)).Code == http.StatusTooManyRequests {
			limited = true
		}
	}
	if !limited {
		t.Fatal("rotating Do-Connecting-Ip evaded the limit")
	}
}
