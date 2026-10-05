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

func TestAutomationVerbs(t *testing.T) {
	results := map[string]any{"queue.add": map[string]any{}, "bisect.start": map[string]any{"id": "bisect-1"}, "store.check": map[string]any{"checked": 3}}
	r := invoke(t, results, "queue", "add", "stardew", "Farm", "1915", "--source", "nexus", "--file", "7", "--version", "2.0")
	if p := r.calls[0].params; r.code != 0 || r.calls[0].method != "queue.add" || p.Source != "nexus" || p.ID != "1915" || p.File != "7" || p.Version != "2.0" {
		t.Fatalf("queue add: %+v", r)
	}
	if r = invoke(t, results, "queue", "add", "stardew", "Farm", "1915"); r.code != 2 || len(r.calls) != 0 {
		t.Fatalf("queue add without a source: %+v", r)
	}
	if r = invoke(t, results, "bisect", "start", "stardew", "Farm"); r.code != 0 || r.calls[0].method != "bisect.start" || r.calls[0].params.Profile != "Farm" {
		t.Fatalf("bisect start: %+v", r)
	}
	if r = invoke(t, results, "store", "check", "stardew"); r.code != 0 || r.calls[0].method != "store.check" {
		t.Fatalf("store check: %+v", r)
	}
}

func TestMoreVerbsMapOntoControlMethods(t *testing.T) {
	results := map[string]any{"bundles.create": nil, "templates.preview": map[string]any{"add": []string{"A"}}, "data.move": nil, "links.enable": nil, "lan.send": nil, "lan.pair": nil, "lan.unpair": nil}
	r := invoke(t, results, "bundles", "create", "stardew", "Core", "Farm", "smapi:A", "smapi:B")
	if p := r.calls[0].params; r.code != 0 || r.calls[0].method != "bundles.create" || p.Name != "Core" || p.Profile != "Farm" || len(p.IDs) != 2 {
		t.Fatalf("bundles create: %+v", r)
	}
	if r = invoke(t, results, "templates", "apply", "stardew", "T", "Farm", "--preview"); r.calls[0].method != "templates.preview" {
		t.Fatalf("templates apply --preview: %+v", r)
	}
	if r = invoke(t, results, "data", "move", "/tmp/x", "--preview"); r.calls[0].method != "data.move" || !r.calls[0].params.Preview {
		t.Fatalf("data move: %+v", r)
	}
	if r = invoke(t, results, "links", "enable"); r.code != 2 || len(r.calls) != 0 {
		t.Fatalf("links enable without a source: %+v", r)
	}
	if r = invoke(t, results, "lan", "send", "stardew", "Farm", "peer-1"); r.calls[0].params.Name != "peer-1" {
		t.Fatalf("lan send: %+v", r)
	}
	if r = invoke(t, results, "lan", "pair", "--code", "ABCD-EFGH", "--peer", "10.0.0.2:47630"); r.calls[0].method != "lan.pair" || r.calls[0].params.Name != "10.0.0.2:47630" || r.calls[0].params.Value != "ABCD-EFGH" {
		t.Fatalf("lan pair: %+v", r)
	}
	if r = invoke(t, results, "lan", "unpair", "abc"); r.calls[0].method != "lan.unpair" || r.calls[0].params.Name != "abc" {
		t.Fatalf("lan unpair: %+v", r)
	}
}
