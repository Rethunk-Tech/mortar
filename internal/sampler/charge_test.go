package sampler

import "testing"

func TestChargeUsesInnermostMappedAssembly(t *testing.T) {
	samples := []Sample{
		{Timestamp: 0, ThreadID: 7, Frames: []Frame{{Assembly: "Game"}, {Assembly: "Spin"}}},
		{Timestamp: 100_000_000, ThreadID: 7, Frames: []Frame{{Assembly: "Spin"}}},
		{Timestamp: 250_000_000, ThreadID: 7, Frames: []Frame{{Assembly: "Game"}}},
		{Timestamp: 400_000_000, ThreadID: 7, Frames: []Frame{{Assembly: "Game"}}},
		{Timestamp: 500_000_000, ThreadID: 8, Frames: []Frame{{Assembly: "Spin"}}},
	}
	result := Charge(samples, 7, map[string]string{"Spin": "FixtureMod"})
	if result.ThreadSamples != 4 {
		t.Fatalf("thread samples = %d, want 4", result.ThreadSamples)
	}
	if result.Mods["FixtureMod"] != 250 {
		t.Fatalf("mod time = %d, want 250ms", result.Mods["FixtureMod"])
	}
	if result.OtherMs != 150 {
		t.Fatalf("other time = %d, want 150ms", result.OtherMs)
	}
}
