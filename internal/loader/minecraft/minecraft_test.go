package minecraft

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestInstallMakesAJavaInstanceAndContributeLaunchesThroughJava(t *testing.T) {
	t.Parallel()
	profile, install := t.TempDir(), t.TempDir()
	testfs.WriteAt(t, filepath.Join(install, "versions", "1.21.1", "1.21.1.json"), []byte(`{}`))
	java := filepath.Join(t.TempDir(), "java")
	testfs.WriteAt(t, java, []byte("#!/bin/sh\n"))
	l := Loader{Kind: Fabric, Java: java}
	ver, err := l.Install(t.Context(), loader.Target{ProfileDir: profile, InstallDir: install}, loader.Package{Version: "0.16.9"}, nil)
	if err != nil || ver != "0.16.9" {
		t.Fatalf("install %q %v", ver, err)
	}
	st, err := l.Status(loader.Target{ProfileDir: profile})
	if err != nil || !st.Installed || st.Version != "0.16.9" || st.GameVersion != "1.21.1" || !st.PerProfile {
		t.Fatalf("status %+v %v", st, err)
	}
	plan := launchplan.New(launchplan.ModeProfile)
	if err := l.Contribute(t.Context(), plan, loader.ProfileView{Dir: profile, Game: "minecraft"}); err != nil {
		t.Fatal(err)
	}
	if plan.Exe != java || !slices.Contains(plan.Args, "net.fabricmc.loader.impl.launch.knot.KnotClient") {
		t.Fatalf("plan exe=%q args=%v", plan.Exe, plan.Args)
	}
	if len(plan.RuntimeReqs) != 1 || plan.RuntimeReqs[0].Kind != "java" || plan.RuntimeReqs[0].Value != java {
		t.Fatalf("runtime %+v", plan.RuntimeReqs)
	}
	if !slices.Contains(plan.Args, profile) {
		t.Fatalf("args %v", plan.Args)
	}
}

func TestForgeAndNeoForgeRegisterAndSetTheirMainClass(t *testing.T) {
	t.Parallel()
	java := filepath.Join(t.TempDir(), "java")
	testfs.WriteAt(t, java, []byte("x"))
	for _, kind := range []string{Forge, NeoForge} {
		l := Loader{Kind: kind, Java: java}
		profile := t.TempDir()
		if _, err := l.Install(t.Context(), loader.Target{ProfileDir: profile}, loader.Package{Version: "1"}, nil); err != nil {
			t.Fatal(err)
		}
		plan := launchplan.New(launchplan.ModeProfile)
		if err := l.Contribute(t.Context(), plan, loader.ProfileView{Dir: profile}); err != nil {
			t.Fatal(kind, err)
		}
		if !strings.Contains(strings.Join(plan.Args, " "), "bootstraplauncher") {
			t.Fatalf("%s args %v", kind, plan.Args)
		}
		got, ok := loader.Get(kind)
		if !ok || got.ID() != kind {
			t.Fatalf("registered %s: %v %v", kind, got, ok)
		}
	}
	if !Family([]string{"fabric", "folder"}) || Family([]string{"folder"}) {
		t.Fatal("Family")
	}
}

func TestVersionsReadsFabricAndMavenLists(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/fabric", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"version":"0.16.9","stable":true},{"version":"0.16.8","stable":false}]`))
	})
	mux.HandleFunc("/maven", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<metadata><versioning><latest>2</latest><versions><version>1</version><version>2</version></versions></versioning></metadata>`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	fab := Loader{Kind: Fabric, HTTP: srv.Client()}
	fabURL := srv.URL + "/fabric"
	got, err := fabricVersions(mustGet(t, fab, fabURL))
	if err != nil || !slices.Equal(got, []string{"0.16.9"}) {
		t.Fatalf("fabric %v %v", got, err)
	}
	body, err := fab.get(t.Context(), srv.URL+"/maven")
	if err != nil {
		t.Fatal(err)
	}
	got, err = mavenVersions(body)
	if err != nil || !slices.Equal(got, []string{"2", "1"}) {
		t.Fatalf("maven %v %v", got, err)
	}
}

func mustGet(t *testing.T, l Loader, url string) []byte {
	t.Helper()
	b, err := l.get(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestContributeWithoutJavaFails(t *testing.T) {
	t.Parallel()
	l := Loader{Kind: Fabric, Java: filepath.Join(t.TempDir(), "missing")}
	profile := t.TempDir()
	if _, err := l.Install(t.Context(), loader.Target{ProfileDir: profile}, loader.Package{Version: "1"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := l.Contribute(t.Context(), launchplan.New(launchplan.ModeProfile), loader.ProfileView{Dir: profile}); err == nil {
		t.Fatal("want missing java")
	}
}
