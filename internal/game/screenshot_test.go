package game

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// A screenshot request names a bare picture in one of the game's screenshot folders; a path that climbs out of the
// folder is not served, whatever its extension.
func TestScreenshotMiddlewareServesOnlyTheFolder(t *testing.T) {
	home := t.TempDir()
	dir, err := PathFor(home, settings.Settings{}, "stardew", "", PathScreenshots)
	if err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{filepath.Join(dir, "farm.png"): "farm", filepath.Join(filepath.Dir(dir), "secret.png"): "secret"} {
		testfs.WriteAt(t, path, []byte(body))
	}
	serve := ScreenshotMiddleware(home, func() settings.Settings { return settings.Settings{} })(http.NotFoundHandler())
	for url, want := range map[string]int{
		"/screenshot/stardew/game/farm.png":            http.StatusOK,
		"/screenshot/stardew/game/..%2Fsecret.png":     http.StatusNotFound,
		"/screenshot/stardew/game/sub%2F..%2Ffarm.png": http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		serve.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, url, nil))
		if rec.Code != want {
			t.Errorf("%s = %d (%q), want %d", url, rec.Code, rec.Body.String(), want)
		}
	}
}
