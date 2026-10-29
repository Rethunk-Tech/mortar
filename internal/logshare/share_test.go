package logshare

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-Tech/go-smapiweb"
)

func TestUploadWording(t *testing.T) {
	cases := map[string]struct {
		handler http.HandlerFunc
		want    string
	}{
		"status": {func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }, "smapi.io answered 502 Bad Gateway"},
		"page":   {func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) }, "smapi.io did not accept the log"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			t.Cleanup(srv.Close)
			_, err := Upload(context.Background(), &smapiweb.Client{HTTP: srv.Client(), LogURL: srv.URL}, "log", 0)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}
