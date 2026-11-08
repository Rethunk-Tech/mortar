//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// A loader log is untrusted: the game writes it, and a bad byte must not reach the analysis as a broken rune, as the
// game's own log and crash report are already cleaned.
func TestLoaderLogIsValidUTF8(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	_, ps := testenv.Stores(t)
	const lc = "lethal-company"
	p := testenv.Profile(t, ps, lc, "A")
	dir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "BepInEx"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "BepInEx", "LogOutput.log"), []byte("[Info:x] caf\xff plugin\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), nil, ps)
	l, ok := svc.loaderOf(lc, p.ID)
	logs, withLogs := l.(loader.WithLogs)
	if !ok || !withLogs {
		t.Fatalf("loader = %T, %v", l, ok)
	}
	text, read := svc.loaderLog(logs, lc, p.ID)
	if !read || !utf8.ValidString(text) {
		t.Fatalf("loader log = %q, valid %v, read %v", text, utf8.ValidString(text), read)
	}
}
