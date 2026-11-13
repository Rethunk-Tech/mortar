package components

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"
)

// A key arrives either as the raw 32 bytes or as PEM, and anything else is refused before a signature is checked.
func TestParsePublicKeyReadsBothSpellings(t *testing.T) {
	t.Parallel()
	public, private := newKey(t)
	body := []byte("manifest")
	sig := Sign(ListCatalog, body, private)
	if err := Verify(ListCatalog, body, sig, public); err != nil {
		t.Fatalf("a raw key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if err := Verify(ListCatalog, body, sig, pemKey); err != nil {
		t.Fatalf("a PEM key: %v", err)
	}
	if _, err := parsePublicKey([]byte("not a key")); err == nil {
		t.Error("a non-PEM key parsed")
	}
	if _, err := parsePublicKey(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("garbage")})); err == nil {
		t.Error("a PEM block that is not a key parsed")
	}
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaDer, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parsePublicKey(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: rsaDer})); err == nil {
		t.Error("a key that is not Ed25519 parsed")
	}
}

// A signature arrives as raw 64 bytes, base64 of them, or with surrounding whitespace, and any other length is refused.
func TestParseSignatureReadsRawAndBase64(t *testing.T) {
	t.Parallel()
	_, private := newKey(t)
	sig := ed25519.Sign(private, []byte("x"))
	for name, spelling := range map[string][]byte{
		"raw":     sig,
		"spaced":  []byte(" \n" + base64.StdEncoding.EncodeToString(sig) + "\n"),
		"base64":  []byte(base64.StdEncoding.EncodeToString(sig)),
		"raw-url": []byte(base64.RawStdEncoding.EncodeToString(sig)),
	} {
		got, err := parseSignature(spelling)
		if err != nil || string(got) != string(sig) {
			t.Errorf("%s signature = %x, %v", name, got, err)
		}
	}
	if _, err := parseSignature([]byte("c2hvcnQ=")); err == nil {
		t.Error("a short signature parsed")
	}
}

// The overlay setting applies only to a companion released before the split; a version that cannot be compared is
// treated as one from before it, so the overlay never loads beside a build that may bind the port.
func TestServesOverlayOnlyForAnOldCompanion(t *testing.T) {
	t.Parallel()
	for version, want := range map[string]bool{"0.6.9": true, "0.7.0": false, "0.7.1": false} {
		if got := ServesOverlay(Component{Version: version, Source: Source{Repo: "mortar-bepinex-bridge"}}); got != want {
			t.Errorf("bepinex bridge %s serves overlay = %v, want %v", version, got, want)
		}
	}
	if got := ServesOverlay(Component{Version: "0.1.0", Source: Source{Repo: "mortar-smapi-bridge"}}); !got {
		t.Error("an old SMAPI bridge does not serve the overlay")
	}
	if ServesOverlay(Component{Version: "0.1.0", Source: Source{Repo: "something-else"}}) {
		t.Error("a repo without a split serves the overlay")
	}
}

// A rule route places a package's files below a folder named for it when the route tracks, and straight into the
// route folder when it does not.
func TestLoaderRoutePlaceFollowsTrack(t *testing.T) {
	t.Parallel()
	plain := LoaderRoute{Dir: "BepInEx/plugins"}
	sub := LoaderRoute{Dir: "BepInEx/plugins", Track: TrackSubdir}
	keep := LoaderRoute{Dir: "BepInEx/plugins", Track: TrackKeep}
	if got := plain.Place("Pack", "a.dll", "Sub/a.dll"); got != "BepInEx/plugins/a.dll" {
		t.Errorf("plain Place = %q", got)
	}
	if got := sub.Place("Pack", "a.dll", "Sub/a.dll"); got != "BepInEx/plugins/Pack/a.dll" {
		t.Errorf("subdir Place = %q", got)
	}
	if got := keep.Place("Pack", "a.dll", "Sub/a.dll"); got != "BepInEx/plugins/Pack/Sub/a.dll" {
		t.Errorf("keep Place = %q", got)
	}
}

