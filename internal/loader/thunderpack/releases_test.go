package thunderpack

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

type offline struct{}

func (offline) RoundTrip(*http.Request) (*http.Response, error) { return nil, errors.New("offline") }

func TestReleasesNeedACommunityAndALoaderPackage(t *testing.T) {
	r := Releases{Name: "Pack"}
	if _, err := r.Versions(t.Context(), components.GameInfo{Name: "None"}); err == nil || !strings.Contains(err.Error(), "no Thunderstore community") {
		t.Errorf("Versions without a community: %v", err)
	}
	if _, err := r.Latest(t.Context(), components.GameInfo{Name: "None"}); err == nil {
		t.Error("Latest without a community: want an error")
	}
	if err := r.Fetch(t.Context(), components.GameInfo{Name: "None"}, "1.0.0", t.TempDir()+"/x.zip"); err == nil {
		t.Error("Fetch without a community: want an error")
	}
}

func TestReleasesNameTheLoaderWhenTheIndexIsUnreachable(t *testing.T) {
	g, ok := components.Game("lethal-company")
	if !ok {
		t.Fatal("no lethal-company")
	}
	d := thunderstore.Driver{HTTP: &http.Client{Transport: offline{}}, CacheDir: t.TempDir()}
	r := Releases{Name: "BepInEx", Index: &d}
	if _, err := r.Versions(t.Context(), g); err == nil || !strings.Contains(err.Error(), "look up BepInEx releases") {
		t.Errorf("Versions: %v", err)
	}
	if err := r.Fetch(t.Context(), g, "5.4.2100", t.TempDir()+"/x.zip"); err == nil || !strings.Contains(err.Error(), "find BepInEx 5.4.2100") {
		t.Errorf("Fetch: %v", err)
	}
}
