package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/profile"
)

func TestProfileHealthCLI(t *testing.T) {
	at := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
	results := map[string]any{
		"profile.health": []profile.HealthPoint{
			{At: at, Problems: 2, Warnings: 1, Updates: 4},
		},
	}
	r := invoke(t, results, "profile", "health", "stardew", "Farm")
	if r.code != 0 || !strings.Contains(r.out, "PROBLEMS") || !strings.Contains(r.out, "2") {
		t.Fatalf("health: %+v", r)
	}
	if got := r.calls[0]; got.method != "profile.health" || got.params.Profile != "Farm" {
		t.Fatalf("health params: %+v", got)
	}
	r = invoke(t, results, "profile", "health", "stardew", "Farm", "--json")
	if r.code != 0 || !strings.Contains(r.out, `"problems": 2`) {
		t.Fatalf("health json: %q", r.out)
	}
}
