package loadersvc

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// splitClient serves a game whose catalog names the overlay component beside the companion, and answers with two
// zips: the companion's and the overlay's.
func splitClient(t *testing.T, companionVersion string) (*components.Client, string) {
	t.Helper()
	var bridgeZip, overlayZip bytes.Buffer
	bridgeHash := writeZip(t, &bridgeZip, map[string]string{
		"MortarSmapiBridge/manifest.json":         `{"Name":"Mortar SMAPI Bridge","Version":"0.2.10","UniqueID":"Rethunk.MortarSmapiBridge","EntryDll":"MortarSmapiBridge.dll"}`,
		"MortarSmapiBridge/MortarSmapiBridge.dll": "bridge",
	})
	overlayHash := writeZip(t, &overlayZip, map[string]string{
		"StreamOverlay/manifest.json":     `{"Name":"Stream Overlay","Version":"0.1.0","UniqueID":"tech.rethunk.mortar.StreamOverlay","EntryDll":"StreamOverlay.dll"}`,
		"StreamOverlay/StreamOverlay.dll": "overlay",
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "overlay") {
			_, _ = w.Write(overlayZip.Bytes())
			return
		}
		_, _ = w.Write(bridgeZip.Bytes())
	}))
	t.Cleanup(srv.Close)
	client := components.NewClient(srv.Client())
	client.GitHubBaseURL = srv.URL
	client.SetManifest(components.Manifest{
		Serial: 1,
		Games: []components.GameInfo{{
			ID: "stardew", Name: "Stardew Valley", Marker: "m", Deploy: components.DeployRedirect,
			Targets: []components.TargetDef{{ID: "mods", Root: "{profileMods}"}},
			Loaders: []components.GameLoader{{ID: "smapi", Name: "SMAPI", Companion: "bridge", Overlay: "overlay"}},
		}},
		Components: []components.Component{
			{
				Game: "stardew", Name: "bridge", Kind: "bridge",
				Source: components.Source{Host: "github.com", Owner: "Rethunk-Tech", Repo: "mortar-smapi-bridge"},
				Tag:    "v" + companionVersion, Asset: "bridge.zip", Version: companionVersion, SHA256: bridgeHash,
			},
			{
				Game: "stardew", Name: "overlay", Kind: "overlay",
				Source: components.Source{Host: "github.com", Owner: "Rethunk-Tech", Repo: "mortar-smapi-overlay"},
				Tag:    "v0.1.0", Asset: "overlay.zip", Version: "0.1.0", SHA256: overlayHash,
			},
		},
	})
	return client, overlayHash
}

func writeZip(t *testing.T, buf *bytes.Buffer, files map[string]string) string {
	t.Helper()
	writer := zip.NewWriter(buf)
	for name, body := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:])
}

// overlayEntries lists the profile's overlay bundles by key.
func overlayEntries(p profile.Profile) []string {
	var out []string
	for _, e := range p.Entries {
		if e.Source.Kind == profile.SourceMortar && e.Source.BundleComponent() == profile.BundleOverlay {
			out = append(out, e.Key)
		}
	}
	return out
}

// The overlay setting on, with a post-split companion: every profile gets the overlay bundle beside the companion,
// under its own byte-identical overlay-<version>-<sha12> key with source mortar.
func TestTheOverlayBundleJoinsTheProfilesWithTheSettingOn(t *testing.T) {
	set, err := tempSettings(t)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.OverlayEnabled = true }); err != nil {
		t.Fatal(err)
	}
	items, profiles := testenv.Stores(t)
	client, overlayHash := splitClient(t, "0.6.0")
	svc := testService(t, set, items, profiles, client)
	created, err := profiles.Create("stardew", "Main")
	if err != nil {
		t.Fatal(err)
	}
	SyncBundled(t.Context(), svc, "stardew")
	got, err := profiles.Get("stardew", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	overlay := store.OverlayKey("0.1.0", overlayHash)
	if !slices.Equal(overlayEntries(got), []string{overlay}) {
		t.Fatalf("overlay entries = %v, want %s", overlayEntries(got), overlay)
	}
	if _, err := os.Stat(filepath.Join(dirOf(t, items, overlay), "StreamOverlay", "StreamOverlay.dll")); err != nil {
		t.Fatal(err)
	}
}

// The setting off keeps the overlay out of the profiles, and flipping it off takes the bundle back out.
func TestTheOverlayBundleFollowsItsSetting(t *testing.T) {
	set, err := tempSettings(t)
	if err != nil {
		t.Fatal(err)
	}
	items, profiles := testenv.Stores(t)
	client, overlayHash := splitClient(t, "0.6.0")
	svc := testService(t, set, items, profiles, client)
	created, err := profiles.Create("stardew", "Main")
	if err != nil {
		t.Fatal(err)
	}
	SyncBundled(t.Context(), svc, "stardew")
	if got, _ := profiles.Get("stardew", created.ID); len(overlayEntries(got)) != 0 {
		t.Fatalf("overlay entries with the setting off = %v", overlayEntries(got))
	}
	if _, err := set.Update(func(v *settings.Settings) { v.OverlayEnabled = true }); err != nil {
		t.Fatal(err)
	}
	SyncBundled(t.Context(), svc, "stardew")
	got, _ := profiles.Get("stardew", created.ID)
	if want := []string{store.OverlayKey("0.1.0", overlayHash)}; !slices.Equal(overlayEntries(got), want) {
		t.Fatalf("overlay entries = %v, want %v", overlayEntries(got), want)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.OverlayEnabled = false }); err != nil {
		t.Fatal(err)
	}
	SyncBundled(t.Context(), svc, "stardew")
	got, _ = profiles.Get("stardew", created.ID)
	if len(overlayEntries(got)) != 0 {
		t.Fatalf("entries after switching the overlay off = %+v", got.Entries)
	}
}

