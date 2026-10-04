package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func TestSmapiCommands(t *testing.T) {
	results := map[string]any{
		"smapi.versions": []string{"4.1.0", "4.0.0"},
		"smapi.install":  loader.Status{Version: "4.0.0", Installed: true},
		"smapi.pin":      nil,
	}
	r := invoke(t, results, "smapi", "versions", "stardew")
	if r.code != 0 || r.calls[0].method != "smapi.versions" || r.calls[0].params.Game != "stardew" || !strings.Contains(r.out, "4.0.0") {
		t.Fatalf("versions: %+v", r)
	}
	r = invoke(t, results, "smapi", "install", "stardew", "4.0.0")
	if r.code != 0 || r.calls[0].method != "smapi.install" || r.calls[0].params.Name != "4.0.0" {
		t.Fatalf("install: %+v", r)
	}
	r = invoke(t, results, "smapi", "pin", "stardew", "latest")
	if r.code != 0 || r.calls[0].method != "smapi.pin" || r.calls[0].params.Value != "" {
		t.Fatalf("pin latest: %+v", r)
	}
	r = invoke(t, results, "smapi", "pin", "stardew", "4.0.0")
	if r.code != 0 || r.calls[0].params.Value != "4.0.0" {
		t.Fatalf("pin: %+v", r)
	}
}
