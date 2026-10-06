package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
)

type bepinex struct{ need bool }

func (b bepinex) NeedsWinHTTPOverride() bool { return b.need }

func TestProtonChecks(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	library := filepath.Join(t.TempDir(), "Lib")
	compat := filepath.Join(library, "steamapps", "compatdata", "1966720")
	if err := os.MkdirAll(filepath.Join(compat, "pfx"), 0o750); err != nil {
		t.Fatal(err)
	}
	g := game.GameInfo{ID: "lc", Name: "Lethal Company", AppID: "1966720", Installs: []game.Install{
		{Store: "flatpak-steam", Dir: filepath.Join(library, "steamapps", "common", "Lethal Company"), Runtime: "proton", Platform: "windows"},
	}}
	launch, show := "", ""
	flatpakOverrides = func() (string, error) { return show, nil }
	launchOptions = func(string) string { return launch }
	t.Cleanup(func() { delete(loaders, "lc") })

	status := func() map[string]string {
		out := map[string]string{}
		for _, c := range protonChecks(g) {
			out[c.ID] = c.Status
		}
		return out
	}
	// No loader provides the winhttp need: only the Flatpak check runs.
	if got := status(); len(got) != 1 || got["flatpakCompatdata:lc"] != Warn {
		t.Fatalf("no loader: %v", got)
	}
	RegisterLoader("lc", bepinex{need: true})
	for _, c := range protonChecks(g) {
		if c.ID == "winhttp:lc" && !strings.Contains(c.Detail, "has not created its prefix") {
			t.Fatalf("no registry yet: %+v", c)
		}
	}
	if err := fsx.WriteFile(filepath.Join(compat, "pfx", "user.reg"), []byte("WINE REGISTRY\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := status(); got["winhttp:lc"] != Warn {
		t.Fatalf("missing override: %v", got)
	}
	launch = `WINEDLLOVERRIDES="winhttp=n,b" %command%`
	show = "[Context]\nfilesystems=" + library + ":ro;"
	if got := status(); got["winhttp:lc"] != Pass || got["flatpakCompatdata:lc"] != Pass {
		t.Fatalf("launch option and grant: %v", got)
	}
	launch = ""
	if err := fsx.WriteFile(filepath.Join(compat, "pfx", "user.reg"), []byte(`"winhttp"="native,builtin"`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := status(); got["winhttp:lc"] != Pass {
		t.Fatalf("registry override: %v", got)
	}
	// A native install gets none of these.
	g.Installs[0].Runtime = "native"
	if got := status(); len(got) != 0 {
		t.Fatalf("native install: %v", got)
	}
}
