package components

import (
	"strings"
	"testing"
)

// Every field a known-broken entry needs is refused when missing, so a list that lost one is never used.
func TestDecodeBrokenListRefusesIncompleteEntries(t *testing.T) {
	t.Parallel()
	body := func(entry string) []byte {
		return []byte(`{"game":"g","serial":3,"entries":[` + entry + `]}`)
	}
	if _, err := DecodeBrokenList(body(``), "g"); err != nil {
		t.Fatalf("an empty list: %v", err)
	}
	if _, err := DecodeBrokenList([]byte(`not json`), "g"); err == nil {
		t.Error("a body that is not JSON decoded")
	}
	if _, err := DecodeBrokenList(body(`{"ids":{"thunderstore":"A-B"},"through":"1","reason":"r"}`), "other"); err == nil {
		t.Error("a list for another game decoded")
	}
	if _, err := DecodeBrokenList([]byte(`{"game":"g","entries":[]}`), "g"); err == nil {
		t.Error("a list with no serial decoded")
	}
	for name, entry := range map[string]string{
		"no ids":       `{"through":"1","reason":"r"}`,
		"no through":   `{"ids":{"thunderstore":"A-B"},"reason":"r"}`,
		"no reason":    `{"ids":{"thunderstore":"A-B"},"through":"1"}`,
		"unknown kind": `{"ids":{"thunderstore":"A-B"},"through":"1","reason":"r","kind":"wobbly"}`,
		"bad version":  `{"ids":{"thunderstore":"A-B"},"through":"1","reason":"r","gameVersions":"1.6.x"}`,
	} {
		if _, err := DecodeBrokenList(body(entry), "g"); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

// An entry that holds only on some game versions cannot flag an install of unknown version, which is what GameScoped
// tells the check before it asks.
func TestGameScopedOnlyWhenTheEntryNamesVersions(t *testing.T) {
	t.Parallel()
	if (BrokenEntry{GameVersions: ">=50"}).GameScoped() != true || (BrokenEntry{}).GameScoped() != false {
		t.Error("GameScoped does not follow GameVersions")
	}
}

// A game version constraint uses every operator the list format allows, and a version it cannot order counts as met
// rather than as a false flag.
func TestInConstraintsCoversEveryOperator(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		version, constraints string
		want                 bool
	}{
		{"1.6.14", ">=1.6.14", true},
		{"1.6.13", ">=1.6.14", false},
		{"1.6.14", "<=1.6.14", true},
		{"1.6.15", "<=1.6.14", false},
		{"1.6.15", ">1.6.14", true},
		{"1.6.14", ">1.6.14", false},
		{"1.6.13", "<1.6.14", true},
		{"1.6.14", "=1.6.14", true},
		{"1.6.14", "=1.6.15", false},
		{"1.6.14", ">=1.6.0 <1.7.0", true},
		{"1.7.0", ">=1.6.0 <1.7.0", false},
		{"local build", ">=1.6.0", true},
		{"1.6.14", "", true},
	} {
		if got := inConstraints(c.version, c.constraints); got != c.want {
			t.Errorf("inConstraints(%q, %q) = %v, want %v", c.version, c.constraints, got, c.want)
		}
	}
}

// A pin names the release it vouches for and the digest, and a catalog entry that names neither is refused.
func TestPinValidate(t *testing.T) {
	t.Parallel()
	sum := strings.Repeat("a", 64)
	good := map[string]Pin{
		"component": {Kind: PinComponent, Game: "stardew", Name: "bridge", Version: "1", SHA256: sum},
		"pack":      {Kind: PinPack, Name: "BepInEx-BepInExPack", Version: "5.4.21", SHA256: sum},
	}
	for name, p := range good {
		if err := p.Validate(); err != nil {
			t.Errorf("%s pin: %v", name, err)
		}
	}
	bad := map[string]Pin{
		"component with no game": {Kind: PinComponent, Name: "bridge", Version: "1", SHA256: sum},
		"pack with a game":       {Kind: PinPack, Game: "stardew", Name: "pack", Version: "1", SHA256: sum},
		"unknown kind":           {Kind: "sideways", Name: "x", Version: "1", SHA256: sum},
		"no name":                {Kind: PinPack, Version: "1", SHA256: sum},
		"no version":             {Kind: PinPack, Name: "x", SHA256: sum},
		"short digest":           {Kind: PinPack, Name: "x", Version: "1", SHA256: "abc"},
	}
	for name, p := range bad {
		if err := p.Validate(); err == nil {
			t.Errorf("%s validated", name)
		}
	}
}

// A component's source is the host that serves it, and a host nothing serves from is refused before a download.
func TestSourceAndComponentValidate(t *testing.T) {
	t.Parallel()
	sum := strings.Repeat("b", 64)
	if err := (Source{Host: "GitHub.com", Owner: "o", Repo: "r"}).Validate(); err != nil {
		t.Errorf("a GitHub source: %v", err)
	}
	if err := (Source{Host: "thunderstore.io", Namespace: "N", Name: "p"}).Validate(); err != nil {
		t.Errorf("a Thunderstore source: %v", err)
	}
	for name, s := range map[string]Source{
		"unknown host":     {Host: "example.com"},
		"no owner":         {Host: "github.com", Repo: "r"},
		"no repo":          {Host: "github.com", Owner: "o"},
		"owner with slash": {Host: "github.com", Owner: "o/r", Repo: "r"},
		"no namespace":     {Host: "thunderstore.io", Name: "p"},
		"name with ?":      {Host: "thunderstore.io", Namespace: "N", Name: "p?"},
	} {
		if err := s.Validate(); err == nil {
			t.Errorf("%s validated", name)
		}
	}

	c := Component{
		Game: "stardew", Name: "bridge", Kind: "bridge", Version: "1", Tag: "v1", Asset: "bridge.zip",
		Source: Source{Host: "github.com", Owner: "o", Repo: "r"}, SHA256: sum,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("a resolved component: %v", err)
	}
	c.Tag = ".." // a tag that names the folder above, which the download joins to a URL
	if err := c.Validate(); err == nil {
		t.Error("a tag of .. validated")
	}
	c.Tag = "v1"
	c.SHA256 = strings.Repeat("z", 64)
	if err := c.Validate(); err == nil {
		t.Error("a digest that is not hex validated")
	}
}

// A player's own game has no store, so it validates against the folder the player chose and nothing else.
func TestValidateUserAcceptsAStorelessGame(t *testing.T) {
	t.Parallel()
	g := GameInfo{
		ID: "mine", Name: "Mine", Marker: "mine.dll", Enabled: true,
		Deploy: DeployProfile, Loaders: []GameLoader{{ID: "folder"}},
		Targets: []TargetDef{{ID: "mods", Root: "{profile}/Mods"}},
	}
	if err := g.Validate(); err == nil {
		t.Fatal("a game with no store validated without the user rule")
	}
	if err := g.ValidateUser(); err != nil {
		t.Fatalf("ValidateUser: %v", err)
	}
	bad := GameInfo{ID: "mine", Name: "Mine"}
	if err := bad.ValidateUser(); err == nil {
		t.Error("a user game with no marker validated")
	}
}

// Game answers a player's own game when the catalog has no entry for it, and nothing for an id nobody names.
func TestGameFallsBackToThePlayersOwnGames(t *testing.T) {
	t.Cleanup(func() { SetUserGames(nil) })
	SetUserGames([]GameInfo{{ID: "mine", Name: "Mine", Marker: "m"}})
	if g, ok := Game("mine"); !ok || g.Name != "Mine" {
		t.Fatalf("Game(mine) = %+v, %v", g, ok)
	}
	if _, ok := Game("nobody-names-this"); ok {
		t.Error("an unknown id named a game")
	}
	// The player's game never replaces a catalog entry of the same id, and Games appends it after the catalog's.
	if game, ok := Game("stardew"); !ok || game.ID != "stardew" || strings.EqualFold(game.Name, "Mine") {
		t.Errorf("Game(stardew) = %+v, %v", game, ok)
	}
	games := Games()
	if last := games[len(games)-1]; last.ID != "mine" {
		t.Errorf("the player's game is not last: %+v", games)
	}
}
