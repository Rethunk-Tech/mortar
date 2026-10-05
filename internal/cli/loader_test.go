package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func TestLoaderCommands(t *testing.T) {
	results := map[string]any{
		"loader.versions": []string{"4.1.0", "4.0.0"},
		"loader.install":  loader.Status{Version: "4.0.0", Installed: true},
		"loader.pin":      nil,
	}
	r := invoke(t, results, "loader", "versions", "stardew")
	if r.code != 0 || r.calls[0].method != "loader.versions" || r.calls[0].params.Game != "stardew" || !strings.Contains(r.out, "4.0.0") {
		t.Fatalf("versions: %+v", r)
	}
	r = invoke(t, results, "loader", "install", "stardew", "4.0.0")
	if r.code != 0 || r.calls[0].method != "loader.install" || r.calls[0].params.Name != "4.0.0" {
		t.Fatalf("install: %+v", r)
	}
	r = invoke(t, results, "loader", "pin", "stardew", "latest")
	if r.code != 0 || r.calls[0].method != "loader.pin" || r.calls[0].params.Value != "" {
		t.Fatalf("pin latest: %+v", r)
	}
	r = invoke(t, results, "loader", "pin", "stardew", "4.0.0", "--loader", "smapi")
	if r.code != 0 || r.calls[0].params.Value != "4.0.0" || r.calls[0].params.Loader != "smapi" {
		t.Fatalf("pin: %+v", r)
	}
}
