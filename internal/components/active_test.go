package components

import (
	"testing"
)

// The catalog in use answers a component by name from the selected manifest, and from the bundled one before a
// manifest is selected, so startup work ahead of the network fetch still finds what it installs.
func TestActiveComponentAnswersBeforeAndAfterUse(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	Use(nil)
	if c, ok := ActiveComponent("stardew", "bridge"); !ok || c.Version == "" {
		t.Fatalf("the bundled component = %+v, %v", c, ok)
	}
	c := NewClient(nil)
	c.SetManifest(Manifest{Serial: 1, Components: []Component{{
		Game: "stardew", Name: "bridge", Kind: "bridge", Version: "9.9.9",
		Source: Source{Host: "github.com", Owner: "o", Repo: "r"}, Tag: "v1", Asset: "a.zip",
	}}})
	Use(c)
	if Active() != c {
		t.Fatal("Use did not select the client")
	}
	if got, ok := ActiveComponent("stardew", "bridge"); !ok || got.Version != "9.9.9" {
		t.Fatalf("ActiveComponent = %+v, %v", got, ok)
	}
	if !HasComponent("stardew", "bridge") || HasComponent("stardew", "nonsense") {
		t.Error("HasComponent does not follow the selected manifest")
	}
	// A game the selected manifest lacks falls back to the bundled manifest's entry.
	if _, ok := ActiveComponent("lethal-company", "bridge"); !ok {
		t.Error("a component the selected manifest lacks is not found in the bundled one")
	}
}

// Authors are the selected manifest's aliases, else the bundled manifest's, so Browse folds them either way.
func TestAuthorsFollowTheSelectedManifest(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	c := NewClient(nil)
	c.SetManifest(Manifest{Serial: 1, Authors: []AuthorAliases{{Names: []string{"Evaisa", "EvaisaDev"}}}})
	Use(c)
	if got := Authors(); len(got) != 1 || got[0].Names[0] != "Evaisa" {
		t.Fatalf("Authors = %+v", got)
	}
	Use(NewClient(nil))
	if got := Authors(); len(got) != 0 {
		t.Logf("the bundled manifest names %d authors", len(got))
	}
}

// The bundled manifest's accepted game versions answer for its loader, and a component it lacks declares none.
func TestBundledAcceptedReadsTheBundledComponent(t *testing.T) {
	c, ok := findComponent(mustBundled(t).Components, "stardew", "smapi")
	if !ok || c.Accepted == "" {
		t.Fatal("the bundled catalog declares no accepted versions for Stardew's SMAPI")
	}
	if got := BundledAccepted("stardew", "smapi"); got != c.Accepted {
		t.Errorf("BundledAccepted = %q, want %q", got, c.Accepted)
	}
	if got := BundledAccepted("stardew", "nonsense"); got != "" {
		t.Errorf("a component with no entry accepted %q", got)
	}
}

