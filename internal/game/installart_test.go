package game

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestInstallArtServesTheImageFromTheSelectedInstall(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the catalog's Vintage Story folders here are the Linux ones")
	}
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", "")
	dir := filepath.Join(home, ".local", "share", "vintagestory")
	art := filepath.Join(dir, "assets", "game", "textures", "gui", "backgrounds", "mainmenu1.png")
	for path, body := range map[string]string{filepath.Join(dir, "Vintagestory"): "", art: "png"} {
		testfs.WriteAt(t, path, []byte(body))
	}
	h := InstallArtMiddleware(home, settings.Defaults)(http.NotFoundHandler())
	for path, want := range map[string]int{"/install-art/vintage-story": http.StatusOK, "/install-art/stardew": http.StatusNotFound} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		if rec.Code != want || want == http.StatusOK && rec.Body.String() != "png" {
			t.Fatalf("%s = %d %q", path, rec.Code, rec.Body.String())
		}
	}
}
