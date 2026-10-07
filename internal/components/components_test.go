package components

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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

func newKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	return public, private
}

// signedServer serves body and its signature where Load looks for them and returns a client pointed at it.
func signedServer(t *testing.T, body, signature []byte) *Client {
	t.Helper()
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
	return client
}

func TestVerifySignature(t *testing.T) {
	public, private := newKey(t)
	body, signature := signedManifest(t, 1, private)
	if err := Verify(body, signature, public); err != nil {
		t.Fatalf("valid signature: %v", err)
	}
	tampered := append([]byte(nil), body...)
	tampered[0] ^= 1
	if err := Verify(tampered, signature, public); err == nil {
		t.Fatal("tampered manifest was accepted")
	}
	wrong, _ := newKey(t)
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
	public, private := newKey(t)
	base := bundledSerial(t)
	body, signature := signedManifest(t, base+2, private)
	rollback, rollbackSignature := signedManifest(t, base+1, private)
	serveRollback := false
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases" {
			// Newest first, as GitHub lists them: an app release, a draft and an older schema's components-<serial> release
			// come before the newest manifest.
			_, _ = fmt.Fprintf(w, `[{"tag_name":"v9.9.9","assets":[{"name":"components.json","browser_download_url":"%[1]s/wrong"}]},
				{"tag_name":"components-v2-9","draft":true,"assets":[{"name":"components.json","browser_download_url":"%[1]s/wrong"}]},
				{"tag_name":"components-8","assets":[{"name":"components.json","browser_download_url":"%[1]s/wrong"}]},
				{"tag_name":"components-v2-2","assets":[{"name":"components.json","browser_download_url":"%[1]s/components.json"}]}]`, server.URL)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/wrong") {
			http.NotFound(w, r)
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
	if !errors.Is(err, ErrRollback) || errors.Is(err, ErrBadSignature) {
		t.Fatalf("serial rollback reported as %v, want ErrRollback", err)
	}
}

func TestLoadTellsABadSignatureFromAnUnavailableManifest(t *testing.T) {
	public, private := newKey(t)
	body, signature := signedManifest(t, bundledSerial(t)+1, private)
	signature[0] ^= 0xff
	missing := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case missing:
			http.NotFound(w, r)
		case r.URL.Path == "/components.json.sig":
			_, _ = w.Write(signature)
		default:
			_, _ = w.Write(body)
		}
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.Client())
	client.ManifestURL = server.URL + "/components.json"
	if _, err := client.Load(t.Context(), nil, public); !errors.Is(err, ErrBadSignature) || errors.Is(err, ErrRollback) {
		t.Fatalf("tampered signature reported as %v, want ErrBadSignature", err)
	}
	missing = true
	if _, err := client.Load(t.Context(), nil, public); err == nil || errors.Is(err, ErrBadSignature) || errors.Is(err, ErrRollback) {
		t.Fatalf("missing manifest reported as %v, want a plain failure", err)
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
	public, private := newKey(t)
	base := bundledSerial(t)
	body, signature := signedManifest(t, base-1, private)
	client := signedServer(t, body, signature)
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
	public, private := newKey(t)
	base := bundledSerial(t)
	body := []byte(`{"serial":` + strconv.FormatUint(base+1, 10) + `,"components":[],"games":[{"id":"stardew","name":"Stardew Valley","steamAppId":"413150","marker":"Stardew Valley.dll","loader":"SMAPI"}]}`)
	signature := ed25519.Sign(private, body)
	client := signedServer(t, body, signature)
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
	s, _ := bundledGame("stardew")
	if tgt, ok := s.Target("mods"); !ok || tgt.Root != "{profileMods}" {
		t.Fatalf("stardew mods target = %+v", tgt)
	}
	if s.Loaders[0].Companion != "bridge" {
		t.Fatalf("smapi companion = %q", s.Loaders[0].Companion)
	}
	lc, _ := bundledGame("lethal-company")
	if tgt, ok := lc.Target("profile"); !ok || tgt.Root != "{profile}" {
		t.Fatalf("lethal-company profile target = %+v", tgt)
	}
	if s.Deploy != DeployRedirect || lc.Deploy != DeployProfile {
		t.Fatalf("deploy = %q, %q", s.Deploy, lc.Deploy)
	}
	bad := s
	bad.Targets = []TargetDef{{ID: "mods", Root: "mods"}}
	if bad.Validate() == nil {
		t.Fatal("a target root without a token validated")
	}
}

func TestSandboxCanEnableAGameThatHasNotShipped(t *testing.T) {
	t.Setenv(enableGamesEnv, "later,other")
	games := []GameInfo{{ID: "later"}, {ID: "unlisted"}, {ID: "other"}}
	enableForSandbox(games)
	for _, g := range games {
		if g.Enabled != (g.ID != "unlisted") {
			t.Errorf("%s enabled = %v", g.ID, g.Enabled)
		}
	}
}

func TestBundledStardewShipsTheQualityOfLifeTemplate(t *testing.T) {
	m, err := BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range m.Games {
		if err := g.Validate(); err != nil {
			t.Fatalf("%s: %v", g.ID, err)
		}
		if g.ID != "stardew" {
			continue
		}
		if len(g.Templates) != 1 || g.Templates[0].Name != "Vanilla+ quality of life" {
			t.Fatalf("templates = %+v", g.Templates)
		}
		if n := len(g.Templates[0].Packages); n < 4 || n > 6 {
			t.Fatalf("%d packages, want 4 to 6", n)
		}
		return
	}
	t.Fatal("no stardew in the bundled catalog")
}

