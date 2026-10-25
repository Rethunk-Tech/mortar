package devsvc

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestGameCodeSearchesAndDecompilesTheGamesAssembly(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	if _, err := svc.GameCode("lethal-company", dir, "x"); err == nil {
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
	hits, err := svc.GameCode("lethal-company", dir, "MOD switch")
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
	if _, err := svc.Decompile(t.Context(), "lethal-company", dir, "-o /tmp"); err == nil {
		t.Fatal("decompiled a name that is a flag")
	}
	src, err := svc.Decompile(t.Context(), "lethal-company", dir, "Fixture.Mod")
	if err != nil || src != "class Mod {}" || !slices.Equal(args, []string{"-t", "Fixture.Mod", filepath.Join(managed, "Assembly-CSharp.dll")}) {
		t.Fatalf("decompile = %q %v %v", src, args, err)
	}
}
