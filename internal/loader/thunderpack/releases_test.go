package thunderpack

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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

func TestAPackThatDiffersFromItsCatalogPinIsRefused(t *testing.T) {
	file := filepath.Join(t.TempDir(), "pack.zip")
	if err := os.WriteFile(file, []byte("pack bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("pack bytes"))
	client := components.NewClient(nil)
	pin := func(sha string) {
		client.SetManifest(components.Manifest{Pins: []components.Pin{{Kind: components.PinPack, Name: "BepInEx-BepInExPack", Version: "5.4.2100", SHA256: sha}}})
	}
	r := Releases{Name: "BepInEx", Components: client}

	pin(strings.Repeat("0", 64))
	if err := r.checkPin("BepInEx-BepInExPack", "5.4.2100", file); err == nil {
		t.Fatal("a pack that differs from its pin was accepted")
	}
	if _, err := os.Stat(file); err == nil {
		t.Fatal("the refused pack was left on disk")
	}

	if err := os.WriteFile(file, []byte("pack bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	pin(hex.EncodeToString(sum[:]))
	if err := r.checkPin("bepinex-bepinexpack", "5.4.2100", file); err != nil {
		t.Fatalf("the pinned pack: %v", err)
	}
	if err := r.offered("BepInEx-BepInExPack", "5.4.2100"); err != nil {
		t.Fatalf("the pinned version: %v", err)
	}
	if err := r.offered("BepInEx-BepInExPack", "5.4.2200"); err == nil || usererr.KindOf(err) != usererr.Outdated {
		t.Fatalf("a version newer than the catalog's pins was offered: %v", err)
	}
	if err := r.offered("Other-Pack", "1.0.0"); err != nil {
		t.Fatalf("a package the catalog pins nothing of installs unverified: %v", err)
	}
}
