package main

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/problems"
)

func TestOfficialUpdateCount(t *testing.T) {
	got := officialUpdateCount([]problems.Update{
		{Key: "a", UniqueID: "A"},
		{Key: "a", UniqueID: "A"},
		{Key: "b", UniqueID: "B", Unofficial: true},
		{Key: "c", UniqueID: "C"},
	})
	if got != 2 {
		t.Fatalf("official update count = %d, want 2", got)
	}
}
