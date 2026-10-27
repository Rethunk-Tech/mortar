package control

import (
	"context"
	"strings"
	"testing"
)

func TestDevAddTakesAProfileByName(t *testing.T) {
	s, _ := servicesWithItems(t)
	if _, err := s.Profiles.Create("stardew", "ConfigSync Test"); err != nil {
		t.Fatal(err)
	}
	// With no Dev service the call fails after the profile is found, so the error says which step it reached.
	_, err := s.devAdd(context.Background(), Params{Game: "stardew", Path: "/p", Profile: "configsync test"})
	if err == nil || err.Error() != "mod development is unavailable" {
		t.Fatalf("by name: %v", err)
	}
	_, err = s.devAdd(context.Background(), Params{Game: "stardew", Path: "/p", Profile: "Nope"})
	if err == nil || !strings.Contains(err.Error(), `no stardew profile is named or has the id "Nope"`) {
		t.Fatalf("unknown name: %v", err)
	}
}
