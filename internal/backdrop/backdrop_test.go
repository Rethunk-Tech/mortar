package backdrop

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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
	h := Middleware(cur, system, desktop, "")(next)
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
	h := Middleware(func() settings.Settings { return settings.Defaults() }, "", none, "")(next)
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
	h := Middleware(stale, sys, func() string { return wall }, "")(next)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, Path+"?mode=image&v=", nil))
	if rec.Body.String() != "system" {
		t.Fatalf("mode=image served %q, want the default image", rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-cache" || rec.Header().Get("ETag") == "" {
		t.Fatalf("headers %v allow a stale revalidation", rec.Header())
	}
	// An older desktop file than the cached image must not win an If-Modified-Since check: the ETag differs.
	stale304 := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, Path+"?mode=desktop", nil)
	req.Header.Set("If-None-Match", rec.Header().Get("ETag"))
	req.Header.Set("If-Modified-Since", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
	h.ServeHTTP(stale304, req)
	if stale304.Code != http.StatusOK || stale304.Body.String() != "desk" {
		t.Fatalf("another file answered %d %q to the first one's validators", stale304.Code, stale304.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, Path+"?mode=bogus", nil))
	if rec.Body.String() != "desk" {
		t.Fatalf("an unknown mode must fall back to settings, got %q", rec.Body.String())
	}
}

func TestJXLNeedsAClientThatAcceptsIt(t *testing.T) {
	if !negotiates {
		t.Skip("the desktop window always gets the JPEG XL file")
	}
	dir := t.TempDir()
	sys := write(t, dir, "sys.jxl", "jxl")
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	h := Middleware(func() settings.Settings { return settings.Settings{Background: settings.BackgroundImage} }, sys, none, "")(next)
	for accept, want := range map[string]string{"image/jxl,image/*": "image/jxl", "image/webp,image/*": "image/jpeg"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, Path, nil)
		req.Header.Set("Accept", accept)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Header().Get("Content-Type"); got != want || rec.Header().Get("Vary") != "Accept" {
			t.Errorf("Accept %q: type %q, want %q", accept, got, want)
		}
	}
}

func TestWidthParameterShrinksTheBackdrop(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	h := Middleware(func() settings.Settings { return settings.Defaults() }, "", none, "")(next)
	width := func(query string) int {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, Path+query, nil))
		cfg, _, err := image.DecodeConfig(rec.Body)
		if err != nil {
			t.Fatal(err)
		}
		return cfg.Width
	}
	full := width("")
	if full <= 1280 {
		t.Skipf("bundled wallpaper is only %dpx wide", full)
	}
	if got := width("?w=1280"); got != 1280 {
		t.Errorf("w=1280 served %dpx", got)
	}
	if got := width("?w=100"); got != full {
		t.Errorf("a width below the minimum must serve the original, got %dpx", got)
	}
}

func TestAnimatedBackdropsKeepTheirFrames(t *testing.T) {
	dir := t.TempDir()
	// Two frames wider than the width asked for, which a still image would be shrunk to.
	frame := image.NewPaletted(image.Rect(0, 0, 1400, 4), color.Palette{color.Black, color.White})
	var anim bytes.Buffer
	if err := gif.EncodeAll(&anim, &gif.GIF{Image: []*image.Paletted{frame, frame}, Delay: []int{10, 10}}); err != nil {
		t.Fatal(err)
	}
	vp8x := "RIFF\x00\x00\x00\x00WEBPVP8X\x0a\x00\x00\x00\x02\x00\x00\x00anim"
	for name, body := range map[string]string{
		"a.webp": vp8x,
		"a.png":  "\x89PNG\r\n\x1a\nacTL....IDAT",
		"a.gif":  anim.String(),
	} {
		wall := write(t, dir, name, body)
		cur := func() settings.Settings {
			return settings.Settings{Background: settings.BackgroundImage, BackgroundImage: wall}
		}
		h := Middleware(cur, "", none, "")(http.NotFoundHandler())
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, Path+"?w=1280", nil))
		if rec.Body.String() != body {
			t.Errorf("%s: an animation was re-encoded instead of served whole", name)
		}
	}
}

// The system default's JPEG XL cannot be shrunk, so a page asking for its width gets the bundled JPEG of it.
func TestASizedDefaultBackdropIsTheBundledJPEG(t *testing.T) {
	dir := t.TempDir()
	sys := write(t, dir, "sys.jxl", "jxl")
	h := Middleware(func() settings.Settings { return settings.Settings{Background: settings.BackgroundImage} }, sys, none, "")(http.NotFoundHandler())
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, Path+"?w=1280", nil)
	req.Header.Set("Accept", "image/jxl,image/*")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Errorf("type %q, want the bundled JPEG", got)
	}
}

// A start after the first serves the kept shrunk copy, dated by its source so the WebView can revalidate it.
func TestShrunkBackdropIsKeptOnDiskAndDated(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, "cache")
	img := image.NewNRGBA(image.Rect(0, 0, 2000, 40))
	var src bytes.Buffer
	if err := jpeg.Encode(&src, img, nil); err != nil {
		t.Fatal(err)
	}
	wall := write(t, dir, "wall.jpg", src.String())
	cur := func() settings.Settings {
		return settings.Settings{Background: settings.BackgroundImage, BackgroundImage: wall}
	}
	h := Middleware(cur, "", none, cache)(http.NotFoundHandler())
	fetch := func(h http.Handler, header http.Header) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, Path+"?w=1280", nil)
		req.Header = header
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	first := fetch(h, http.Header{})
	if cfg, _, err := image.DecodeConfig(first.Body); err != nil || cfg.Width != 1280 {
		t.Fatalf("shrunk to %v, %v", cfg.Width, err)
	}
	if first.Header().Get("Last-Modified") == "" {
		t.Fatal("no Last-Modified")
	}
	kept, err := os.ReadDir(cache)
	if err != nil || len(kept) != 1 {
		t.Fatalf("kept copies = %v, %v", kept, err)
	}
	// Corrupt the source in place, keeping size and date: only the kept copy can still answer with 1280 pixels.
	info, err := os.Stat(wall)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wall, make([]byte, info.Size()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(wall, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	second := fetch(h, http.Header{})
	if cfg, _, err := image.DecodeConfig(second.Body); err != nil || cfg.Width != 1280 {
		t.Fatalf("second start served %v, %v", cfg.Width, err)
	}
	revalidate := fetch(h, http.Header{"If-None-Match": {first.Header().Get("ETag")}})
	if revalidate.Code != http.StatusNotModified {
		t.Fatalf("revalidation = %d, want 304", revalidate.Code)
	}
}
