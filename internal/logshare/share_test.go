package logshare

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestUploadPostsFormAndReturnsLocation(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var gotBody string
	var gotType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		gotType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Location", "/log/abc123def")
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(srv.Close)

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	c.HTTP.CheckRedirect = noFollow
	link, err := c.Upload(context.Background(), "SMAPI 4.0 for Stardew Valley")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(link, "/log/abc123def") {
		t.Fatalf("link %q", link)
	}
	if !strings.HasPrefix(gotType, "application/x-www-form-urlencoded") {
		t.Fatalf("content-type %q", gotType)
	}
	vals, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatal(err)
	}
	if vals.Get("input") != "SMAPI 4.0 for Stardew Valley" {
		t.Fatalf("input %q", vals.Get("input"))
	}
}

func TestUploadRejectsEmpty(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	c := New()
	if _, err := c.Upload(context.Background(), "  \n"); !errors.Is(err, ErrEmpty) {
		t.Fatalf("got %v", err)
	}
}

func TestUploadNeedsLogPageRedirect(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	c.HTTP.CheckRedirect = noFollow
	if _, err := c.Upload(context.Background(), "not empty"); err == nil {
		t.Fatal("expected error")
	}
}
