package backdrop

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func none() string { return "" }

func getMode(t *testing.T, mode, user, system string, desktop func() string) *httptest.ResponseRecorder {
	t.Helper()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	cur := func() settings.Settings { return settings.Settings{Background: mode, BackgroundImage: user} }
	h := Middleware(cur, system, desktop)(next)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, Path, nil))
	return rec
}

func get(t *testing.T, user, system string) *httptest.ResponseRecorder {
	t.Helper()
	return getMode(t, settings.BackgroundImage, user, system, none)
}

func TestDesktopMode(t *testing.T) {
	dir := t.TempDir()
	wall := write(t, dir, "wall.png", "desk")
	mine := write(t, dir, "mine.png", "user")
	sys := write(t, dir, "sys.webp", "system")
	missing := filepath.Join(dir, "gone.png")
	at := func(p string) func() string { return func() string { return p } }

	cases := []struct {
		name, desktop, body string
	}{
		{"desktop wallpaper wins", wall, "desk"},
		{"image resolution when the desktop is unreadable", "", "user"},
		{"image resolution when the desktop file is gone", missing, "user"},
	}
	for _, c := range cases {
		rec := getMode(t, settings.BackgroundDesktop, mine, sys, at(c.desktop))
		if rec.Code != http.StatusOK || rec.Body.String() != c.body {
			t.Errorf("%s: %d %q", c.name, rec.Code, rec.Body.String())
		}
	}
	if rec := getMode(t, settings.BackgroundImage, mine, sys, at(wall)); rec.Body.String() != "user" {
		t.Errorf("image mode consulted the desktop: %q", rec.Body.String())
	}
}

func TestSolidHasNoBackdrop(t *testing.T) {
	if rec := getMode(t, settings.BackgroundSolid, "", "", none); rec.Code != http.StatusNotFound {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestResolutionOrder(t *testing.T) {
	dir := t.TempDir()
	mine := write(t, dir, "mine.png", "user")
	sys := write(t, dir, "sys.webp", "system")
	missing := filepath.Join(dir, "gone.png")

	cases := []struct {
		name, user, system, body, typ string
	}{
		{"user wins", mine, sys, "user", "image/png"},
		{"system when no user image", "", sys, "system", "image/webp"},
		{"system when user file is gone", missing, sys, "system", "image/webp"},
		{"bundled when neither", "", missing, string(bundled), "image/jpeg"},
		{"bundled when user path is not an image", write(t, dir, "notes.txt", "x"), "", string(bundled), "image/jpeg"},
	}
	for _, c := range cases {
		rec := get(t, c.user, c.system)
		if rec.Code != http.StatusOK || rec.Body.String() != c.body || rec.Header().Get("Content-Type") != c.typ {
			t.Errorf("%s: %d len %d %s", c.name, rec.Code, rec.Body.Len(), rec.Header().Get("Content-Type"))
		}
	}
}

func TestBundledIsJPEG(t *testing.T) {
	if !bytes.HasPrefix(bundled, []byte{0xff, 0xd8, 0xff}) {
		t.Fatal("bundled wallpaper is not a JPEG")
	}
}

func TestOtherPathsPassThrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	h := Middleware(func() settings.Settings { return settings.Defaults() }, "", none)(next)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/backdrop/x", nil))
	if rec.Code != http.StatusTeapot {
		t.Fatalf("code %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, Path, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("POST code %d", rec.Code)
	}
}

func TestCheck(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "dir.png"), 0o700); err != nil {
		t.Fatal(err)
	}
	target := write(t, dir, "real.png", "x")
	link := filepath.Join(dir, "link.png")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks unavailable")
	}
	for _, ok := range []string{target, write(t, dir, "a.JPG", "x"), write(t, dir, "b.jpeg", "x"), write(t, dir, "c.webp", "x")} {
		if err := Check(ok); err != nil {
			t.Errorf("Check(%s) = %v", ok, err)
		}
	}
	for _, bad := range []string{
		"", "relative.png", dir + "/../" + filepath.Base(dir) + "/real.png",
		write(t, dir, "notes.txt", "x"), write(t, dir, "noext", "x"), write(t, dir, "x.svg", "x"),
		filepath.Join(dir, "dir.png"), filepath.Join(dir, "missing.png"), link,
	} {
		if err := Check(bad); err == nil {
			t.Errorf("Check(%q) accepted", bad)
		}
	}
	jxl := write(t, dir, "d.jxl", "x")
	if got := Check(jxl) == nil; got != (runtime.GOOS == "linux") {
		t.Errorf("jxl accepted = %v on %s", got, runtime.GOOS)
	}
}

// A mode switch reaches the backdrop before the settings write does; the page's mode wins and nothing is cached.
func TestRequestedModeWinsAndIsUncached(t *testing.T) {
	dir := t.TempDir()
	wall := write(t, dir, "wall.png", "desk")
	sys := write(t, dir, "sys.webp", "system")
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	stale := func() settings.Settings { return settings.Settings{Background: settings.BackgroundDesktop} }
	h := Middleware(stale, sys, func() string { return wall })(next)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, Path+"?mode=image&v=", nil))
	if rec.Body.String() != "system" {
		t.Fatalf("mode=image served %q, want the default image", rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Last-Modified") != "" {
		t.Fatalf("headers %v allow a stale revalidation", rec.Header())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, Path+"?mode=bogus", nil))
	if rec.Body.String() != "desk" {
		t.Fatalf("an unknown mode must fall back to settings, got %q", rec.Body.String())
	}
}
