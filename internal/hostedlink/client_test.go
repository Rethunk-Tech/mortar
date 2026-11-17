package hostedlink

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-Tech/go-httpkit"
)

const testToken = "abcdefghijklmnopqrstuvwxyz0123456789ABCDE"

func TestCreateUpdateAndWrongToken(t *testing.T) {
	t.Parallel()
	var created payloadBody
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/shares", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &created); err != nil {
			t.Errorf("decode create: %v", err)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("create sent an Authorization header")
		}
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(CreateResponse{Code: "AbCdEfGh12", Token: testToken, Version: 1}); err != nil {
			t.Errorf("encode create: %v", err)
		}
	})
	mux.HandleFunc("PUT /api/shares/AbCdEfGh12", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testToken {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(UpdateResponse{Version: 2}); err != nil {
			t.Errorf("encode update: %v", err)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := Client{BaseURL: srv.URL, HTTPClient: srv.Client()}

	got, err := c.Create(t.Context(), "payload-text")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Payload != "payload-text" || got.Code != "AbCdEfGh12" || got.Token != testToken || got.Version != 1 {
		t.Fatalf("create = %+v payload=%q", got, created.Payload)
	}

	upd, err := c.Update(t.Context(), got.Code, got.Token, "payload-v2")
	if err != nil || upd.Version != 2 {
		t.Fatalf("update = %+v, %v", upd, err)
	}

	_, err = c.Update(t.Context(), got.Code, "wrong-token", "payload-v3")
	var se *httpkit.StatusError
	if !errors.As(err, &se) || se.Code != http.StatusForbidden {
		t.Fatalf("wrong token = %v, want HTTP 403", err)
	}
}

func TestShortURL(t *testing.T) {
	t.Parallel()
	if got := ShortURL("stardew", "AbCdEfGh12"); got != "https://mortar.rethunk.tech/stardew/s/AbCdEfGh12" {
		t.Fatalf("ShortURL = %q", got)
	}
}
