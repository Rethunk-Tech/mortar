package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestDevCommands(t *testing.T) {
	results := map[string]any{
		"dev.add":    settings.DevProject{Name: "Shop", Dir: "/p/shop"},
		"dev.clone":  settings.DevProject{Name: "Shop", Dir: "/p/shop"},
		"dev.list":   []settings.DevProject{{Name: "Shop", Dir: "/p/shop"}},
		"dev.remove": settings.DevProject{Name: "Shop", Dir: "/p/shop"},
	}
	r := invoke(t, results, "dev", "add", "lethal-company", "mods/shop", "--profile", "dev")
	abs, _ := filepath.Abs("mods/shop")
	if p := r.calls[0].params; r.code != 0 || r.calls[0].method != "dev.add" || p.Game != "lethal-company" || p.Path != abs || p.Profile != "dev" {
		t.Fatalf("add: %+v", r)
	}
	r = invoke(t, results, "dev", "clone", "lethal-company", "o/shop", "--dir", "/work")
	if p := r.calls[0].params; r.code != 0 || r.calls[0].method != "dev.clone" || p.ID != "o/shop" || p.Path != "/work" {
		t.Fatalf("clone: %+v", r)
	}
	if r = invoke(t, results, "dev", "list", "lethal-company"); r.code != 0 || r.out != "Shop\t/p/shop\n" || r.calls[0].params.Game != "lethal-company" {
		t.Fatalf("list: %+v", r)
	}
	r = invoke(t, results, "dev", "remove", "lethal-company", "Shop")
	if p := r.calls[0].params; r.code != 0 || r.calls[0].method != "dev.remove" || p.ID != "Shop" || !strings.Contains(r.out, "Removed Shop") {
		t.Fatalf("remove: %+v", r)
	}
	if r = invoke(t, results, "dev", "add", "lethal-company"); r.code != 2 || len(r.calls) != 0 {
		t.Fatalf("add without a folder: %+v", r)
	}
}
