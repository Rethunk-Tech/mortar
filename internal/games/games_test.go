package games

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func home(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	root := filepath.Join(h, ".local", "share", "Steam")
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, "steamapps", "common", "Stardew Valley"), 0o700))
	must(os.MkdirAll(filepath.Join(root, "appcache", "librarycache", "413150"), 0o700))
	must(os.WriteFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"),
		[]byte("\"libraryfolders\"\n{\n\"0\"\n{\n\"path\" \""+root+"\"\n}\n}\n"), 0o600))
	must(os.WriteFile(filepath.Join(root, "steamapps", "appmanifest_413150.acf"),
		[]byte("\"AppState\"\n{\n\"installdir\" \"Stardew Valley\"\n}\n"), 0o600))
	must(os.WriteFile(filepath.Join(root, "appcache", "librarycache", "413150", "library_hero.jpg"), []byte("jpg"), 0o600))
	return h
}

func TestListAndArt(t *testing.T) {
	h0 := home(t)
	svc := NewService(h0)
	list, err := svc.List()
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %v, %v", list, err)
	}
	if g := list[0]; g.ID != "stardew" || !g.Installed || g.ArtURL != "/steam-art/413150" || !g.Available {
		t.Fatalf("stardew = %+v", g)
	}
	if g := list[1]; g.Installed || g.ArtURL != "" || g.Available {
		t.Fatalf("lethal = %+v", g)
	}

	h := ArtMiddleware(h0)(http.NotFoundHandler())
	for path, want := range map[string]int{
		"/steam-art/413150":  http.StatusOK,
		"/steam-art/1966720": http.StatusNotFound,
		"/steam-art/999":     http.StatusNotFound,
		"/steam-art/../etc":  http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("%s = %d, want %d", path, rec.Code, want)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/steam-art/413150", nil))
	if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("content type = %q", ct)
	}
}

func TestNoSteam(t *testing.T) {
	svc := NewService(t.TempDir())
	list, err := svc.List()
	if err != nil || list[0].Installed || svc.SteamStatus() != "not-found" {
		t.Fatalf("list = %v, %v, %s", list, err, svc.SteamStatus())
	}
}
