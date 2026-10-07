package source

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoJSON(t *testing.T) {
	var gotUA, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA, gotKey = r.UserAgent(), r.Header.Get("X-Key")
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte(`{"n":3}`))
		case "/auth":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"n":3}`))
		case "/busy":
			w.WriteHeader(http.StatusTooManyRequests)
		case "/html":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`<html>`))
		case "/bad":
			_, _ = w.Write([]byte(`<html>`))
		}
	}))
	t.Cleanup(srv.Close)
	call := func(path string, out any) error {
		return DoJSON(context.Background(), Request{
			Service: "Site", URL: srv.URL + path, UserAgent: "Mortar/1",
			Header: func(h http.Header) { h.Set("X-Key", "k") },
		}, out)
	}
	var out struct{ N int }
	if err := call("/ok", &out); err != nil || out.N != 3 || gotUA != "Mortar/1" || gotKey != "k" {
		t.Fatalf("ok: %+v %v ua=%q key=%q", out, err, gotUA, gotKey)
	}
	if se, ok := errors.AsType[*StatusError](call("/auth", &out)); !ok || !se.Auth() || se.Code != 403 {
		t.Fatalf("auth: %+v", se)
	}
	if !errors.Is(call("/busy", &out), ErrBusy) {
		t.Fatal("429 should be busy")
	}
	err := call("/html", &out)
	if se, ok := errors.AsType[*StatusError](err); !ok || se.Auth() || se.Error() != "Site answered 502 Bad Gateway" {
		t.Fatalf("other status: %v", err)
	}
	if err := call("/bad", &out); err == nil {
		t.Fatal("a 200 with a body that is not JSON should fail")
	}
}
