package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

const emptySHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func TestReadPins(t *testing.T) {
	dir := t.TempDir()
	t.Run("missing file", func(t *testing.T) {
		pins, err := readPins(filepath.Join(dir, "absent.json"))
		if err != nil || pins != nil {
			t.Fatalf("got pins=%v err=%v", pins, err)
		}
	})
	t.Run("invalid pin", func(t *testing.T) {
		path := filepath.Join(dir, "bad.json")
		if err := os.WriteFile(path, []byte(`{"pins":[{"kind":"component","game":"g","name":"n","version":"1","sha256":"nope"}]}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readPins(path); err == nil {
			t.Fatal("expected invalid pin")
		}
	})
}

func TestWritePinsSorts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pins.json")
	pins := []components.Pin{
		{Kind: components.PinPack, Game: "g", Name: "Bravo", Version: "1.0.0", SHA256: emptySHA},
		{Kind: components.PinComponent, Game: "g", Name: "alpha", Version: "2.0.0", SHA256: emptySHA},
		{Kind: components.PinComponent, Game: "g", Name: "Alpha", Version: "1.0.0", SHA256: emptySHA},
		{Kind: components.PinComponent, Game: "a", Name: "zed", Version: "1.0.0", SHA256: emptySHA},
	}
	if err := writePins(path, pins); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file components.PinFile
	if err := json.Unmarshal(body, &file); err != nil {
		t.Fatal(err)
	}
	want := [][4]string{
		{components.PinComponent, "a", "zed", "1.0.0"},
		{components.PinComponent, "g", "Alpha", "1.0.0"},
		{components.PinComponent, "g", "alpha", "2.0.0"},
		{components.PinPack, "g", "Bravo", "1.0.0"},
	}
	if len(file.Pins) != len(want) {
		t.Fatalf("got %d pins", len(file.Pins))
	}
	for i, p := range file.Pins {
		if p.Kind != want[i][0] || p.Game != want[i][1] || p.Name != want[i][2] || p.Version != want[i][3] {
			t.Fatalf("index %d: %+v want %v", i, p, want[i])
		}
	}
}

func TestPinSetAddIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pins.json")
	set := &pinSet{file: path, have: map[string]bool{}}
	first := components.Pin{Kind: components.PinComponent, Game: "g", Name: "Mod", Version: "1.0.0", SHA256: emptySHA}
	if err := set.add(first); err != nil {
		t.Fatal(err)
	}
	dup := first
	dup.SHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := set.add(dup); err != nil {
		t.Fatal(err)
	}
	if len(set.pins) != 1 || set.pins[0].SHA256 != emptySHA {
		t.Fatalf("got %+v", set.pins)
	}
}

func TestPackTargets(t *testing.T) {
	game := func(id, loaderID string) components.GameInfo {
		return components.GameInfo{
			ID:      id,
			Sources: []components.GameSource{{ID: "thunderstore"}},
			Loaders: []components.GameLoader{{ID: loaderID, Package: "BepInEx-BepInExPack", Major: 5}},
		}
	}
	games := []components.GameInfo{
		game("game-a", "bepinex5"),
		game("game-b", "bepinex5"),
		game("stardew", "smapi"),
		game("amongus", "melonloader"),
	}
	targets := packTargets(games)
	if len(targets) != 1 {
		t.Fatalf("got %d targets, want 1 bepinex pack", len(targets))
	}
	got := targets[0].game
	if got.ID != "game-a" {
		t.Fatalf("got game %q", got.ID)
	}
	key := "bepinex5|" + got.LoaderPackage() + "|" + strconv.Itoa(got.LoaderMajor("bepinex5"))
	if key != "bepinex5|BepInEx-BepInExPack|5" {
		t.Fatalf("got key %q", key)
	}
}

func TestPinPackVersionsStopsOnCanceledContext(t *testing.T) {
	dir := t.TempDir()
	set := &pinSet{file: filepath.Join(dir, "pins.json"), have: map[string]bool{}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n := 0
	target := packTarget{
		game: components.GameInfo{ID: "g", Loaders: []components.GameLoader{{ID: "bepinex", Package: "BepInEx-BepInExPack"}}},
		rel: stubReleases{fetch: func(ctx context.Context, _ components.GameInfo, version, dest string) error {
			n++
			if n >= 2 {
				cancel()
				return ctx.Err()
			}
			return os.WriteFile(dest, []byte("pack-"+version), 0o600)
		}},
	}
	stop, err := pinPackVersions(ctx, set, dir, target, []string{"1.0.0", "1.1.0", "1.2.0"})
	if err != nil {
		t.Fatal(err)
	}
	if !stop {
		t.Fatal("expected stop")
	}
	if len(set.pins) != 1 || set.pins[0].Version != "1.0.0" {
		t.Fatalf("kept %+v", set.pins)
	}
}

type stubReleases struct {
	fetch func(ctx context.Context, game components.GameInfo, version, dest string) error
}

func (s stubReleases) Versions(context.Context, components.GameInfo) ([]string, error) {
	return nil, nil
}

func (s stubReleases) Latest(context.Context, components.GameInfo) (string, error) {
	return "", nil
}

func (s stubReleases) Fetch(ctx context.Context, game components.GameInfo, version, dest string) error {
	return s.fetch(ctx, game, version, dest)
}

var _ loader.Releases = stubReleases{}
