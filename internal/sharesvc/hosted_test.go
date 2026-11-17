package sharesvc

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/hostedlink"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const hostedToken = "abcdefghijklmnopqrstuvwxyz0123456789ABCDE"

func hostedService(t *testing.T) (*Service, string) {
	t.Helper()
	s, _ := newService(t, true)
	p := testenv.Profile(t, s.d.Profiles, "stardew", "Farm")
	s.d.Dir = t.TempDir()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/shares", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var in struct {
			Payload string `json:"payload"`
		}
		_ = json.Unmarshal(body, &in)
		if _, err := share.Parse(in.Payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(hostedlink.CreateResponse{Code: "AbCdEfGh12", Token: hostedToken, Version: 1})
	})
	mux.HandleFunc("PUT /api/shares/AbCdEfGh12", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+hostedToken {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hostedlink.UpdateResponse{Version: 2})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	s.host = hostedlink.Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	return s, p.ID
}

func TestCreateHostedShareStoresTokenOutsideProfileJSON(t *testing.T) {
	s, id := hostedService(t)
	got, err := s.CreateHostedShare(t.Context(), "stardew", id, nil, share.Include{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != "AbCdEfGh12" || got.Version != 1 || got.URL != "https://mortar.rethunk.tech/stardew/s/AbCdEfGh12" {
		t.Fatalf("hosted = %+v", got)
	}
	info, err := s.Share("stardew", id, nil, share.Include{})
	if err != nil || info.Hosted == nil || *info.Hosted != got {
		t.Fatalf("share hosted = %+v, %v", info.Hosted, err)
	}
	raw := testfs.Read(t, filepath.Join(s.d.Dir, "secrets", "shares", "stardew", id+".json"))
	if !strings.Contains(raw, hostedToken) {
		t.Fatal("token missing from secrets file")
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		t.Fatal("no data home")
	}
	for path, body := range testfs.Tree(t, dataHome) {
		if strings.Contains(body, hostedToken) {
			t.Errorf("token leaked into %s", path)
		}
	}
}

func TestUpdateHostedShareAndWrongToken(t *testing.T) {
	s, id := hostedService(t)
	if _, err := s.CreateHostedShare(t.Context(), "stardew", id, nil, share.Include{}); err != nil {
		t.Fatal(err)
	}
	got, err := s.UpdateHostedShare(t.Context(), "stardew", id, nil, share.Include{})
	if err != nil || got.Version != 2 {
		t.Fatalf("update = %+v, %v", got, err)
	}

	rec, found, err := s.records().Get("stardew", id)
	if err != nil || !found {
		t.Fatalf("record = %+v found=%v err=%v", rec, found, err)
	}
	rec.Token = "wrong-token"
	if err := s.records().Put("stardew", id, rec); err != nil {
		t.Fatal(err)
	}
	_, err = s.UpdateHostedShare(t.Context(), "stardew", id, nil, share.Include{})
	if usererr.KindOf(err) != usererr.Permission {
		t.Fatalf("wrong token = %v kind=%s", err, usererr.KindOf(err))
	}
}
