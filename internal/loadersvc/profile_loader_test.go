package loadersvc

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// A loader that lives in each profile is downloaded once into the store, laid into every profile, and a profile
// created later gets it without another download.
func TestEnsureInstallsAProfileLoaderIntoEveryProfile(t *testing.T) {
	t.Setenv("MORTAR_ENABLE_GAMES", "lethal-company")
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("LOCALAPPDATA", data)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"manifest.json":                                  `{"name":"BepInExPack","version_number":"5.4.2304"}`,
		"BepInExPack/winhttp.dll":                        "proxy",
		"BepInExPack/BepInEx/core/BepInEx.Preloader.dll": "pre",
	} {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	var downloads atomic.Int32
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/c/lethal-company/api/v1/package-listing-index/":
			_, _ = fmt.Fprintf(w, `[%q]`, srv.URL+"/chunk")
		case "/chunk":
			_, _ = w.Write([]byte(`[{"name":"BepInExPack","owner":"BepInEx","versions":[{"version_number":"5.4.2304"}]}]`))
		case "/package/download/BepInEx/BepInExPack/5.4.2304/":
			downloads.Add(1)
			_, _ = w.Write(buf.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	loader.Register(bepinex5.Loader{Index: &thunderstore.Driver{HTTP: srv.Client(), URL: srv.URL, CacheDir: t.TempDir()}, HTTP: srv.Client()})
	t.Cleanup(func() { loader.Register(bepinex5.Loader{}) })

	folder := t.TempDir()
	put(t, filepath.Join(folder, "Lethal Company.exe"), "x")
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders["lethal-company"] = folder }); err != nil {
		t.Fatal(err)
	}
	items, profiles := testenv.Stores(t)
	svc := NewService(t.TempDir(), set, items, profiles)
	first := testenv.Profile(t, profiles, "lethal-company", "First")

	st, err := svc.Ensure(t.Context(), "lethal-company", "", false)
	if err != nil || !st.Installed || st.Version != "5.4.2304" || !st.PerProfile {
		t.Fatalf("ensure = %+v, %v", st, err)
	}
	second := testenv.Profile(t, profiles, "lethal-company", "Second")
	if st, _ := svc.LocalStatus("lethal-company", ""); st.Installed {
		t.Fatal("a profile without the loader still counts as installed")
	}
	if _, err := svc.Ensure(t.Context(), "lethal-company", "", false); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{first.ID, second.ID} {
		dir, _ := profiles.ProfileDir("lethal-company", p)
		if st, _ := (bepinex5.Loader{}).Status(loader.Target{ProfileDir: dir}); !st.Installed || st.Version != "5.4.2304" {
			t.Fatalf("profile %s status %+v", p, st)
		}
	}
	if got := downloads.Load(); got != 1 {
		t.Fatalf("downloads = %d, want 1", got)
	}
}
