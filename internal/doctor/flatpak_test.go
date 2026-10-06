package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

func TestFlatpakLibraryChecksNameLibrariesTheSandboxCannotSee(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".local", "share", "Steam")
	gone := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.MkdirAll(filepath.Join(root, "steamapps"), 0o700); err != nil {
		t.Fatal(err)
	}
	vdf := fmt.Sprintf("\"libraryfolders\"\n{\n\t\"0\"\n\t{\n\t\t\"path\"\t\t%q\n\t}\n\t\"1\"\n\t{\n\t\t\"path\"\t\t%q\n\t}\n}\n", root, gone)
	if err := os.WriteFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"), []byte(vdf), 0o600); err != nil {
		t.Fatal(err)
	}
	flatpakID := ""
	prevHome, prevEnv := userHome, sandbox.Getenv
	userHome = func() (string, error) { return home, nil }
	sandbox.Getenv = func(k string) string {
		if k == "FLATPAK_ID" {
			return flatpakID
		}
		return ""
	}
	t.Cleanup(func() { userHome, sandbox.Getenv = prevHome, prevEnv })
	if got := flatpakLibraryChecks(); len(got) != 0 {
		t.Fatalf("outside a Flatpak: %+v", got)
	}
	flatpakID = sandbox.AppID
	got := flatpakLibraryChecks()
	if len(got) != 1 || got[0].Status != Warn || !strings.Contains(got[0].Fix, "--filesystem="+gone+" ") {
		t.Fatalf("inside a Flatpak: %+v", got)
	}
}
