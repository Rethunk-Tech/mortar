package devsvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestGameCodeSearchesAndDecompilesTheGamesAssembly(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	if _, err := svc.GameCode("lethal-company", dir, "", "x"); err == nil {
		t.Fatal("searched without a game folder")
	}
	game := t.TempDir()
	dll, err := fsx.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(game, "Game_Data", "Managed")
	if err := os.MkdirAll(managed, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(managed, "Assembly-CSharp.dll"), dll, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeUserProps(dir, game); err != nil {
		t.Fatal(err)
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
	if err := fsx.WriteFile(filepath.Join(tools, "ilspycmd"), nil, 0o600); err != nil {
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
	if body, _ := fsx.ReadFile(filepath.Join(dir, "LICENSE")); string(body) != "Apache License\nVersion 2.0" {
		t.Fatalf("LICENSE = %q", body)
	}
	if err := svc.applyLicense(t.Context(), dir, "../x"); err == nil {
		t.Fatal("wrote a license New mod does not offer")
	}
	if err := fsx.WriteFile(filepath.Join(dir, "global.json"), []byte(`{"sdk":{"version":"10.0.100","rollForward":"latestMajor"}}`), 0o600); err != nil {
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
	game := t.TempDir()
	dll, err := fsx.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(game, "Game_Data", "Managed")
	if err := os.MkdirAll(managed, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(managed, "Assembly-CSharp.dll"), dll, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeUserProps(dir, game); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Usages("lethal-company", dir, "Fixture.Mod::set_Own")
	if err != nil || len(got) != 1 || got[0].Caller != "Fixture.Mod::SwitchTool" || got[0].Label != "Assembly-CSharp.dll" {
		t.Fatalf("usages = %+v, %v", got, err)
	}
}
