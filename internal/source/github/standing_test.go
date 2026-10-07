package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestStandingsFlagsArchivedRepositories(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/me/old":
			_, _ = w.Write([]byte(`{"archived":true}`))
		case "/repos/me/live":
			_, _ = w.Write([]byte(`{"archived":false}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	got, err := (&Driver{URL: srv.URL}).Standings(context.Background(), "1", []string{"me/old", "me/live", "me/gone", "bad"})
	if err != nil || len(got) != 1 || got["me/old"].State != "archived" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestStandingsStopsWhenGitHubIsBusy(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	_, err := (&Driver{URL: srv.URL}).Standings(context.Background(), "1", []string{"me/a", "me/b"})
	if !errors.Is(err, source.ErrBusy) {
		t.Fatalf("err = %v, want busy", err)
	}
}
