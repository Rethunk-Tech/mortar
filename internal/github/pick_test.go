package github

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

func names(assets []Asset) []string {
	out := make([]string, len(assets))
	for i, a := range assets {
		out[i] = a.Name
	}
	return out
}

func TestInstallable(t *testing.T) {
	t.Parallel()
	all := []Asset{
		{Name: "Mod-1.2.zip"},
		{Name: "Mod-1.2-source.zip"},
		{Name: "src.zip"},
		{Name: "checksums.zip"},
		{Name: "Mod-1.2-Linux.zip"},
		{Name: "Mod-1.2-win64.zip"},
		{Name: "Mod-1.2-macOS.zip"},
	}
	if got := names(Installable(all, "linux")); !slices.Equal(got, []string{"Mod-1.2.zip", "Mod-1.2-Linux.zip"}) {
		t.Errorf("linux kept %v", got)
	}
	if got := names(Installable(all, "windows")); !slices.Equal(got, []string{"Mod-1.2.zip", "Mod-1.2-win64.zip"}) {
		t.Errorf("windows kept %v", got)
	}
	// Only another platform's build ships, so it stays: the author meant it for everyone.
	if got := names(Installable([]Asset{{Name: "Mod-Windows.zip"}, {Name: "Mod-src.zip"}}, "linux")); !slices.Equal(got, []string{"Mod-Windows.zip"}) {
		t.Errorf("kept %v", got)
	}
}

func TestShape(t *testing.T) {
	t.Parallel()
	if a, b := Shape("MyMod-1.2.0-linux.zip"), Shape("MyMod v1.3 linux.ZIP"); a != b || a != "mymod linux" {
		t.Errorf("shapes %q %q", a, b)
	}
}

func TestZipNamesReadsOnlyTheDirectory(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range []string{"Mod/manifest.json", "Mod/Mod.dll"} {
		w, _ := zw.CreateHeader(&zip.FileHeader{Name: n, Method: zip.Store})
		_, _ = w.Write(bytes.Repeat([]byte("x"), 200<<10))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	served := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") == "" {
			t.Error("asked for the whole file")
		}
		http.ServeContent(counter{w, &served}, r, "a.zip", time.Time{}, bytes.NewReader(buf.Bytes()))
	}))
	t.Cleanup(srv.Close)
	got, err := ZipNames(t.Context(), srv.Client(), Asset{Name: "a.zip", URL: srv.URL, Size: int64(buf.Len())})
	if err != nil || !slices.Equal(got, []string{"Mod/manifest.json", "Mod/Mod.dll"}) {
		t.Fatalf("names %v %v", got, err)
	}
	if served > peekChunk {
		t.Errorf("read %d bytes of a %d byte zip", served, buf.Len())
	}
}

type counter struct {
	http.ResponseWriter
	n *int
}

func (c counter) Write(b []byte) (int, error) {
	*c.n += len(b)
	return c.ResponseWriter.Write(b)
}
