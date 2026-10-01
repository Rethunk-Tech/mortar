package gamesettings

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
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

func TestPatchFileMissingLeavesFileAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "startup_preferences")
	mode := "windowed"
	if got, err := PatchFile(path, Settings{WindowMode: &mode}); err != nil || got != nil {
		t.Fatalf("missing file: got %q, err %v", got, err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing file was created: %v", err)
	}
}
