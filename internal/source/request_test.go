package source

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
		case "/busy-wait":
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(http.StatusTooManyRequests)
		case "/html":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`<html>`))
		case "/bad":
			_, _ = w.Write([]byte(`<html>`))
		case "/big":
			// Valid JSON that a read cut at MaxBody would still parse.
			_, _ = w.Write([]byte(`{"n":3}`))
			_, _ = w.Write(bytes.Repeat([]byte(" "), MaxBody))
		}
	}))
	t.Cleanup(srv.Close)
	call := func(path string, out any) error {
		return DoJSON(t.Context(), Request{
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
	if busy, ok := errors.AsType[*BusyError](call("/busy-wait", &out)); !ok {
		t.Fatal("429 with Retry-After should be busy")
	} else if d := time.Until(busy.Reset); d < 110*time.Second || d > 120*time.Second {
		t.Fatalf("429 Retry-After reset %v", busy.Reset)
	}
	err := call("/html", &out)
	if se, ok := errors.AsType[*StatusError](err); !ok || se.Auth() || se.Error() != "Site answered 502 Bad Gateway" {
		t.Fatalf("other status: %v", err)
	}
	if usererr.KindOf(err) != usererr.Network {
		t.Fatalf("502 should classify as Network: %v", usererr.KindOf(err))
	}
	if k := usererr.KindOf(&StatusError{Service: "Site", Code: 404, Status: "404 Not Found"}); k != usererr.NotFound {
		t.Fatalf("404 should classify as NotFound: %v", k)
	}
	if err := call("/bad", &out); err == nil || !strings.Contains(err.Error(), "site:") {
		t.Fatalf("a 200 with a body that is not JSON should name the site: %v", err)
	}
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()
	leak := DoJSON(t.Context(), Request{Service: "Site", URL: closed.URL + "/ok?apikey=secret", UserAgent: "Mortar/1"}, &out)
	if leak == nil || strings.Contains(leak.Error(), "apikey") {
		t.Fatalf("a transport error should redact the query: %v", leak)
	}
	if err := call("/big", &out); !errors.Is(err, safefs.ErrTooLarge) {
		t.Fatalf("an oversize body should be refused, not truncated: %v", err)
	}
}
