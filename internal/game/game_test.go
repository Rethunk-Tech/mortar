package game

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/opener"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func testStore(t *testing.T) *settings.Store {
	t.Helper()
	testfs.DataHome(t)
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
	if err != nil || len(list) != 3 {
		t.Fatalf("list = %v, %v", list, err)
	}
	if g := list[0]; g.ID != "stardew" || !g.Installed || g.ArtURL != "/steam-art/413150" || !g.Available || g.Store != StoreSteam {
		t.Fatalf("stardew = %+v", g)
	}
	if g := list[1]; g.Installed || g.ArtURL != "" || !g.Available {
		t.Fatalf("lethal = %+v", g)
	}

	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1966720" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("hero"))
	}))
	defer cdn.Close()
	was := heroCDN
	heroCDN = cdn.URL + "/%s"
	defer func() { heroCDN = was }()

	h := ArtMiddleware(h0)(http.NotFoundHandler())
	for path, want := range map[string]int{
		"/steam-art/413150":  http.StatusOK,
		"/steam-art/1966720": http.StatusOK,
		"/steam-art/999":     http.StatusNotFound,
		"/steam-art/../etc":  http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("%s = %d, want %d", path, rec.Code, want)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/steam-art/1966720", nil))
	if body := rec.Body.String(); body != "hero" {
		t.Errorf("uncached art served %q, want Steam's public copy", body)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/steam-art/413150", nil))
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

func TestGOGAndPreferredStore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("GOG offline path is a Linux home tree")
	}
	h := t.TempDir()
	gogDir := filepath.Join(h, "GOG Games", "Stardew Valley", "game")
	if err := os.MkdirAll(gogDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gogDir, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	store := testStore(t)
	list, err := NewService(h, store).List()
	if err != nil || list[0].InstallDir != gogDir || list[0].Store != StoreGOG {
		t.Fatalf("gog only = %+v, %v", list[0], err)
	}

	h2 := home(t)
	steamDir := filepath.Join(h2, ".local", "share", "Steam", "steamapps", "common", "Stardew Valley")
	gog2 := filepath.Join(h2, "GOG Games", "Stardew Valley", "game")
	if err := os.MkdirAll(gog2, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gog2, "Stardew Valley.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	store2 := testStore(t)
	list, err = NewService(h2, store2).List()
	if err != nil || list[0].InstallDir != steamDir || list[0].Store != StoreSteam || len(list[0].Installs) < 2 {
		t.Fatalf("both default steam = %+v, %v", list[0], err)
	}
	if _, err := store2.Update(func(s *settings.Settings) { s.GameStores["stardew"] = StoreGOG }); err != nil {
		t.Fatal(err)
	}
	list, err = NewService(h2, store2).List()
	if err != nil || list[0].InstallDir != gog2 || list[0].Store != StoreGOG {
		t.Fatalf("preferred gog = %+v, %v", list[0], err)
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

func TestRequire(t *testing.T) {
	g, err := Require("stardew")
	if err != nil || g == nil || g.ID() != "stardew" {
		t.Fatalf("stardew = %v, %v", g, err)
	}
	_, err = Require("nope")
	if usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("unknown = %v", err)
	}
}

func TestByR2modmanFolderNamesTheCatalogGame(t *testing.T) {
	if id, ok := ByR2modmanFolder("LethalCompany"); !ok || id != "lethal-company" {
		t.Fatalf("got %q %v", id, ok)
	}
	if _, ok := ByR2modmanFolder("Nope"); ok {
		t.Fatal("unknown folder matched")
	}
}

func TestLoaderRefsSayWhichTabsApply(t *testing.T) {
	st := testStore(t)
	games, err := List(t.TempDir(), st.Get())
	if err != nil {
		t.Fatal(err)
	}
	caps := map[string]LoaderRef{}
	for _, g := range games {
		caps[g.ID] = g.Loaders[0]
	}
	if s := caps["stardew"]; !s.Order || !s.Console || !s.Commands || !s.Startup || !s.Assets || !s.Frameworks || !s.Overlay || s.IntroSkip {
		t.Fatalf("SMAPI = %+v", s)
	}
	if b := caps["lethal-company"]; !b.Order || !b.Console || b.Commands || !b.Startup || b.Assets || b.Frameworks || b.Overlay || !b.IntroSkip {
		t.Fatalf("BepInEx = %+v", b)
	}
	if _, err := opener.Web(caps["lethal-company"].Paste); err != nil || caps["stardew"].Paste != "" {
		t.Fatalf("paste sites: BepInEx %q (%v), SMAPI %q", caps["lethal-company"].Paste, err, caps["stardew"].Paste)
	}
}

type anyFolder struct{ Game }

func (anyFolder) ValidInstall(string) error { return nil }

func TestPickCleansTheOverride(t *testing.T) {
	dir, store := pick([]Install{{Dir: filepath.Join("g", "Stardew Valley"), Store: "steam"}}, filepath.Join("g", "Stardew Valley")+string(filepath.Separator), "", anyFolder{})
	if dir != filepath.Join("g", "Stardew Valley") || store != "steam" {
		t.Fatalf("pick = %q, %q", dir, store)
	}
}
