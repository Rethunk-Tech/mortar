package customgame

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// useTempData points the data folder at a fresh one and forgets the games it registers afterwards.
func useTempData(t *testing.T) {
	t.Helper()
	datadirtest.Use(t, t.TempDir())
	t.Cleanup(func() { components.SetUserGames(nil) })
}

// makeExe is a file that stands in for a game's executable, in a folder named after the game.
func makeExe(t *testing.T, folder string) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), folder, "game.exe")
	if err := os.MkdirAll(filepath.Dir(exe), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("MZ"), 0o600); err != nil {
		t.Fatal(err)
	}
	return exe
}

func TestAddRefusesWhatItCannotUse(t *testing.T) {
	useTempData(t)
	exe := makeExe(t, "My Game")
	abs := t.TempDir()
	for _, tc := range []struct {
		name, game, exe, loader, community, want string
	}{
		{"empty name", "", exe, Folder, "", "a name of up to 60 characters"},
		{"61-rune name", strings.Repeat("ü", 61), exe, Folder, "", "a name of up to 60 characters"},
		{"relative exe", "My Game", "game.exe", Folder, "", "full path"},
		{"missing exe", "My Game", filepath.Join(abs, "gone.exe"), Folder, "", "not there"},
		{"no letters or digits", "!!!", exe, Folder, "", "letters or digits"},
		{"unknown loader", "My Game", exe, "nonsense", "", "not a loader a custom game can use"},
		{"shimloader without a community", "My Game", exe, Shimloader, "", "Thunderstore community"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Add(tc.game, tc.exe, tc.loader, tc.community); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Add = %v, want %q", err, tc.want)
			}
			if kind := usererr.KindOf(mustFail(t, func() error { _, _, err := Add(tc.game, tc.exe, tc.loader, tc.community); return err })); kind != usererr.Invalid {
				t.Errorf("kind %q", kind)
			}
			if games, err := List(); err != nil || len(games) != 0 {
				t.Fatalf("a refused Add saved %+v, %v", games, err)
			}
		})
	}
	// A 60-rune name is still within the limit.
	if _, _, err := Add("Game"+strings.Repeat("ü", 56), exe, Folder, ""); err != nil {
		t.Fatalf("a 60-rune name: %v", err)
	}
}

func mustFail(t *testing.T, fn func() error) error {
	t.Helper()
	err := fn()
	if err == nil {
		t.Fatal("no error")
	}
	return err
}

func TestAddSuffixesEveryCollisionBeyondTheFirst(t *testing.T) {
	useTempData(t)
	exe := makeExe(t, "My Game")
	var ids []string
	for range 3 {
		e, _, err := Add("My Game", exe, Folder, "")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, e.ID)
	}
	want := []string{"custom-my-game", "custom-my-game-2", "custom-my-game-3"}
	if !slices.Equal(ids, want) {
		t.Fatalf("ids %v, want %v", ids, want)
	}
	games, err := List()
	if err != nil || len(games) != 3 {
		t.Fatalf("saved %+v, %v", games, err)
	}
}

func TestAddRefusesPastTheGameCap(t *testing.T) {
	useTempData(t)
	exe := makeExe(t, "My Game")
	for i := range maxGames {
		if _, _, err := Add(fmt.Sprintf("Game %d", i), exe, Folder, ""); err != nil {
			t.Fatalf("game %d: %v", i, err)
		}
	}
	if _, _, err := Add("One More", exe, Folder, ""); err == nil || !strings.Contains(err.Error(), "too many custom games") {
		t.Fatalf("Add past the cap = %v", err)
	}
}

func TestListRefusesAMalformedOrNewerFile(t *testing.T) {
	useTempData(t)
	p, err := path()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		kind usererr.Kind
	}{
		{"malformed JSON", `{"games": [`, usererr.Damaged},
		{"a newer format", `{"formatVersion": 99, "games": []}`, usererr.Invalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(p, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := List()
			if err == nil {
				t.Fatal("List read it")
			}
			if kind := usererr.KindOf(err); kind != tc.kind {
				t.Fatalf("kind %q, want %q: %v", kind, tc.kind, err)
			}
		})
	}
	if games, err := List(); err == nil || games != nil {
		t.Fatalf("List = %v, %v", games, err)
	}
}

func TestRegisterKeepsTheGoodEntriesAndNamesTheBadOne(t *testing.T) {
	useTempData(t)
	games := []Entry{
		{ID: "custom-good", Name: "Good", Exe: "game.exe", Loader: Folder},
		{ID: "custom-bad", Name: "Bad", Exe: "game.exe", Loader: "nonsense"},
		{ID: "custom-also-good", Name: "Also Good", Exe: "game.exe", Loader: BepInEx5},
	}
	err := Register(games)
	if err == nil || !strings.Contains(err.Error(), "custom-bad") {
		t.Fatalf("Register = %v", err)
	}
	if g, ok := components.Game("custom-good"); !ok || g.Name != "Good" {
		t.Errorf("the good entry before the bad one did not register")
	}
	if g, ok := components.Game("custom-also-good"); !ok || g.Name != "Also Good" {
		t.Errorf("the good entry after the bad one did not register")
	}
	if _, ok := components.Game("custom-bad"); ok {
		t.Error("the bad entry registered")
	}
}

func TestRemoveRefusesAnUnknownID(t *testing.T) {
	useTempData(t)
	err := Remove("custom-missing")
	if err == nil || !strings.Contains(err.Error(), "no such custom game") {
		t.Fatalf("Remove = %v", err)
	}
	if kind := usererr.KindOf(mustFail(t, func() error { return Remove("custom-missing") })); kind != usererr.NotFound {
		t.Fatalf("kind %q, want not_found", kind)
	}
	if games, err := List(); err != nil || len(games) != 0 {
		t.Fatalf("a refused Remove left %+v, %v", games, err)
	}
}

func TestRemoveForgetsTheGameAndItsRegistration(t *testing.T) {
	useTempData(t)
	exe := makeExe(t, "My Game")
	e, _, err := Add("My Game", exe, Folder, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := components.Game(e.ID); !ok {
		t.Fatal("Add did not register the game")
	}
	if err := Remove(e.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := components.Game(e.ID); ok {
		t.Error("Remove left the game registered")
	}
	if games, err := List(); err != nil || len(games) != 0 {
		t.Fatalf("Remove left %+v, %v", games, err)
	}
}

func TestAListingOfNoGamesIsEmptyNotNil(t *testing.T) {
	useTempData(t)
	games, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if games == nil {
		t.Error("List returned nil for no file; JSON prints it as null")
	}
}