func mustBundled(t *testing.T) Manifest {
	t.Helper()
	m, err := BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// A pin the catalog lists answers for the version it names, and HasPins says an unpinned version of it is
// unavailable; a component's own entry is a pin too.
func TestPinsAnswerForTheVersionTheyName(t *testing.T) {
	sum := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	c := NewClient(nil)
	c.SetManifest(Manifest{
		Serial: 1,
		Components: []Component{{
			Game: "stardew", Name: "bridge", Kind: "bridge", Version: "2", SHA256: sum,
			Source: Source{Host: "github.com", Owner: "o", Repo: "r"}, Tag: "v2", Asset: "a.zip",
		}},
		Pins: []Pin{{Kind: PinPack, Name: "BepInEx-BepInExPack", Version: "5.4.21", SHA256: sum}},
	})
	if got, ok := c.Pin(PinPack, "", "bepinex-bepinexpack", "5.4.21"); !ok || got != sum {
		t.Errorf("a pack pin = %q, %v", got, ok)
	}
	if got, ok := c.Pin(PinComponent, "stardew", "bridge", "2"); !ok || got != sum {
		t.Errorf("a component's own version = %q, %v", got, ok)
	}
	if _, ok := c.Pin(PinPack, "", "BepInEx-BepInExPack", "5.4.22"); ok {
		t.Error("an unpinned version answered a digest")
	}
	if !c.HasPins(PinPack, "", "bepinex-bepinexpack") {
		t.Error("HasPins missed a pinned package")
	}
	if c.HasPins(PinPack, "", "some-other-pack") {
		t.Error("HasPins named a package nothing pins")
	}
}

// A game's target answers by id, names the folder it lays out in, and a loader's Doorstop proxy falls back to
// winhttp.dll when the game's pack ships none.
func TestTargetAndLoaderDefaults(t *testing.T) {
	t.Parallel()
	g := GameInfo{
		MarkerDir: "BepInEx",
		Loaders:   []GameLoader{{ID: "bepinex5"}, {ID: "bepinex6", Proxy: "version.dll"}},
		Targets: []TargetDef{
			{ID: "mods", Root: "{profileMods}"},
			{ID: "config", Root: "{profile}/{sub}"},
			{ID: "root", Root: "{profile}"},
		},
	}
	if got := g.Targets[0].ProfileFolder(); got != "mods" {
		t.Errorf("a profileMods target lays out in %q", got)
	}
	if got := g.Targets[1].ProfileFolder(); got != "{sub}" {
		t.Errorf("a subfolder target lays out in %q", got)
	}
	if got := g.Targets[2].ProfileFolder(); got != "" {
		t.Errorf("a profile-root target lays out in %q", got)
	}
	if t2, ok := g.Target("config"); !ok || t2.Root != "{profile}/{sub}" {
		t.Errorf("Target = %+v, %v", t2, ok)
	}
	if _, ok := g.Target("nonsense"); ok {
		t.Error("an unknown target id named one")
	}
	if got := g.DoorstopProxy(); got != "version.dll" {
		t.Errorf("DoorstopProxy = %q", got)
	}
	if got := (GameInfo{Loaders: []GameLoader{{ID: "bepinex5"}}}).DoorstopProxy(); got != "winhttp.dll" {
		t.Errorf("a pack with no proxy falls back to %q", got)
	}
}

// The loader's Thunderstore pack answers by package name across the whole catalog, and a BepInEx pack's major comes
// from the game that installs it (5 for every pack no game overrides).
func TestBepInExPackageLookups(t *testing.T) {
	t.Parallel()
	if !IsLoaderPackage(DefaultLoaderPackage) || !IsBepInExPackage(DefaultLoaderPackage) {
		t.Fatal("the default pack is not a loader package")
	}
	if IsLoaderPackage("Some-CommunityThing") || IsBepInExPackage("Some-CommunityThing") {
		t.Error("an ordinary package is a loader package")
	}
	if got := LoaderPackageMajor(DefaultLoaderPackage); got != 5 {
		t.Errorf("the default pack's major = %d", got)
	}
	var game string
	for _, g := range Games() {
		if p := g.LoaderPackage(); p != DefaultLoaderPackage && IsBepInExPackage(p) {
			game = g.ID
			break
		}
	}
	if game == "" {
		t.Skip("no catalog game ships its own BepInEx pack")
	}
	g, _ := Game(game)
	if !IsBepInExPackage(g.LoaderPackage()) {
		t.Errorf("%s's pack is not a BepInEx package", game)
	}
	if got, want := LoaderPackageMajor(g.LoaderPackage()), g.LoaderMajor("bepinex5"); want != 0 && got != want {
		t.Errorf("%s's pack major = %d, want %d", game, got, want)
	}
}

// A check that runs outside an install asks an entry with a game-version range nothing, since the version is unknown.
func TestKnownBrokenForAnswersNothingWithoutASource(t *testing.T) {
	if got := KnownBrokenFor(t.Context(), "no-such-game"); len(got.Entries) != 0 {
		t.Fatalf("KnownBrokenFor = %+v", got)
	}
}
