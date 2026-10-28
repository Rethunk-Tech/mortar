package gamesettings

import "testing"

func TestValidateRejectsInvalidValues(t *testing.T) {
	mode := "unsupported"
	if err := (Settings{WindowMode: &mode}).Validate(); err == nil {
		t.Fatal("invalid window mode was accepted")
	}
	volume := 101
	if err := (Settings{MusicVolumeLevel: &volume}).Validate(); err == nil {
		t.Fatal("invalid volume was accepted")
	}
}
