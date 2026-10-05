package components

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

func signedManifest(t *testing.T, serial uint64, public ed25519.PrivateKey) ([]byte, []byte) {
	t.Helper()
	m := Manifest{
		Serial: serial,
		Components: []Component{{
			Game: "stardew", Name: "bridge", Kind: "bridge",
			Source: Source{Host: "github.com", Owner: "Rethunk-Tech", Repo: "mortar-smapi-bridge"},
			Tag:    "v1.1.0", Asset: "bridge.zip", Version: "1.1.0",
			SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}},
	}
	body, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return body, ed25519.Sign(public, body)
}

func TestVerifySignature(t *testing.T) {
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	body, signature := signedManifest(t, 1, private)
	if err := Verify(body, signature, public); err != nil {
		t.Fatalf("valid signature: %v", err)
	}
	tampered := append([]byte(nil), body...)
	tampered[0] ^= 1
	if err := Verify(tampered, signature, public); err == nil {
		t.Fatal("tampered manifest was accepted")
	}
	wrong, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(body, signature, wrong); err == nil {
		t.Fatal("signature verified with the wrong key")
	}
}

func TestManifestRejectsUntrustedHost(t *testing.T) {
	m := Manifest{
		Serial: 1,
		Components: []Component{{
			Game: "stardew", Name: "bridge", Kind: "bridge",
			Source: Source{Host: "example.com", Owner: "a", Repo: "b"},
			Tag:    "v1", Asset: "bridge.zip", Version: "1", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}},
	}
	if err := m.Validate(); err == nil {
		t.Fatal("manifest accepted a source outside the allowlist")
	}
}

func TestLoadRefusesSerialRollbackAndKeepsCachedManifest(t *testing.T) {
	_, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	public, ok := private.Public().(ed25519.PublicKey)
	if !ok {
		t.Fatal("generated key is not Ed25519")
	}
	base := bundledSerial(t)
	body, signature := signedManifest(t, base+2, private)
	rollback, rollbackSignature := signedManifest(t, base+1, private)
	serveRollback := false
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases" {
			// Newest first, as GitHub lists them: an app release and a draft come before the newest manifest.
			_, _ = fmt.Fprintf(w, `[{"tag_name":"v9.9.9","assets":[{"name":"components.json","browser_download_url":"%[1]s/wrong"}]},
				{"tag_name":"components-9","draft":true,"assets":[{"name":"components.json","browser_download_url":"%[1]s/wrong"}]},
				{"tag_name":"components-2","assets":[{"name":"components.json","browser_download_url":"%[1]s/components.json"}]}]`, server.URL)
			return
		}
		if r.URL.Path == "/components.json.sig" {
			if serveRollback {
				_, _ = w.Write(rollbackSignature)
			} else {
				_, _ = w.Write(signature)
			}
			return
		}
		if serveRollback {
			_, _ = w.Write(rollback)
		} else {
			_, _ = w.Write(body)
		}
	}))
	t.Cleanup(server.Close)
	now := time.Now()
	cache := &meta.Client{CacheDir: t.TempDir(), Now: func() time.Time { return now }}
	client := NewClient(server.Client())
	client.ReleasesURL = server.URL + "/releases"
	if manifest, err := client.Load(t.Context(), cache, public); err != nil || manifest.Serial != base+2 {
		t.Fatalf("initial manifest = %#v, %v", manifest, err)
	}
	now = now.Add(25 * time.Hour)
	serveRollback = true
	manifest, err := client.Load(t.Context(), cache, public)
	if manifest.Serial != base+2 {
		t.Fatalf("rollback replaced cached serial: %d", manifest.Serial)
	}
	if err == nil {
		t.Fatal("serial rollback was not reported")
	}
}

func TestDownloadHashMismatchKeepsCurrentVersion(t *testing.T) {
	body := []byte("new but wrong")
	sum := sha256.Sum256([]byte("expected"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.Client())
	client.GitHubBaseURL = server.URL
	dest := filepath.Join(t.TempDir(), "bridge.zip")
	if err := os.WriteFile(dest, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	component := Component{
		Game: "stardew", Name: "bridge", Kind: "bridge",
		Source: Source{Host: "github.com", Owner: "a", Repo: "b"},
		Tag:    "v1", Asset: "bridge.zip", Version: "1",
		SHA256: fmtSHA(sum),
	}
	if err := client.Download(t.Context(), component, dest); err == nil {
		t.Fatal("accepted a mismatched component")
	}
	got, err := os.ReadFile(filepath.Clean(dest))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "current" {
		t.Fatalf("current component changed to %q", got)
	}
}

func TestOfflineFirstRunUsesBundledManifest(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	client := NewClient(http.DefaultClient)
	client.ManifestURL = server.URL + "/components.json"
	manifest, err := client.Load(t.Context(), &meta.Client{CacheDir: t.TempDir()}, []byte("not a key"))
	if manifest.Serial == 0 {
		t.Fatal("offline load returned no bundled manifest")
	}
	if err == nil {
		t.Fatal("offline load unexpectedly fetched a manifest")
	}
}

func fmtSHA(sum [sha256.Size]byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, len(sum)*2)
	for i, b := range sum {
		out[i*2] = hex[b>>4]
		out[i*2+1] = hex[b&15]
	}
	return string(out)
}

func bundledSerial(t *testing.T) uint64 {
	t.Helper()
	bundled, err := BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	return bundled.Serial
}

func TestLoadRefusesManifestOlderThanBundled(t *testing.T) {
	_, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	public, ok := private.Public().(ed25519.PublicKey)
	if !ok {
		t.Fatal("generated key is not Ed25519")
	}
	base := bundledSerial(t)
	body, signature := signedManifest(t, base-1, private)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/components.json.sig" {
			_, _ = w.Write(signature)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.Client())
	client.ManifestURL = server.URL + "/components.json"
	manifest, err := client.Load(t.Context(), nil, public)
	if err == nil || manifest.Serial != base {
		t.Fatalf("older signed manifest = serial %d, %v; want the bundled serial %d and an error", manifest.Serial, err, base)
	}
}

