package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestEndorseAbstainAndTrackedList(t *testing.T) {
	var hits atomic.Int32
	var last struct {
		method, path, body string
	}
	now := t0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		raw, _ := io.ReadAll(r.Body)
		last.method, last.path, last.body = r.Method, r.URL.Path, string(raw)
		w.Header().Set("X-Rl-Daily-Remaining", "19000")
		w.Header().Set("X-Rl-Daily-Limit", "20000")
		w.Header().Set("X-Rl-Daily-Reset", "2026-10-01 00:00:00 +0000")
		setRate(w.Header(), "1900")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/games/stardewvalley/mods/541/endorse.json":
			_, _ = w.Write([]byte(`{"message":"Updated to: Endorsed","status":"Endorsed"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/games/stardewvalley/mods/541/abstain.json":
			_, _ = w.Write([]byte(`{"message":"Updated to: Abstained","status":"Abstained"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/user/tracked_mods.json":
			_, _ = w.Write([]byte(`[{"mod_id":541,"domain_name":"stardewvalley"},{"mod_id":1,"domain_name":"skyrim"}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/user/tracked_mods.json":
			_, _ = w.Write([]byte(`{"message":"Tracking"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/user/tracked_mods.json":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	c := New("9.9")
	c.BaseURL, c.CacheDir, c.Now = srv.URL, t.TempDir(), func() time.Time { return now }
	c = c.WithKey("k")
	ctx := context.Background()

	got, err := c.Endorse(ctx, stardew, 541, "1.55.0")
	if err != nil || got != EndorseEndorsed || last.method != http.MethodPost ||
		last.path != "/v1/games/stardewvalley/mods/541/endorse.json" {
		t.Fatalf("endorse = %q, %v, %+v", got, err, last)
	}
	var body struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(last.body), &body); err != nil || body.Version != "1.55.0" {
		t.Fatalf("endorse body = %s", last.body)
	}

	got, err = c.Abstain(ctx, stardew, 541, "1.55.0")
	if err != nil || got != EndorseAbstained || last.path != "/v1/games/stardewvalley/mods/541/abstain.json" {
		t.Fatalf("abstain = %q, %v, %+v", got, err, last)
	}

	list, err := c.TrackedMods(ctx)
	if err != nil || len(list) != 2 || list[0] != (TrackedMod{ModID: 541, DomainName: "stardewvalley"}) {
		t.Fatalf("tracked = %+v, %v", list, err)
	}
	n := hits.Load()
	if _, err := c.TrackedMods(ctx); err != nil || hits.Load() != n {
		t.Fatalf("tracked list was not cached: hits %d -> %d, %v", n, hits.Load(), err)
	}
	now = t0.Add(trackedCacheTTL + time.Second)
	if _, err := c.TrackedMods(ctx); err != nil || hits.Load() != n+1 {
		t.Fatalf("tracked cache did not expire: hits %d", hits.Load())
	}

	if err := c.Track(ctx, stardew, 541); err != nil {
		t.Fatal(err)
	}
	var track trackBody
	if err := json.Unmarshal([]byte(last.body), &track); err != nil || track != (trackBody{DomainName: stardew.Domain, ModID: 541}) ||
		last.method != http.MethodPost {
		t.Fatalf("track = %+v, %s", last, last.body)
	}
	n = hits.Load()
	if _, err := c.TrackedMods(ctx); err != nil || hits.Load() != n+1 {
		t.Fatalf("track should drop the cached list: hits %d -> %d", n, hits.Load())
	}
	if err := c.Untrack(ctx, stardew, 541); err != nil || last.method != http.MethodDelete {
		t.Fatalf("untrack = %v, %+v", err, last)
	}
}

func TestEndorseWithoutDownloadReturnsNexusMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rl-Daily-Remaining", "19000")
		w.Header().Set("X-Rl-Daily-Limit", "20000")
		w.Header().Set("X-Rl-Daily-Reset", "2026-10-01 00:00:00 +0000")
		setRate(w.Header(), "1900")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"You must download this mod before you can endorse it."}`))
	}))
	t.Cleanup(srv.Close)
	c := New("9.9")
	c.BaseURL, c.CacheDir = srv.URL, t.TempDir()
	c = c.WithKey("k")
	_, err := c.Endorse(context.Background(), stardew, 541, "1.0.0")
	var msg *MessageError
	if !errors.As(err, &msg) || msg.Message != "You must download this mod before you can endorse it." {
		t.Fatalf("endorse = %v", err)
	}
}

func TestTrackAlreadyTrackedIsOk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rl-Daily-Remaining", "19000")
		w.Header().Set("X-Rl-Daily-Limit", "20000")
		w.Header().Set("X-Rl-Daily-Reset", "2026-10-01 00:00:00 +0000")
		setRate(w.Header(), "1900")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"already tracking"}`))
	}))
	t.Cleanup(srv.Close)
	c := New("9.9")
	c.BaseURL, c.CacheDir = srv.URL, t.TempDir()
	if err := c.WithKey("k").Track(context.Background(), stardew, 541); err != nil {
		t.Fatal(err)
	}
}