func TestValidateRejectsABadTemplatePackage(t *testing.T) {
	g := GameInfo{
		ID: "g", Name: "G", Marker: "g.dll", Deploy: DeployRedirect, Enabled: false,
		Loaders: []GameLoader{{ID: "smapi"}},
		Sources: []GameSource{{ID: "nexus"}},
	}
	bad := []TemplatePackage{{Source: "nexus", Ref: "abc", Name: "x"}, {Source: "github", Ref: "o/r", Name: "x"}, {Source: "nexus", Ref: "0", Name: "x"}}
	for _, p := range bad {
		g.Templates = []StarterTemplate{{ID: "t", Name: "T", Packages: []TemplatePackage{p}}}
		if g.Validate() == nil {
			t.Fatalf("package %+v passed", p)
		}
	}
	g.Templates = []StarterTemplate{{ID: "t", Name: "T", Packages: []TemplatePackage{{Source: "nexus", Ref: "541", Name: "x"}}}}
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveFilePatternsStayInTheSavesFolder(t *testing.T) {
	g := GameInfo{
		ID: "g", Name: "G", Marker: "g.dll", Deploy: DeployRedirect,
		Loaders: []GameLoader{{ID: "smapi"}},
		Sources: []GameSource{{ID: "nexus"}},
	}
	for _, p := range []string{"", "../LCSaveFile*", "a/b/c", "/a", "*/a", "!a/b", `a\b`, "["} {
		g.SaveFiles = []string{p}
		if g.Validate() == nil {
			t.Errorf("pattern %q passed", p)
		}
	}
	for _, p := range []string{"worlds_local/*.fwl", "!*_backup_*"} {
		g.SaveFiles = []string{p}
		if err := g.Validate(); err != nil {
			t.Errorf("pattern %q: %v", p, err)
		}
	}
	g.SaveFiles, g.SaveCompanions = nil, []string{"db"}
	if g.Validate() == nil {
		t.Error("companion without a dot passed")
	}
	lc, ok := bundledGame("lethal-company")
	if !ok || !slices.Equal(lc.SaveFiles, []string{"LCSaveFile*", "LCChallengeFile"}) {
		t.Fatalf("bundled Lethal Company save files = %v", lc.SaveFiles)
	}
	if err := lc.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestACommunityBepInExPackIsTheLoaderPackage(t *testing.T) {
	vh, ok := Game("valheim")
	if !ok || vh.LoaderPackage() != "denikson-BepInExPack_Valheim" {
		t.Fatalf("valheim pack = %q", vh.LoaderPackage())
	}
	if lc, _ := Game("lethal-company"); lc.LoaderPackage() != DefaultLoaderPackage {
		t.Fatalf("lethal-company pack = %q", lc.LoaderPackage())
	}
	for id, want := range map[string]bool{"denikson-BepInExPack_Valheim": true, "bepinex-bepinexpack": true, "denikson-Other": false} {
		if IsLoaderPackage(id) != want {
			t.Errorf("IsLoaderPackage(%q) = %v", id, !want)
		}
	}
}

// The catalog's serial is not bumped with every change to it, so a manifest cached by another build, or by one that did
// not stamp its cache, is fetched again instead of shadowing what this build ships.
func TestLoadFetchesAgainAManifestCachedWithoutThisBuildsStamp(t *testing.T) {
	public, private := newKey(t)
	base := bundledSerial(t)
	oldBody, oldSignature := signedManifest(t, base+1, private)
	body, signature := signedManifest(t, base+2, private)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/components.json.sig" {
			_, _ = w.Write(signature)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	seeded, err := json.Marshal(map[string]any{"fetched": time.Now(), "value": cachedManifest{Manifest: oldBody, Signature: oldSignature}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, cacheName), seeded, 0o600); err != nil {
		t.Fatal(err)
	}
	client := NewClient(server.Client())
	client.ManifestURL = server.URL + "/components.json"

	manifest, err := client.Load(t.Context(), &meta.Client{CacheDir: dir}, public)

	if err != nil || manifest.Serial != base+2 {
		t.Fatalf("manifest = serial %d, %v; want the fetched %d, not the unstamped cache's %d", manifest.Serial, err, base+2, base+1)
	}
}

func TestKnownBrokenIsValidatedAndOptional(t *testing.T) {
	good := KnownBroken{ID: "Ns-Mod", Versions: ">=1.0.0 <1.5.0", Reason: "Crashes."}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []KnownBroken{{Reason: "x"}, {ID: "a"}, {ID: "a", Reason: "x", Versions: "1.0.0"}, {ID: "a", Reason: "x", Versions: "<"}} {
		if bad.Validate() == nil {
			t.Errorf("%+v passed", bad)
		}
	}
	m, err := BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range m.Games {
		if err := g.Validate(); err != nil {
			t.Errorf("%s: %v", g.ID, err)
		}
	}
}

func TestAnOlderClientIgnoresTheKnownBrokenField(t *testing.T) {
	var g GameInfo
	if err := json.Unmarshal([]byte(`{"id":"g","knownBroken":[{"id":"Ns-Mod","versions":"<2.0.0","reason":"r","replacement":"Ns-New"}]}`), &g); err != nil || len(g.KnownBroken) != 1 || g.KnownBroken[0].Replacement != "Ns-New" {
		t.Fatalf("%+v %v", g, err)
	}
	type older struct {
		ID string `json:"id"`
	}
	var o older
	if err := json.Unmarshal([]byte(`{"id":"g","knownBroken":[{"id":"x"}]}`), &o); err != nil || o.ID != "g" {
		t.Fatalf("%+v %v", o, err)
	}
}