func TestGameFallsBackToBundledAndRejectsUnsafeNames(t *testing.T) {
	c := NewClient(nil)
	c.SetManifest(Manifest{Serial: 1})
	g, ok := c.Game("stardew")
	if !ok || g.Name != "Stardew Valley" || g.Marker == "" || g.Stores.GOG == nil || g.Stores.GOG.ProductID == "" || len(g.Loaders) == 0 {
		t.Fatalf("a manifest without games must fall back to the bundled identity: %+v %v", g, ok)
	}
	bad := Manifest{Serial: 1, Games: []GameInfo{{ID: "x", Name: "X", Marker: "../escape.dll", Loaders: []GameLoader{{ID: "l"}}}}}
	if err := bad.Validate(); err == nil {
		t.Fatal("a marker that leaves the game folder must be refused")
	}
	dup := Manifest{Serial: 1, Games: []GameInfo{{ID: "x", Name: "X", Marker: "m", Loaders: []GameLoader{{ID: "l"}}}, {ID: "x", Name: "X", Marker: "m", Loaders: []GameLoader{{ID: "l"}}}}}
	if err := dup.Validate(); err == nil {
		t.Fatal("a game listed twice must be refused")
	}
}

func TestLoadRefusesManifestWithoutCatalogShape(t *testing.T) {
	_, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	public, ok := private.Public().(ed25519.PublicKey)
	if !ok {
		t.Fatal("generated key is not Ed25519")
	}
	base := bundledSerial(t)
	body := []byte(`{"serial":` + strconv.FormatUint(base+1, 10) + `,"components":[],"games":[{"id":"stardew","name":"Stardew Valley","steamAppId":"413150","marker":"Stardew Valley.dll","loader":"SMAPI"}]}`)
	signature := ed25519.Sign(private, body)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/components.json.sig" {
			_, _ = w.Write(signature)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.Client())
	client.ManifestURL = server.URL + "/components.json"
	manifest, err := client.Load(t.Context(), nil, public)
	if err == nil || manifest.Serial != base {
		t.Fatalf("old-shape manifest = serial %d, %v; want the bundled serial %d and an error", manifest.Serial, err, base)
	}
}

func TestComponentFallsBackToTheBundledManifestBeforeLoad(t *testing.T) {
	c := NewClient(nil)
	got, ok := c.Component("stardew", "bridge")
	if !ok || got.Kind != "bridge" || got.Version == "" {
		t.Fatalf("component %+v ok %v", got, ok)
	}
}

func TestBundledGamesCarryPathTemplates(t *testing.T) {
	m, err := BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range m.Games {
		if err := g.Validate(); err != nil {
			t.Fatal(err)
		}
		switch g.ID {
		case "stardew":
			if got := g.Paths["saves"].Linux; got != "{xdgConfig}/StardewValley/Saves" {
				t.Fatalf("stardew linux saves = %q", got)
			}
			if got := g.Paths["startupPreferences"].Windows; got != "{appData}/StardewValley/startup_preferences" {
				t.Fatalf("stardew windows startup preferences = %q", got)
			}
		case "lethal-company":
			if got := g.Paths["saves"].Windows; got != "{localLow}/ZeekerssRBLX/Lethal Company" {
				t.Fatalf("lc windows saves = %q", got)
			}
		}
	}
	bad := GameInfo{ID: "x", Name: "x", Marker: "x", Loaders: []GameLoader{{ID: "l"}}, Paths: map[string]PathTemplate{"saves": {Linux: "/etc"}}}
	if bad.Validate() == nil {
		t.Fatal("a path without a token validated")
	}
}

func TestBundledGamesCarryTargetsAndCompanions(t *testing.T) {
	s, _ := BundledGame("stardew")
	if tgt, ok := s.Target("mods"); !ok || tgt.Root != "{profileMods}" || tgt.Writable {
		t.Fatalf("stardew mods target = %+v", tgt)
	}
	if s.Loaders[0].Companion != "bridge" {
		t.Fatalf("smapi companion = %q", s.Loaders[0].Companion)
	}
	lc, _ := BundledGame("lethal-company")
	if tgt, ok := lc.Target("profile"); !ok || tgt.Root != "{profile}" || tgt.Writable {
		t.Fatalf("lethal-company profile target = %+v", tgt)
	}
	if s.Deploy != DeployRedirect || lc.Deploy != DeployLink {
		t.Fatalf("deploy = %q, %q", s.Deploy, lc.Deploy)
	}
	if c, _ := lc.Target("config"); !c.Writable || c.Install != "{install}/BepInEx/config" || c.Root != "{profile}/BepInEx/config" {
		t.Fatalf("lethal-company config target = %+v", c)
	}
	bad := s
	bad.Targets = []TargetDef{{ID: "mods", Root: "mods"}}
	if bad.Validate() == nil {
		t.Fatal("a target root without a token validated")
	}
	bad.Targets = []TargetDef{{ID: "mods", Root: "{profileMods}", Install: "{install}"}}
	if bad.Validate() == nil {
		t.Fatal("an install root on a redirected game validated")
	}
}
