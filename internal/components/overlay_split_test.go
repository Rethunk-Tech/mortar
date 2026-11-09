package components

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A catalog that adds the overlay component still reaches a build that predates it: Decode drops only the entries it
// cannot validate, so the companion and the game beside them keep working.
func TestAnOlderBuildReadsACatalogWithAnOverlayComponent(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "overlay-split.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Components) != 2 {
		t.Fatalf("components = %+v, want the bridge and the overlay", got.Components)
	}
	overlay, ok := findComponent(got.Components, "lethal-company", "overlay")
	if !ok || overlay.Kind != "overlay" || overlay.Version != "0.1.0" {
		t.Fatalf("overlay = %+v, %v", overlay, ok)
	}
	if g, ok := findGame(got.Games, "lethal-company"); !ok || g.Loaders[0].Overlay != "overlay" || g.Loaders[0].Companion != "bridge" {
		t.Fatalf("game = %+v, %v", g, ok)
	}

	// What a build that knows only loader and bridge keeps of the same catalog: the unknown kind is left out with a
	// warning and nothing else is lost.
	unknown := got
	unknown.Components = append([]Component(nil), got.Components...)
	unknown.Components[1].Kind = "future-kind"
	kept, dropped := withoutUnreadable(unknown)
	if dropped != 0 || len(kept.Components) != 1 || kept.Components[0].Name != "bridge" {
		t.Fatalf("kept = %+v, dropped = %d", kept.Components, dropped)
	}
	if err := kept.Validate(); err != nil {
		t.Fatalf("what an older build keeps does not validate: %v", err)
	}
}

// A game names its overlay component beside the companion, so a catalog entry that says which component holds the
// overlay round-trips.
func TestAGameNamesTheOverlayComponent(t *testing.T) {
	body := `{"serial":1,"games":[{"id":"g","name":"G","marker":"m","deploy":"profile",` +
		`"loaders":[{"id":"bepinex5","name":"BepInEx 5","companion":"bridge","overlay":"overlay"}]}]}`
	m, err := Decode([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if g, ok := findGame(m.Games, "g"); !ok || g.Loaders[0].Overlay != "overlay" {
		t.Fatalf("game = %+v", g)
	}
	// A game that names none still validates: its overlay settings come from its companion alone.
	plain := strings.Replace(body, `,"overlay":"overlay"`, "", 1)
	if _, err := Decode([]byte(plain)); err != nil {
		t.Fatalf("a game without an overlay component: %v", err)
	}
}

// The generator resolves the same kinds the client validates, so a source entry for an overlay is not refused.
func TestAnOverlayIsAKnownComponentKind(t *testing.T) {
	c := Component{
		Game: "g", Name: "overlay", Kind: "overlay", Version: "1", Tag: "v1", Asset: "a.zip",
		Source: Source{Host: "github.com", Owner: "o", Repo: "r"},
		SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("overlay component refused: %v", err)
	}
	c.Kind = "something-else"
	if err := c.Validate(); err == nil {
		t.Fatal("an unknown component kind was accepted")
	}
}
