package queue

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

func TestPackageInstallsWithItsDependencies(t *testing.T) {
	f := newFixture(t)
	c, _ := f.s.d.Client()
	f.s.d.Closure = func(_ context.Context, game string, roots []thunderstore.Ref, _ bool) ([]thunderstore.Resolved, error) {
		if game != "riskofrain2" || len(roots) != 2 || roots[0].Name != "Mod" || roots[1].Name != "Other" {
			t.Errorf("closure asked %q %+v", game, roots)
		}
		url := c.BaseURL + "/cdn/file.zip"
		return []thunderstore.Resolved{
			{Namespace: "BepInEx", Name: "BepInExPack", Version: "5.4.2100", URL: url},
			{Namespace: "Me", Name: "Held", Version: "2.0.0", URL: url},
			{Namespace: "Me", Name: "Mod", Version: "1.0.0", URL: url, Icon: "https://ccdn.thunderstore.io/live/repository/icons/Me-Mod-1.0.0.png", Category: "Tools"},
			{Namespace: "Me", Name: "Other", Version: "1.0.0", URL: url},
		}, nil
	}
	f.s.d.Held = func(_, _, pkg string) string {
		if pkg == "Me-Held" {
			return "2.1.0"
		}
		return ""
	}
	f.s.d.InstallPackage = f.s.d.Install
	f.start()
	if _, err := f.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "Me-Mod"}, {Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "Me-Other"}}); err != nil {
		t.Fatal(err)
	}
	st := f.wait("both done", func(st State) bool {
		return len(st.Items) == 2 && st.Items[0].State == StateDone && st.Items[1].State == StateDone
	})
	if st.Items[0].Package != "Me-Mod" || st.Items[1].Package != "Me-Other" || st.Items[0].Kind != KindInstall {
		t.Errorf("a held dependency, the loader pack or a root was mishandled: %+v", st.Items)
	}
	// Both roots install on concurrent workers, so their order is not fixed.
	var mod *profile.Source
	for i := range f.installs {
		if f.installs[i].Name == "Me-Mod" {
			mod = &f.installs[i]
		}
	}
	if len(f.installs) != 2 || mod == nil || mod.Kind != profile.KindThunderstore || mod.Picture != "https://ccdn.thunderstore.io/live/repository/icons/Me-Mod-1.0.0.png" || mod.Category != "Tools" {
		t.Errorf("installed %+v", f.installs)
	}
	if _, err := f.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "not a package"}}); err == nil {
		t.Error("a malformed package was queued")
	}
}

// A listed package another listed package depends on, directly or through a dependency in between, is implicit.
func TestInferPackageReasons(t *testing.T) {
	list := []thunderstore.Resolved{
		{Namespace: "Lib", Name: "Core", Version: "1.0.0"},
		{Namespace: "Lib", Name: "Mid", Version: "1.0.0", Dependencies: []string{"Lib-Core-1.0.0"}},
		{Namespace: "Me", Name: "Mod", Version: "1.0.0", Dependencies: []string{"Lib-Mid-1.0.0"}},
	}
	roots := map[string]Request{"lib-core": {}, "me-mod": {}}
	if got := inferPackageReasons(list, roots); !got["lib-core"] || got["me-mod"] {
		t.Fatalf("inferred = %v", got)
	}
}

// An update to a version that adds a dependency installs that dependency as an implicit entry beside the update.
func TestPackageUpdateInstallsANewDependency(t *testing.T) {
	f := newFixture(t)
	c, _ := f.s.d.Client()
	f.s.d.Closure = func(_ context.Context, _ string, roots []thunderstore.Ref, _ bool) ([]thunderstore.Resolved, error) {
		if len(roots) != 1 || roots[0].Version != "2.0.0" {
			t.Errorf("closure asked %+v", roots)
		}
		url := c.BaseURL + "/cdn/file.zip"
		return []thunderstore.Resolved{
			{Namespace: "Lib", Name: "Added", Version: "1.0.0", URL: url},
			{Namespace: "Me", Name: "Mod", Version: "2.0.0", URL: url, Dependencies: []string{"Lib-Added-1.0.0"}},
		}, nil
	}
	f.s.d.Held = func(_, _, pkg string) string {
		if pkg == "Me-Mod" {
			return "1.0.0"
		}
		return ""
	}
	f.s.d.InstallPackage = f.s.d.Install
	f.start()
	if _, err := f.s.Add(t.Context(), []Request{{Kind: KindUpdate, Game: "riskofrain2", Profile: "p1", Package: "Me-Mod", Version: "2.0.0", CurrentKey: "thunderstore-me-mod-1.0.0"}}); err != nil {
		t.Fatal(err)
	}
	st := f.wait("both done", func(st State) bool {
		return len(st.Items) == 2 && st.Items[0].State == StateDone && st.Items[1].State == StateDone
	})
	got := map[string]bool{}
	for _, it := range st.Items {
		got[it.Package+" "+it.Version] = it.Implicit
	}
	if len(got) != 2 || !got["Lib-Added 1.0.0"] || got["Me-Mod 2.0.0"] {
		t.Errorf("items %v", got)
	}
}

