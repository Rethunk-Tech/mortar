package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
)

const tagSHA = "a98c256a31dff44471b905b0a4451cf836485555"

func templateGames(commit string) []components.GameInfo {
	return []components.GameInfo{{ID: "lethal-company", Loaders: []components.GameLoader{
		{DevTemplate: &components.DevTemplate{Repo: "Rethunk-Tech/mortar-bepinex5-template", Ref: "v0.2.2", Commit: commit}},
		{},
	}}}
}

func TestPinTemplatesRecordsTheTagsCommitAndRefusesAMovedTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/Rethunk-Tech/mortar-bepinex5-template/commits/v0.2.2" || r.Header.Get("Accept") != "application/vnd.github.sha" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(tagSHA))
	}))
	defer srv.Close()

	for _, recorded := range []string{"", tagSHA} {
		games := templateGames(recorded)
		if err := pinTemplates(t.Context(), srv.Client(), srv.URL, games); err != nil || games[0].Loaders[0].DevTemplate.Commit != tagSHA {
			t.Fatalf("recorded %q: commit %q, %v", recorded, games[0].Loaders[0].DevTemplate.Commit, err)
		}
	}
	err := pinTemplates(t.Context(), srv.Client(), srv.URL, templateGames(strings.Repeat("0", 40)))
	if err == nil || !strings.Contains(err.Error(), "not the recorded") {
		t.Fatalf("a moved tag must stop the build: %v", err)
	}
	missing := templateGames("")
	missing[0].Loaders[0].DevTemplate.Ref = "v9.9.9"
	if err := pinTemplates(t.Context(), srv.Client(), srv.URL, missing); err == nil {
		t.Fatal("a tag GitHub does not know must stop the build")
	}
}

func TestTagCommitRejectsABodyLargerThanAGitHubSHA(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("a", 129)))
	}))
	defer srv.Close()
	_, err := tagCommit(t.Context(), srv.Client(), srv.URL, "Rethunk-Tech/mortar-bepinex5-template", "v0.2.2")
	if !errors.Is(err, safefs.ErrTooLarge) {
		t.Fatalf("a truncated SHA must not pin: %v", err)
	}
}
