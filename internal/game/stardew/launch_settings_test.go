package stardew

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func TestDirectCommandIncludesPrefixAndEnvironment(t *testing.T) {
	g := Game{DataDir: t.TempDir()}
	req := launch.Request{
		InstallDir: filepath.FromSlash("/games/Stardew Valley"),
		ModsDir:    filepath.FromSlash("/data/profile/mods"),
		Direct:     true,
		Prefix:     []string{"gamemoderun", "mangohud"},
		Env:        []string{"MORTAR_TEST=1"},
	}
	cmd, err := g.command("linux", req, "", "")
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"gamemoderun", "mangohud", "--skip-terminal", "--", "--mods-path", req.ModsDir}
	if !reflect.DeepEqual(cmd.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", cmd.Args, wantArgs)
	}
	if !reflect.DeepEqual(cmd.Env, req.Env) {
		t.Fatalf("env = %#v, want %#v", cmd.Env, req.Env)
	}
}
