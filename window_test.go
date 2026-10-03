package main

import "testing"

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
