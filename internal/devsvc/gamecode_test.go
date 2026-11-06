package devsvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func TestGameCodeSearchesAndDecompilesTheGamesAssembly(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	if _, err := svc.GameCode("lethal-company", dir, "", "x"); err == nil {
		t.Fatal("searched without a game folder")
	}
	managed := gameWithModDLL(t, dir)
	mvid, _ := dotnet.Identity(filepath.Join(managed, "Assembly-CSharp.dll"))
	if got, err := svc.FrameAssembly("lethal-company", dir, mvid); err != nil || got != filepath.Join(managed, "Assembly-CSharp.dll") {
		t.Fatalf("FrameAssembly(%s) = %q, %v", mvid, got, err)
	}
	if got, err := svc.FrameAssembly("lethal-company", dir, "00000000000000000000000000000000"); err != nil || got != "" {
		t.Fatalf("FrameAssembly(unknown) = %q, %v", got, err)
	}
	hits, err := svc.GameCode("lethal-company", dir, "", "MOD switch")
	if err != nil || len(hits) != 1 || hits[0].Type != "Fixture.Mod" || hits[0].Method != "SwitchTool" {
		t.Fatalf("hits = %+v, %v", hits, err)
	}

	var args []string
	svc.Run = func(_ context.Context, _, _ string, a ...string) ([]byte, error) {
		args = a
		return []byte("class Mod {}"), nil
	}
	tools := filepath.Join(svc.Home, ".dotnet", "tools")
	if err := os.MkdirAll(tools, 0o750); err != nil {
		t.Fatal(err)
	}
	tool := "ilspycmd"
	if runtime.GOOS == "windows" {
		tool += ".exe"
	}
	if err := safefs.WriteFile(filepath.Join(tools, tool), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Decompile(t.Context(), "lethal-company", dir, "", "-o /tmp", false); err == nil {
		t.Fatal("decompiled a name that is a flag")
	}
	src, err := svc.Decompile(t.Context(), "lethal-company", dir, "", "Fixture.Mod", false)
	if err != nil || src != "class Mod {}" || !slices.Equal(args, []string{"-t", "Fixture.Mod", filepath.Join(managed, "Assembly-CSharp.dll")}) {
		t.Fatalf("decompile = %q %v %v", src, args, err)
	}
	if _, err := svc.Decompile(t.Context(), "lethal-company", dir, filepath.Join(managed, "Assembly-CSharp.dll"), "Fixture.Mod", true); err != nil || args[0] != "-il" {
		t.Fatalf("IL view %v %v", args, err)
	}
	// Side by side reads a copy in the cache, so each run must name the game's folder for its references.
	var runs [][]string
	svc.Run = func(_ context.Context, _, _ string, a ...string) ([]byte, error) {
		runs = append(runs, a)
		return []byte("class Mod {}"), nil
	}
	if _, err := svc.DecompileMapped(t.Context(), "lethal-company", dir, "", "Fixture.Mod"); err != nil || len(runs) != 3 {
		t.Fatalf("mapped: %d runs, %v", len(runs), err)
	}
	for _, r := range runs {
		if i := slices.Index(r, "-r"); i < 0 || r[i+1] != managed {
			t.Fatalf("ILSpy ran without the game's folder as its reference path: %v", r)
		}
	}
	if _, err := svc.GameCode("lethal-company", dir, "/etc/passwd", "x"); err == nil {
		t.Fatal("read an assembly Game code does not list")
	}
}

func TestLicenseAndSDKNeed(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apache-2.0" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"key":"apache-2.0","body":"Apache License\nVersion 2.0"}`))
	}))
	defer srv.Close()
	svc.LicensesURL = srv.URL + "/"
	dir := t.TempDir()
	if err := svc.applyLicense(t.Context(), dir, "apache-2.0"); err != nil {
		t.Fatal(err)
	}
	if body, _ := safefs.ReadFile(filepath.Join(dir, "LICENSE")); string(body) != "Apache License\nVersion 2.0" {
		t.Fatalf("LICENSE = %q", body)
	}
	if err := svc.applyLicense(t.Context(), dir, "../x"); err == nil {
		t.Fatal("wrote a license New mod does not offer")
	}
	if err := safefs.WriteFile(filepath.Join(dir, "global.json"), []byte(`{"sdk":{"version":"10.0.100","rollForward":"latestMajor"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readSDKNeed(dir); got != (SDKNeed{Version: "10.0.100", RollForward: "latestMajor"}) {
		t.Fatalf("sdk = %+v", got)
	}
}

func TestILBlock(t *testing.T) {
	il := ".class public A.B\n{\n\t.class nested public C\n\t{\n\t} // end of class C\n} // end of class A.B\n.class public A.D\n{\n} // end of class A.D"
	if got := ilBlock(il, "A.D"); got != ".class public A.D\n{\n} // end of class A.D" {
		t.Errorf("A.D = %q", got)
	}
	if got := ilBlock(il, "A.B+C"); got != "\t.class nested public C\n\t{\n\t} // end of class C" {
		t.Errorf("A.B+C = %q", got)
	}
	if got := ilBlock(il, "A.Missing"); got != il {
		t.Error("a missing type should leave the listing whole")
	}
}

func TestUsagesSearchesTheListedAssemblies(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	gameWithModDLL(t, dir)
	got, err := svc.Usages("lethal-company", dir, "Fixture.Mod::set_Own")
	if err != nil || len(got) != 1 || got[0].Caller != "Fixture.Mod::SwitchTool" || got[0].Label != "Assembly-CSharp.dll" {
		t.Fatalf("usages = %+v, %v", got, err)
	}
}

func TestWriteUserPropsEscapesTheGamePathForMSBuild(t *testing.T) {
	dir := t.TempDir()
	if err := writeUserProps(dir, filepath.Join(t.TempDir(), "50%25 off")); err != nil {
		t.Fatal(err)
	}
	if body, _ := safefs.ReadFile(filepath.Join(dir, "Config.Build.user.props")); !strings.Contains(string(body), "50%2525 off") {
		t.Errorf("props = %s", body)
	}
}

func TestWriteUserPropsRefusesAGamePathXMLCannotCarry(t *testing.T) {
	dir := t.TempDir()
	if err := writeUserProps(dir, filepath.Join(t.TempDir(), "bad\x01")); err == nil {
		t.Fatal("wrote a props file XML cannot parse")
	}
	if _, err := os.Stat(filepath.Join(dir, "Config.Build.user.props")); !os.IsNotExist(err) {
		t.Errorf("props written: %v", err)
	}
}

func TestAssembliesListTheGamesMainAssemblyFirst(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	game := t.TempDir()
	managed := filepath.Join(game, "Game_Data", "Managed")
	if err := os.MkdirAll(managed, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Another.dll", "Assembly-CSharp.dll", "Zed.dll", "System.Core.dll"} {
		if err := safefs.WriteFile(filepath.Join(managed, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeUserProps(dir, game); err != nil {
		t.Fatal(err)
	}
	all, err := svc.Assemblies("lethal-company", dir)
	if err != nil {
		t.Fatal(err)
	}
	var labels []string
	for _, a := range all {
		labels = append(labels, a.Label)
	}
	if !slices.Equal(labels, []string{"Assembly-CSharp.dll", "Another.dll", "Zed.dll"}) {
		t.Errorf("assemblies = %v", labels)
	}
}

// gameWithModDLL gives the project a game folder whose Assembly-CSharp.dll is the test mod, and returns its Managed folder.
func gameWithModDLL(t *testing.T, dir string) string {
	t.Helper()
	game := t.TempDir()
	dll, err := safefs.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(game, "Game_Data", "Managed")
	if err := os.MkdirAll(managed, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(filepath.Join(managed, "Assembly-CSharp.dll"), dll, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeUserProps(dir, game); err != nil {
		t.Fatal(err)
	}
	return managed
}

func TestAssembliesWithoutAProjectUseTheInstallAndTheOpenProfile(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	items, profiles := testenv.Stores(t)
	st, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc.Settings = settings.NewService(st)
	svc.Profiles = profile.NewService(profiles, t.TempDir(), st)

	game := t.TempDir()
	managed := filepath.Join(game, "Game_Data", "Managed")
	pkg := filepath.Join(t.TempDir(), "pkg")
	for _, dir := range []string{managed, pkg} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"Another.dll", "Assembly-CSharp.dll", "System.Core.dll"} {
		if err := safefs.WriteFile(filepath.Join(managed, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := safefs.WriteFile(filepath.Join(game, "Lethal Company.exe"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(filepath.Join(pkg, "Dep.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := items.AddDir(t.Context(), "lethal-company", "dep-1", pkg); err != nil {
		t.Fatal(err)
	}
	open := testenv.Profile(t, profiles, "lethal-company", "Open")
	if _, err := profiles.AddEntry("lethal-company", open.ID, "dep-1", profile.Source{Kind: profile.KindThunderstore, Name: "Auth-Dep"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Settings.SetGameFolder("lethal-company", game); err != nil {
		t.Fatal(err)
	}
	if err := svc.Settings.SetLastProfile("lethal-company", open.ID); err != nil {
		t.Fatal(err)
	}

	all, err := svc.Assemblies("lethal-company", "")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, a := range all {
		got = append(got, a.Group+":"+a.Label)
	}
	if want := []string{"game:Assembly-CSharp.dll", "game:Another.dll", "dependency:Auth-Dep/Dep.dll"}; !slices.Equal(got, want) {
		t.Fatalf("assemblies = %v, want %v", got, want)
	}
	if path, err := svc.assembly("lethal-company", "", ""); err != nil || path != filepath.Join(managed, "Assembly-CSharp.dll") {
		t.Fatalf("default assembly = %q, %v", path, err)
	}
	if _, err := svc.GameCode("lethal-company", "", "/etc/passwd", "x"); err == nil {
		t.Fatal("read an assembly Game code does not list")
	}
}
