package support

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/problems"
)

func upload(t *testing.T, handler http.HandlerFunc, log string) (string, error) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	s := newService(srv.URL, "1.2.3", func(string) problems.Environment { return problems.Environment{} })
	return s.Upload(context.Background(), log)
}

func TestUploadRedirectIsTheLink(t *testing.T) {
	var got string
	link, err := upload(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/log" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		got = r.PostFormValue("input")
		http.Redirect(w, r, "/log/abc123", http.StatusFound)
	}, "[12:00:00 INFO  SMAPI] hi & bye")
	if err != nil {
		t.Fatal(err)
	}
	if got != "[12:00:00 INFO  SMAPI] hi & bye" {
		t.Errorf("input %q", got)
	}
	if !strings.HasSuffix(link, "/log/abc123") || !strings.HasPrefix(link, "http://127.0.0.1") {
		t.Errorf("link %q", link)
	}
}

func TestUploadFailures(t *testing.T) {
	cases := map[string]http.HandlerFunc{
		"200 page": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) },
		"500":      func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
		"400":      func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) },
		"redirect elsewhere": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/", http.StatusFound)
		},
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			if link, err := upload(t, h, "log"); err == nil {
				t.Errorf("got link %q, want an error", link)
			}
		})
	}
}

func TestUploadRejectsOversizeWithoutSending(t *testing.T) {
	_, err := upload(t, func(http.ResponseWriter, *http.Request) { t.Error("request sent") }, strings.Repeat("x", MaxLog+1))
	if err == nil {
		t.Fatal("want an error")
	}
}
