package gamesettings

import (
	"bytes"
	"testing"
)

func TestPatchPreservesUntouchedBytes(t *testing.T) {
	windowed := "windowed"
	volume := 42
	input := append([]byte{0xef, 0xbb, 0xbf}, []byte("<root>\n<windowMode>fullscreen</windowMode>\n<unknown>x&y</unknown>\n<soundVolumeLevel>80</soundVolumeLevel></root>")...)
	got, err := Patch(input, Settings{WindowMode: &windowed, SoundVolumeLevel: &volume})
	if err != nil {
		t.Fatal(err)
	}
	want := append([]byte{0xef, 0xbb, 0xbf}, []byte("<root>\n<windowMode>windowed</windowMode>\n<unknown>x&y</unknown>\n<soundVolumeLevel>42</soundVolumeLevel></root>")...)
	if !bytes.Equal(got, want) {
		t.Fatalf("patched XML differs:\n got %q\nwant %q", got, want)
	}
}

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
