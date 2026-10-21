package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClampWindowMovesOffscreenOntoAScreen(t *testing.T) {
	screens := []screenRect{{X: 0, Y: 0, Width: 1920, Height: 1080}}
	x, y, w, h := clampWindow(-4000, -4000, 400, 300, screens)
	if x != 0 || y != 0 {
		t.Fatalf("got %d,%d", x, y)
	}
	if w < minWindowWidth || h < minWindowHeight {
		t.Fatalf("size %dx%d", w, h)
	}
}

func TestClampWindowKeepsOnscreenPlacement(t *testing.T) {
	screens := []screenRect{{X: 0, Y: 0, Width: 1920, Height: 1080}}
	x, y, w, h := clampWindow(100, 80, 1280, 720, screens)
	if x != 100 || y != 80 || w != 1280 || h != 720 {
		t.Fatalf("got %d,%d %dx%d", x, y, w, h)
	}
}

func TestClosingMaximisedKeepsTheNormalBoundsAndRemembersMaximised(t *testing.T) {
	prev := windowGeom{X: 100, Y: 80, W: 1280, H: 720}
	full := windowGeom{X: -8, Y: -8, W: 1936, H: 1056}
	if got := closingGeom(prev, true, true, full); got != (windowGeom{X: 100, Y: 80, W: 1280, H: 720, Max: true}) {
		t.Errorf("maximised close saved %+v", got)
	}
	if got := closingGeom(windowGeom{}, false, true, full); got != (windowGeom{W: defaultWindowWidth, H: defaultWindowHeight, Max: true}) {
		t.Errorf("first maximised close saved %+v", got)
	}
	moved := windowGeom{X: 50, Y: 60, W: 1400, H: 900}
	if got := closingGeom(windowGeom{X: 1, Y: 1, W: 1280, H: 720, Max: true}, true, false, moved); got != moved {
		t.Errorf("normal close saved %+v", got)
	}
}

func TestWindowGeomWithoutMaxStillReads(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "window.json"), []byte(`{"x":10,"y":20,"w":1280,"h":720}`), 0o600); err != nil {
		t.Fatal(err)
	}
	g, ok := loadWindowGeom(dir, nil)
	if !ok || g.Max || g.W != 1280 {
		t.Fatalf("%+v %v", g, ok)
	}
}
