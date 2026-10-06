package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetailsReadsTheRawReadme(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/r/readme" || r.Header.Get("Accept") != "application/vnd.github.raw" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("# Hello"))
	}))
	t.Cleanup(srv.Close)
	d := &Driver{URL: srv.URL}
	got, err := d.Details(t.Context(), "", "o/r", "1")
	if err != nil || got.Description != "# Hello" {
		t.Fatalf("details = %+v, %v", got, err)
	}
	if got, err = d.Details(t.Context(), "", "o/none", "1"); err != nil || got.Description != "" {
		t.Fatalf("no readme = %+v, %v", got, err)
	}
	if _, err = d.Details(t.Context(), "", "o/r/../x", "1"); err == nil {
		t.Fatal("a path was accepted as a repo")
	}
}
