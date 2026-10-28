package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/control"
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
	if r = invoke(t, results, "queue", "add", "stardew", "Farm", "1915"); r.code != 2 || len(r.calls) != 0 || !strings.Contains(r.errOut, "vsmoddb") {
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
	results := map[string]any{"groups.create": nil, "templates.preview": map[string]any{"add": []string{"A"}}, "data.move": nil, "links.enable": nil, "lan.send": nil, "lan.pair": nil, "lan.unpair": nil}
	r := invoke(t, results, "groups", "create", "stardew", "Core", "Farm", "smapi:A", "smapi:B")
	if p := r.calls[0].params; r.code != 0 || r.calls[0].method != "groups.create" || p.Name != "Core" || p.Profile != "Farm" || len(p.IDs) != 2 {
		t.Fatalf("groups create: %+v", r)
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
	if r = invoke(t, results, "links", "enable", "--source", "vsmoddb"); r.code != 0 || r.calls[0].method != "links.enable" || r.calls[0].params.Source != "vsmoddb" {
		t.Fatalf("links enable vsmoddb: %+v", r)
	}
	results["links"] = []linkOwner{
		{Scheme: "nxm", Name: "Mortar", ID: "mortar", Mine: true},
		{Scheme: "ror2mm", Name: "r2modman", ID: "r2", Mine: false},
	}
	if r = invoke(t, results, "links"); r.code != 0 || r.calls[len(r.calls)-1].method != "links" || !strings.Contains(r.out, "nxm") || !strings.Contains(r.out, "r2modman") {
		t.Fatalf("links list: %+v %q", r, r.out)
	}
	if r = invoke(t, results, "links", "list"); r.code != 0 || r.calls[len(r.calls)-1].method != "links" {
		t.Fatalf("links list subverb: %+v", r)
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

func TestLanPairJSONPrintsOneJSONResult(t *testing.T) {
	paired := 0
	call := func(method string, _ control.Params, out any, _ time.Duration) error {
		var v any
		switch method {
		case "lan.paircode":
			v = "ABCD-EFGH"
		case "lan.paired":
			paired++
			v = []map[string]string{}
			if paired > 1 {
				v = []map[string]string{{"ID": "p1", "Name": "Laptop"}}
			}
		}
		b, _ := json.Marshal(v)
		return json.Unmarshal(b, out)
	}
	var o, e bytes.Buffer
	if code := run("9.9.9", call, []string{"--json", "lan", "pair"}, &o, &e); code != 0 {
		t.Fatalf("code %d: %s", code, e.String())
	}
	var got map[string]string
	if err := json.Unmarshal(o.Bytes(), &got); err != nil || got["name"] != "Laptop" || !strings.Contains(e.String(), "ABCD-EFGH") {
		t.Fatalf("stdout %q stderr %q: %v", o.String(), e.String(), err)
	}
}
