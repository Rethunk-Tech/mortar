package smapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

// A source pattern may name the whole tag; the installer the catalog pins must carry the same name.
func TestInstallerAssetFillsTagPlaceholder(t *testing.T) {
	l := Loader{AssetPattern: "SMAPI-{tag}-installer.zip"}
	if got := l.installerAsset("4.5.2"); got != "SMAPI-4.5.2-installer.zip" {
		t.Fatalf("installerAsset = %q", got)
	}
}

func TestVersionsSkipsPrereleasesLikeThePinRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"tag_name":"4.3.0","prerelease":true,"assets":[]},{"tag_name":"4.2.1","assets":[]}]`))
	}))
	t.Cleanup(srv.Close)
	l := Loader{ReleasesURL: srv.URL, Components: pinning("4.3.0", strings.Repeat("ab", 32))}
	got, err := l.Versions(t.Context(), components.GameInfo{})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range got {
		if v == "4.3.0" {
			t.Fatalf("versions = %v, pinned prerelease offered", got)
		}
	}
}
