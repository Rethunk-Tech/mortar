package nexussvc

import "testing"

func TestPageAvailability(t *testing.T) {
	t.Parallel()
	hidden := PageAvailability("published", false, "2026-01-02", "2025-01-01")
	if hidden.Kind != "hidden" || hidden.Date != "2026-01-02" {
		t.Fatalf("hidden: %+v", hidden)
	}
	removed := PageAvailability("under_moderation", true, "", "2024-06-01")
	if removed.Kind != "removed" || removed.Date != "2024-06-01" {
		t.Fatalf("removed: %+v", removed)
	}
	ok := PageAvailability("published", true, "x", "y")
	if ok.Kind != "" {
		t.Fatalf("published: %+v", ok)
	}
}
