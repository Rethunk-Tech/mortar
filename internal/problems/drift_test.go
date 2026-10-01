package problems

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/profile"
)

func TestCountIncludesDrift(t *testing.T) {
	r := Result{
		Missing: []Missing{{}},
		Drift:   []profile.Drift{{Kind: profile.DriftUnknown, Folder: "loose", Key: "loose"}},
	}
	if r.Count() != 2 {
		t.Fatalf("Count() = %d", r.Count())
	}
}
