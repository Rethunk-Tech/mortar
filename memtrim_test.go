package main

import "testing"

func TestTrimOnceAfterABurstGoesQuiet(t *testing.T) {
	var w quietWatch
	trims := 0
	for _, cpu := range []float64{0.01, 0.01, 1.5, 0.9, 0.01, 0.01, 0.01, 0.01, 0.01, 0.01, 0.01, 0.01} {
		if w.step(cpu) {
			trims++
		}
	}
	if trims != 1 {
		t.Fatalf("trims = %d, want one after the burst", trims)
	}
	for range 20 {
		if w.step(0.05) {
			t.Fatal("background noise below a burst trimmed")
		}
	}
}
