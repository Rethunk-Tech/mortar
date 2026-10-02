package nexussvc

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/zalando/go-keyring"
)

func TestEndorseAndTrackActOnSignedInAccount(t *testing.T) {
	keyring.MockInit()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	var last string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last = r.Method + " " + r.URL.Path
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/v1/users/validate.json":
			_, _ = w.Write([]byte(`{"user_id":7,"name":"Ada","is_premium":true}`))
		case "/v1/games/stardewvalley/mods/541/endorse.json":
			if r.Method != http.MethodPost || string(body) != `{"version":"1.55.0"}` {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"message":"Updated to: Endorsed","status":"Endorsed"}`))
		case "/v1/user/tracked_mods.json":
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`[{"mod_id":541,"domain_name":"stardewvalley"}]`))
				return
			}
			_, _ = w.Write([]byte(`{"message":"ok"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	c := nexus.New("1")
	c.BaseURL = srv.URL
	s := NewService(store, c, &meta.Client{})
	ctx := context.Background()
	if _, err := s.Endorse(ctx, 541, "1.55.0"); !errors.Is(err, ErrSignedOut) {
		t.Fatalf("signed out endorse = %v", err)
	}
	if _, err := s.SignIn(ctx, "secret-key"); err != nil {
		t.Fatal(err)
	}
	status, err := s.Endorse(ctx, 541, "1.55.0")
	if err != nil || status != "Endorsed" || last != "POST /v1/games/stardewvalley/mods/541/endorse.json" {
		t.Fatalf("endorse = %q %v %s", status, err, last)
	}
	list, err := s.TrackedMods(ctx)
	if err != nil || len(list) != 1 || list[0].ModID != 541 {
		t.Fatalf("tracked = %+v, %v", list, err)
	}
	if err := s.Track(ctx, 541); err != nil {
		t.Fatal(err)
	}
	if count, err := s.TrackedCount(ctx, "stardew"); err != nil || count != 1 {
		t.Fatalf("tracked count = %d, %v", count, err)
	}
	result, err := s.UntrackAll(ctx, "stardew", false)
	if err != nil || result.Untracked != 1 || result.Remaining != 0 || result.StoppedForLimit {
		t.Fatalf("untrack all = %+v, %v", result, err)
	}
	if _, err := s.UntrackAll(ctx, "stardew", true); err == nil {
		t.Fatal("untracking only unused mods without the profile store must refuse, not untrack everything")
	}
}

func TestEndorseWithoutDownloadSurfacesNexusMessage(t *testing.T) {
	keyring.MockInit()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/users/validate.json" {
			_, _ = w.Write([]byte(`{"user_id":7,"name":"Ada","is_premium":false}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"You must download this mod before you can endorse it."}`))
	}))
	defer srv.Close()
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	c := nexus.New("1")
	c.BaseURL = srv.URL
	s := NewService(store, c, &meta.Client{})
	ctx := context.Background()
	if _, err := s.SignIn(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	_, err = s.Endorse(ctx, 541, "1.0.0")
	if err == nil || err.Error() != "You must download this mod before you can endorse it." {
		t.Fatalf("endorse = %v", err)
	}
}
