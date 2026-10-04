package support

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/doctor"
	"github.com/Rethunk-Tech/mortar/internal/problems"
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

func TestLogIsOnlyItsProfiles(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	dirs := map[string]string{"a": "/home/u/p/a/mods", "b": "/home/u/p/b/mods"}
	s := NewService("1.2.3", func(string) problems.Environment { return problems.Environment{} }, "/home/u",
		func(_, id string) (string, error) { return dirs[id], nil })
	if got, err := s.Log("stardew", "a"); err != nil || got != "" {
		t.Fatalf("no log yet: %q, %v", got, err)
	}
	dir := filepath.Join(cfg, "StardewValley", "ErrorLogs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	log := "[05:00:01 INFO  SMAPI] Mods go here: ~/p/a/mods\n"
	if err := os.WriteFile(filepath.Join(dir, "SMAPI-latest.txt"), []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Log("stardew", "a"); err != nil || got != log {
		t.Fatalf("profile a: %q, %v", got, err)
	}
	if got, err := s.Log("stardew", "b"); err != nil || got != "" {
		t.Fatalf("profile b: %q, %v", got, err)
	}
}

func TestDiagnosticsSectionHidesHomeAndStaysShort(t *testing.T) {
	report := doctor.Report{Checks: []doctor.Check{{Status: "ok", Detail: "/home/me/.local/share/mortar exists"}}}
	got := diagnosticsSection(report, "/home/me", "")
	if !strings.Contains(got, "ok: ~/.local/share/mortar exists") || strings.Contains(got, "/home/me") {
		t.Fatalf("section %q", got)
	}
	long := doctor.Report{}
	for range 500 {
		long.Checks = append(long.Checks, doctor.Check{Status: "warn", Detail: strings.Repeat("x", 40)})
	}
	if got := diagnosticsSection(long, "", ""); len(got) > maxDiagnostics+64 || !strings.Contains(got, "…") {
		t.Fatalf("long section %d bytes", len(got))
	}
}

func TestBugURLCarriesTheReportAndDiagnosticsOnlyWhenAsked(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	s := NewService("1.2.3", func(string) problems.Environment { return problems.Environment{} }, t.TempDir(), nil)
	parse := func(raw string) url.Values {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return u.Query()
	}
	q := parse(s.BugURL("", BugReport{Title: "Play button stuck", Happened: "It spins", Steps: "Click Play"}))
	body := q.Get("body")
	if q.Get("title") != "Play button stuck" || !strings.Contains(body, "**What happened**\nIt spins") ||
		!strings.Contains(body, "**Steps to reproduce**\nClick Play") || !strings.Contains(body, "Mortar 1.2.3") {
		t.Fatalf("title %q body %q", q.Get("title"), body)
	}
	if strings.Contains(body, "**Diagnostics**") {
		t.Fatal("diagnostics included without being asked")
	}
	if !strings.Contains(parse(s.BugURL("", BugReport{Diagnostics: true})).Get("body"), "**Diagnostics**") {
		t.Fatal("diagnostics missing when asked")
	}
}
