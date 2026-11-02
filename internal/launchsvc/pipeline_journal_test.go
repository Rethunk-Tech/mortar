package launchsvc

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/game"
)

func TestJournalDirOfAFolderPinIsOneSegment(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	probe, err := journalDir("x")
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Dir(probe)
	for _, pin := range []string{game.FolderPin("/games/Stardew Valley") + "-saves", `dir:C:\Games\Stardew`} {
		dir, err := journalDir(pin)
		if err != nil {
			t.Fatal(err)
		}
		seg, err := filepath.Rel(base, dir)
		if err != nil || !filepath.IsLocal(seg) || strings.ContainsAny(seg, `/\:`) {
			t.Errorf("%q -> %q is not one segment", pin, seg)
		}
	}
}
