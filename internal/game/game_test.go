package game

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/settings"
)

func testStore(t *testing.T) *settings.Store {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	return store
}

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
	svc := NewService(h0, testStore(t))
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
	svc := NewService(t.TempDir(), testStore(t))
	list, err := svc.List()
	if err != nil || list[0].Installed || svc.SteamStatus() != "not-found" {
		t.Fatalf("list = %v, %v, %s", list, err, svc.SteamStatus())
	}
}

func stardewFolder(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDiscoveryPrecedence(t *testing.T) {
	h := home(t)
	steamDir := filepath.Join(h, ".local", "share", "Steam", "steamapps", "common", "Stardew Valley")
	override := stardewFolder(t)

	for name, tc := range map[string]struct{ override, want string }{
		"steam only":       {"", steamDir},
		"valid override":   {override, override},
		"invalid override": {t.TempDir(), steamDir},
	} {
		store := testStore(t)
		if tc.override != "" {
			if _, err := store.Update(func(s *settings.Settings) { s.GameFolders["stardew"] = tc.override }); err != nil {
				t.Fatal(err)
			}
		}
		list, err := NewService(h, store).List()
		if err != nil || list[0].InstallDir != tc.want {
			t.Errorf("%s: dir = %q (%v), want %q", name, list[0].InstallDir, err, tc.want)
		}
	}
}

func TestOverrideWithoutSteam(t *testing.T) {
	override := stardewFolder(t)
	store := testStore(t)
	if _, err := store.Update(func(s *settings.Settings) { s.GameFolders["stardew"] = override }); err != nil {
		t.Fatal(err)
	}
	list, err := NewService(t.TempDir(), store).List()
	if err != nil || !list[0].Installed || list[0].InstallDir != override {
		t.Fatalf("list = %+v, %v", list[0], err)
	}
}

func TestValidateFolder(t *testing.T) {
	for _, tc := range []struct {
		id, dir string
		ok      bool
	}{
		{"stardew", stardewFolder(t), true},
		{"stardew", "", true},
		{"stardew", t.TempDir(), false},
		{"stardew", "relative", false},
		{"lethal", stardewFolder(t), false},
	} {
		if err := ValidateFolder(tc.id, tc.dir); (err == nil) != tc.ok {
			t.Errorf("ValidateFolder(%s, %q) = %v", tc.id, tc.dir, err)
		}
	}
}