func TestPackageDownloadFallsBackAcrossCDNs(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL
	dead.Close()
	stalled := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer stalled.Close()
	var served atomic.Int32
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		served.Add(1)
		fmt.Fprint(w, payload)
	}))
	defer good.Close()

	f := newFixture(t)
	f.s.d.CDNs = []string{deadURL, stalled.URL, good.URL}
	f.s.cdnStall = 150 * time.Millisecond
	f.s.d.Closure = func(_ context.Context, _ string, _ []thunderstore.Ref, _ bool) ([]thunderstore.Resolved, error) {
		return []thunderstore.Resolved{{Namespace: "Me", Name: "Mod", Version: "1.0.0", URL: deadURL + "/p/Me-Mod.zip"}}, nil
	}
	f.s.d.InstallPackage = f.s.d.Install
	f.start()
	if _, err := f.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "Me-Mod"}}); err != nil {
		t.Fatal(err)
	}
	f.wait("done", func(st State) bool { return len(st.Items) == 1 && st.Items[0].State == StateDone })
	if served.Load() != 1 {
		t.Fatalf("the working CDN served %d downloads", served.Load())
	}
	f.s.mu.Lock()
	remembered := f.s.cdnGood
	f.s.mu.Unlock()
	if remembered != good.URL {
		t.Fatalf("remembered %q, want %q", remembered, good.URL)
	}
}

func TestPinnedCDNIsTheOnlyOneTried(t *testing.T) {
	var other atomic.Int32
	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { other.Add(1) }))
	defer a.Close()
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, payload) }))
	defer b.Close()
	f := newFixture(t)
	f.s.d.CDNs = []string{a.URL, b.URL}
	f.s.d.ThunderstoreHost = func() string { return b.URL }
	f.s.d.Closure = func(_ context.Context, _ string, _ []thunderstore.Ref, _ bool) ([]thunderstore.Resolved, error) {
		return []thunderstore.Resolved{{Namespace: "Me", Name: "Mod", Version: "1.0.0", URL: a.URL + "/p/Me-Mod.zip"}}, nil
	}
	f.s.d.InstallPackage = f.s.d.Install
	f.start()
	if _, err := f.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "Me-Mod"}}); err != nil {
		t.Fatal(err)
	}
	f.wait("done", func(st State) bool { return len(st.Items) == 1 && st.Items[0].State == StateDone })
	if other.Load() != 0 {
		t.Fatalf("the unpinned CDN was asked %d times", other.Load())
	}
}

func TestPackageStalledCDNHandsItsPartialToTheNext(t *testing.T) {
	const cut = 5
	stalled := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		fmt.Fprint(w, payload[:cut])
		_ = http.NewResponseController(w).Flush()
		<-r.Context().Done()
	}))
	defer stalled.Close()
	var rng atomic.Value
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rng.Store(r.Header.Get("Range"))
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", cut, len(payload)-1, len(payload)))
		w.WriteHeader(http.StatusPartialContent)
		fmt.Fprint(w, payload[cut:])
	}))
	defer good.Close()

	f := newFixture(t)
	f.s.d.CDNs = []string{stalled.URL, good.URL}
	f.s.cdnStall = 150 * time.Millisecond
	f.s.d.Closure = func(_ context.Context, _ string, _ []thunderstore.Ref, _ bool) ([]thunderstore.Resolved, error) {
		return []thunderstore.Resolved{{Namespace: "Me", Name: "Mod", Version: "1.0.0", URL: stalled.URL + "/p/Me-Mod.zip"}}, nil
	}
	f.s.d.InstallPackage = f.s.d.Install
	f.start()
	if _, err := f.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "Me-Mod"}}); err != nil {
		t.Fatal(err)
	}
	f.wait("done", func(st State) bool { return len(st.Items) == 1 && st.Items[0].State == StateDone })
	if got, _ := rng.Load().(string); got != fmt.Sprintf("bytes=%d-", cut) {
		t.Fatalf("the next CDN was asked for %q", got)
	}
}
