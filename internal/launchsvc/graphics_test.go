package launchsvc

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

var testOffer = &components.Graphics{
	Recommended: "dx12",
	Choices:     []components.GraphicsChoice{{ID: "dx12", Label: "DX12", Args: []string{"-dx12"}}, {ID: "vulkan", Label: "Vulkan"}},
}

func TestChosenGraphicsResolvesGameThenProfile(t *testing.T) {
	var st settings.Settings
	sc := settings.Scope{Game: "peak"}
	if _, ok := chosenGraphics(testOffer, st, sc, nil); ok {
		t.Fatal("an unset setting chose something")
	}
	if err := settings.ApplyKeyGame(&st, "graphicsApi", "dx12", "peak"); err != nil {
		t.Fatal(err)
	}
	if c, ok := chosenGraphics(testOffer, st, sc, nil); !ok || !slices.Equal(c.Args, []string{"-dx12"}) {
		t.Errorf("game choice = %+v, %v", c, ok)
	}
	if c, ok := chosenGraphics(testOffer, st, sc, map[string]string{"graphicsApi": "vulkan"}); !ok || len(c.Args) != 0 {
		t.Errorf("profile override = %+v, %v; want Vulkan with no arguments", c, ok)
	}
	if _, ok := chosenGraphics(testOffer, st, sc, map[string]string{"graphicsApi": "gone"}); ok {
		t.Error("an id the catalog lacks chose something")
	}
	if _, ok := chosenGraphics(nil, st, sc, nil); ok {
		t.Error("a game with no offer chose something")
	}
}