// A 0.5.x companion serves the overlay itself and binds its port, so the overlay mod never joins the profile beside it.
func TestAPreSplitCompanionKeepsTheOverlayOut(t *testing.T) {
	set, err := tempSettings(t)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.OverlayEnabled = true }); err != nil {
		t.Fatal(err)
	}
	items, profiles := testenv.Stores(t)
	client, _ := splitClient(t, "0.2.10")
	svc := testService(t, set, items, profiles, client)
	created, err := profiles.Create("stardew", "Main")
	if err != nil {
		t.Fatal(err)
	}
	SyncBundled(t.Context(), svc, "stardew")
	got, _ := profiles.Get("stardew", created.ID)
	if len(overlayEntries(got)) != 0 {
		t.Fatalf("a companion that still serves the overlay got an overlay mod beside it: %v", overlayEntries(got))
	}
	for _, e := range got.Entries {
		if e.Source.BundleComponent() == profile.BundleOverlay {
			t.Fatalf("entries = %+v", got.Entries)
		}
	}
}

// A profile upgraded while the setting is on keeps its overlay: the same sync replaces it with the catalog's newer
// one and leaves the companion beside it alone.
func TestTheOverlayBundleIsReplacedInPlaceAndOnlyItsOwn(t *testing.T) {
	set, err := tempSettings(t)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.OverlayEnabled = true }); err != nil {
		t.Fatal(err)
	}
	items, profiles := testenv.Stores(t)
	client, firstHash := splitClient(t, "0.6.0")
	svc := testService(t, set, items, profiles, client)
	created, err := profiles.Create("stardew", "Main")
	if err != nil {
		t.Fatal(err)
	}
	SyncBundled(t.Context(), svc, "stardew")
	before, _ := profiles.Get("stardew", created.ID)
	bridgesBefore := countComponent(before, profile.BundleBridge)

	secondHash := firstHash
	client.SetManifest(newerOverlayManifest(t, client.Manifest(), secondHash))
	SyncBundled(t.Context(), svc, "stardew")
	after, _ := profiles.Get("stardew", created.ID)
	if want := []string{store.OverlayKey("0.2.0", secondHash)}; !slices.Equal(overlayEntries(after), want) {
		t.Fatalf("overlay entries = %v, want %v", overlayEntries(after), want)
	}
	if countComponent(after, profile.BundleBridge) != bridgesBefore {
		t.Fatalf("the sync took the companion with the overlay: %+v", after.Entries)
	}
}

// newerOverlayManifest bumps the overlay component of m to 0.2.0, leaving the companion as it is.
func newerOverlayManifest(t *testing.T, m components.Manifest, hash string) components.Manifest {
	t.Helper()
	for i := range m.Components {
		if m.Components[i].Name == "overlay" {
			m.Components[i].Version, m.Components[i].Tag, m.Components[i].SHA256 = "0.2.0", "v0.2.0", hash
		}
	}
	return m
}

func countComponent(p profile.Profile, component string) int {
	n := 0
	for _, e := range p.Entries {
		if e.Source.Kind == profile.SourceMortar && e.Source.BundleComponent() == component {
			n++
		}
	}
	return n
}

// Today's catalog names no overlay component, so nothing above runs: the setting on adds no bundle and no key.
func TestNoOverlayComponentKeepsTheProfilesAsTheyWere(t *testing.T) {
	set, err := tempSettings(t)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.OverlayEnabled = true }); err != nil {
		t.Fatal(err)
	}
	items, profiles := testenv.Stores(t)
	client, bridgeHash := bridgeClient(t)
	svc := testService(t, set, items, profiles, client)
	created, err := profiles.Create("stardew", "Main")
	if err != nil {
		t.Fatal(err)
	}
	SyncBundled(t.Context(), svc, "stardew")
	got, _ := profiles.Get("stardew", created.ID)
	if len(overlayEntries(got)) != 0 {
		t.Fatalf("overlay entries = %v", overlayEntries(got))
	}
	if _, err := items.Path("stardew", store.OverlayKey("0.1.0", bridgeHash)); err == nil {
		t.Fatal("an overlay item was built with no overlay component in the catalog")
	}
}

// tempSettings is a settings store in a throwaway data folder, which the tests must point the data dir at.
func tempSettings(t *testing.T) (*settings.Store, error) {
	t.Helper()
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("LOCALAPPDATA", data)
	return settings.OpenIn(data)
}

func dirOf(t *testing.T, items *store.Store, key string) string {
	t.Helper()
	dir, err := items.Path("stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