// A route that could place a file outside the profile root is refused at decode, so a catalog edit cannot escape.
func TestLoaderRouteRefusesUnsafeFoldersAndExtensions(t *testing.T) {
	t.Parallel()
	cases := []LoaderRoute{
		{Dir: ""},
		{Dir: "."},
		{Dir: "/abs"},
		{Dir: "a/../b"},
		{Dir: `a\b`},
		{Dir: "a:b"},
		{Dir: "ok", Track: "sideways"},
		{Dir: "ok", Extensions: []string{"dll"}},
		{Dir: "ok", Extensions: []string{".d/ll"}},
		{Dir: "ok", Extensions: []string{".dl*l"}},
		{Dir: "ok", Extensions: []string{".d:ll"}},
	}
	for _, r := range cases {
		if err := r.Validate(); err == nil {
			t.Errorf("%+v validated", r)
		}
	}
	if err := (LoaderRoute{Dir: "BepInEx/plugins", Track: TrackKeep, Extensions: []string{".dll"}, Default: true}).Validate(); err != nil {
		t.Errorf("a usable route was refused: %v", err)
	}
}

// A package path that climbs out, names a drive, or names a package with a separator is refused, so no entry can be
// written above its folder. Package metadata at the top never reaches a profile.
func TestCleanPackagePathRefusesEscapes(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ rel, pkg string }{
		{"../a.dll", "Pack"},
		{"a/../../a.dll", "Pack"},
		{"/a.dll", "Pack"},
		{"c:/a.dll", "Pack"},
		{"a.dll", ""},
		{"a.dll", "a/b"},
		{"a.dll", "."},
		{"a.dll", ".."},
	} {
		if got, ok := CleanPackagePath(c.rel, c.pkg); ok {
			t.Errorf("CleanPackagePath(%q, %q) = %q", c.rel, c.pkg, got)
		}
	}
	if got, ok := CleanPackagePath(`Sub\a.dll`, "Pack"); !ok || got != "Sub/a.dll" {
		t.Errorf("a Windows path = %q, %v", got, ok)
	}
	if _, ok := PackageFile("manifest.json", "Pack"); ok {
		t.Error("the package metadata reached a profile")
	}
	if got, ok := PackageFile("Sub/manifest.json", "Pack"); !ok || got != "Sub/manifest.json" {
		t.Errorf("a nested manifest = %q, %v", got, ok)
	}
}

// A file goes to the first route whose folder it sits below, else by its longest matching extension, else to the
// default route; no route taking it is the empty answer.
func TestPlaceByRulesPrefersFolderThenLongestExtensionThenDefault(t *testing.T) {
	t.Parallel()
	routes := []LoaderRoute{
		{Dir: "BepInEx/plugins"},
		{Dir: "BepInEx/core", Extensions: []string{".dll"}},
		{Dir: "BepInEx/patchers", Extensions: []string{".plugin.dll"}},
		{Dir: "BepInEx/config", Default: true},
	}
	for _, c := range []struct{ path, want string }{
		{"plugins/Mod.dll", "BepInEx/plugins/Mod.dll"},
		{"Mod.dll", "BepInEx/core/Mod.dll"},
		{"Mod.plugin.dll", "BepInEx/patchers/Mod.plugin.dll"},
		{"readme.txt", "BepInEx/config/readme.txt"},
	} {
		if got := PlaceByRules(c.path, "Pack", routes); got != c.want {
			t.Errorf("PlaceByRules(%q) = %q, want %q", c.path, got, c.want)
		}
	}
	if got := PlaceByRules("readme.txt", "Pack", routes[:3]); got != "" {
		t.Errorf("no default route still placed it at %q", got)
	}
	if _, ok := RouteByFolder(routes, "PLUGINS"); !ok {
		t.Error("a route folder is compared with case")
	}
	if r, ok := RouteByExtension(routes, "MOD.PLUGIN.DLL"); !ok || r.Dir != "BepInEx/patchers" {
		t.Errorf("an upper-case name went to %q, %v", r.Dir, ok)
	}
}

