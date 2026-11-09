package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/github"
)

func TestResolveDownloadReportsTheLibraryStatus(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases") && !strings.Contains(r.URL.Path, "/assets/") {
			fmt.Fprintf(w, `[{"tag_name":"v1.2.3","assets":[{"name":"mod.zip","browser_download_url":%q}]}]`, srv.URL+"/mod.zip")
			return
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	_, err := resolve(t.Context(), &github.Client{HTTP: srv.Client(), CacheDir: t.TempDir(), APIBase: srv.URL}, components.SourceComponent{
		Game: "stardew-valley", Name: "SMAPI", Kind: "loader",
		Source:       components.Source{Host: "github.com", Owner: "Pathoschild", Repo: "SMAPI"},
		AssetPattern: "mod.zip", Version: "1.2.3",
	})
	var se *httpkit.StatusError
	if !errors.As(err, &se) || se.Code != http.StatusBadGateway {
		t.Fatalf("asset GET must keep the library status: %v", err)
	}
}

// The generator accepts the overlay kind the client reads, so a source entry for the split overlay resolves.
func TestResolveAcceptsAnOverlayComponent(t *testing.T) {
	_, err := resolve(t.Context(), &github.Client{HTTP: http.DefaultClient, CacheDir: t.TempDir()}, components.SourceComponent{
		Game: "lethal-company", Name: "overlay", Kind: "overlay",
		Source:       components.Source{Host: "thunderstore.io", Namespace: "Mortar", Name: "StreamOverlay"},
		AssetPattern: "overlay.zip", Version: "0.1.0",
	})
	if err == nil || strings.Contains(err.Error(), "unknown kind") {
		t.Fatalf("an overlay source entry was refused by kind: %v", err)
	}
}
