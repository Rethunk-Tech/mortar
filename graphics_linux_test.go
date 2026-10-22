package main

import (
	"os"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestGraphicsModeSetsWebKitRendering(t *testing.T) {
	const dmabuf, compositing = "WEBKIT_DISABLE_DMABUF_RENDERER", "WEBKIT_DISABLE_COMPOSITING_MODE"
	if _, set := startEnv()[dmabuf]; set {
		t.Skip("the test's own environment sets " + dmabuf)
	}
	for mode, want := range map[string][2]string{
		settings.GraphicsAutomatic:  {"", ""},
		settings.GraphicsSmooth:     {"0", "1"},
		settings.GraphicsCompatible: {"1", ""},
	} {
		t.Setenv(dmabuf, "")
		t.Setenv(compositing, "")
		applyGraphics(mode)
		if got := [2]string{os.Getenv(dmabuf), os.Getenv(compositing)}; got != want {
			t.Errorf("%s: %s, %s = %q, want %q", mode, dmabuf, compositing, got, want)
		}
	}
}
