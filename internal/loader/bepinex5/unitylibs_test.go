package bepinex5

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

// An IL2CPP game's Unity base libraries are fetched once, kept, and placed in each profile from the kept copy.
func TestPlaceUnityLibs(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	game := t.TempDir()
	data := filepath.Join(game, "Cat Mail Co_Data")
	if err := os.MkdirAll(data, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "globalgamemanagers"), []byte("\x00\x00junk 2022.3.21f1 more"), 0o600); err != nil {
		t.Fatal(err)
	}
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked.Add(1)
		if r.URL.Path != "/2022.3.21.zip" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("PK\x03\x04zip"))
	}))
	defer srv.Close()
	unityLibsURL = srv.URL + "/"
	for _, profile := range []string{t.TempDir(), t.TempDir()} {
		if err := placeUnityLibs(t.Context(), game, profile); err != nil {
			t.Fatal(err)
		}
		if b, err := safefs.ReadFile(filepath.Join(profile, "BepInEx", "unity-libs", "2022.3.21.zip")); err != nil || string(b) != "PK\x03\x04zip" {
			t.Fatalf("placed = %q, %v", b, err)
		}
	}
	if asked.Load() != 1 {
		t.Fatalf("asked %d times, want once", asked.Load())
	}
	if unityVersion(t.TempDir()) != "" {
		t.Fatal("a folder with no Unity data has a version")
	}
}