// A game with no loader entry of the asked id still names a place for the loader's files, and a BepInEx pack version
// is read from the catalog game that installs it.
func TestGameLoaderLookupsFallBack(t *testing.T) {
	t.Parallel()
	g := GameInfo{MarkerDir: "BepInEx", Loaders: []GameLoader{{ID: "bepinex5", PlaceDir: "BepInEx/core", Major: 5}}}
	if got := g.LoaderPlaceDir("bepinex5"); got != "BepInEx/core" {
		t.Errorf("LoaderPlaceDir of a named loader = %q", got)
	}
	if got := g.LoaderPlaceDir("smapi"); got != "BepInEx" {
		t.Errorf("LoaderPlaceDir of an unknown loader = %q", got)
	}
	if got := g.LoaderMajor("bepinex5"); got != 5 {
		t.Errorf("LoaderMajor = %d", got)
	}
	if got := g.LoaderMajor("smapi"); got != 0 {
		t.Errorf("LoaderMajor of an unknown loader = %d", got)
	}
	if got := g.LoaderNexusModID(); got != 0 {
		t.Errorf("a loader with no Nexus page = %d", got)
	}
}

// A game's sources answer by id, the keys it names, its Nexus domain and id, and its Steam app.
func TestGameSourceLookups(t *testing.T) {
	t.Parallel()
	g := GameInfo{
		Sources: []GameSource{{ID: "nexus", Key: "stardewvalley", GameID: 1303}, {ID: "thunderstore", Key: "stardew"}, {ID: "github"}},
		Stores:  GameStores{Steam: &SteamStore{AppID: "413150"}},
	}
	if got := g.SourceKeys(); len(got) != 2 || got["nexus"] != "stardewvalley" || got["thunderstore"] != "stardew" {
		t.Errorf("SourceKeys = %v", got)
	}
	if got := g.NexusDomain(); got != "stardewvalley" {
		t.Errorf("NexusDomain = %q", got)
	}
	if got := g.NexusID(); got != 1303 {
		t.Errorf("NexusID = %d", got)
	}
	if got := g.SteamAppID(); got != "413150" {
		t.Errorf("SteamAppID = %q", got)
	}
	bare := GameInfo{}
	if bare.NexusDomain() != "" || bare.NexusID() != 0 || bare.SteamAppID() != "" {
		t.Error("a game without Nexus or Steam named one")
	}
}

// A metadata feature is answered from the game's list, which is empty for a game that names none.
func TestHasMetadataReadsTheGamesList(t *testing.T) {
	t.Parallel()
	g := GameInfo{Metadata: []string{"stardew-dataset"}}
	if !g.HasMetadata("stardew-dataset") {
		t.Error("a listed feature is missing")
	}
	if g.HasMetadata("nonsense") || (GameInfo{}).HasMetadata("nonsense") {
		t.Error("an unlisted feature is present")
	}
}

// The browse route answers the game whose Nexus key is the domain, compared without case, and nothing for a domain no
// game names.
func TestGameByNexusDomainMatchesWithoutCase(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	// A user game is a catalog entry of this test's own, so the answer does not depend on the signed catalog's games.
	SetUserGames([]GameInfo{{ID: "test-game", Name: "Test", Marker: "marker", Sources: []GameSource{{ID: "nexus", Key: "TestDomain"}}}})
	t.Cleanup(func() { SetUserGames(nil) })
	g, ok := GameByNexusDomain("testdomain")
	if !ok || g.ID != "test-game" {
		t.Fatalf("GameByNexusDomain = %+v, %v", g, ok)
	}
	if _, ok := GameByNexusDomain("nobody-hosts-this"); ok {
		t.Error("an unhosted domain named a game")
	}
}
