package profile

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

var pngHeader = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

func nexusEntry(key string, endorsements int, picture string) Entry {
	return Entry{Key: key, Source: Source{Kind: KindNexus, ModID: 1, Picture: picture, EndorsementCount: endorsements}}
}

func TestCoversOrder(t *testing.T) {
	p := Profile{ID: "0123456789abcdef", Entries: []Entry{
		nexusEntry("a", 10, "https://img/a.png"),
		nexusEntry("b", 900, "https://img/b.png"),
		nexusEntry("c", 5000, ""),
		nexusEntry("d", 8000, "http://img/insecure.png"),
		{Key: "e", Source: Source{Kind: KindGitHub, Picture: "https://img/gh.png", EndorsementCount: 9000}},
	}}
	art := "/steam-art/413150"
	if got := Covers("stardew", p); !slices.Equal(got, []string{"https://img/b.png", art}) {
		t.Fatalf("automatic covers = %v", got)
	}
	p.Cover = "cover.png"
	got := Covers("stardew", p)
	if len(got) != 3 || !strings.HasPrefix(got[0], "/profile-cover/stardew/0123456789abcdef?v=") || got[1] != "https://img/b.png" || got[2] != art {
		t.Fatalf("picked covers = %v", got)
	}
	p.Cover, p.Entries = "../../secret.png", nil
	if got := Covers("stardew", p); !slices.Equal(got, []string{art}) {
		t.Fatalf("tampered cover = %v", got)
	}
}

func TestSetCoverValidatesAndCopies(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	write := func(name string, b []byte) string {
		path := filepath.Join(src, name)
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	for name, b := range map[string][]byte{
		"text.png": []byte("not an image"),
		"huge.png": append(slices.Clone(pngHeader), make([]byte, MaxCover)...),
	} {
		if _, err := s.SetCover("stardew", p.ID, write(name, b)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if _, err := s.SetCover("stardew", p.ID, src); err == nil {
		t.Error("a folder was accepted")
	}

	img := write("mine.jpg", pngHeader)
	got, err := s.SetCover("stardew", p.ID, img)
	if err != nil || got.Cover != "cover.png" {
		t.Fatalf("set = %+v, %v", got, err)
	}
	if err := os.Remove(img); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(s.root, "stardew", p.ID)
	if b, err := fsx.ReadFile(filepath.Join(dir, "cover.png")); err != nil || !bytes.Equal(b, pngHeader) {
		t.Fatalf("copied cover = %q, %v", b, err)
	}

	dup, err := s.Duplicate("stardew", p.ID)
	if err != nil || dup.Cover != "cover.png" {
		t.Fatalf("duplicate = %+v, %v", dup, err)
	}
	if _, err := os.Stat(filepath.Join(s.root, "stardew", dup.ID, "cover.png")); err != nil {
		t.Fatalf("duplicate cover: %v", err)
	}

	jpeg := write("next.png", []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00"))
	if got, err := s.SetCover("stardew", p.ID, jpeg); err != nil || got.Cover != "cover.jpg" {
		t.Fatalf("replace = %+v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "cover.png")); !os.IsNotExist(err) {
		t.Fatalf("old cover kept: %v", err)
	}
	if got, err := s.ClearCover("stardew", p.ID); err != nil || got.Cover != "" {
		t.Fatalf("clear = %+v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "cover.jpg")); !os.IsNotExist(err) {
		t.Fatalf("cleared cover kept: %v", err)
	}
}

func TestCoverMiddlewarePathSafety(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	bare, err := s.Create("stardew", "Bare")
	if err != nil {
		t.Fatal(err)
	}
	img := filepath.Join(t.TempDir(), "c.png")
	if err := os.WriteFile(img, pngHeader, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetCover("stardew", p.ID, img); err != nil {
		t.Fatal(err)
	}
	tampered, err := s.Create("stardew", "Tampered")
	if err != nil {
		t.Fatal(err)
	}
	tdir := filepath.Join(s.root, "stardew", tampered.ID)
	tampered.Cover = "../" + p.ID + "/cover.png"
	raw, _ := json.Marshal(tampered)
	if err := os.WriteFile(filepath.Join(tdir, fileName), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	h := CoverMiddleware(func() *Store { return s })(http.NotFoundHandler())
	for path, want := range map[string]int{
		"/profile-cover/stardew/" + p.ID:          http.StatusOK,
		"/profile-cover/stardew/" + bare.ID:       http.StatusNotFound,
		"/profile-cover/stardew/" + tampered.ID:   http.StatusNotFound,
		"/profile-cover/stardew/../" + p.ID:       http.StatusNotFound,
		"/profile-cover/nosuchgame/" + p.ID:       http.StatusNotFound,
		"/profile-cover/stardew/" + p.ID + "/x":   http.StatusNotFound,
		"/profile-cover/stardew/0000000000000000": http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("%s = %d, want %d", path, rec.Code, want)
		}
		if want == http.StatusOK && rec.Header().Get("Content-Type") != "image/png" {
			t.Errorf("%s content type = %q", path, rec.Header().Get("Content-Type"))
		}
	}
	rec := httptest.NewRecorder()
	CoverMiddleware(func() *Store { return nil })(http.NotFoundHandler()).
		ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/profile-cover/stardew/"+p.ID, nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("before the store opens = %d", rec.Code)
	}
}
