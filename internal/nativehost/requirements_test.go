package nativehost

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/datadir"
)

func TestParsePageRequirements(t *testing.T) {
	t.Parallel()
	items := parsePageRequirements([]byte(`{
		"fetched": "2026-01-01T00:00:00Z",
		"value": [
			{"ModID": 1915, "Name": "Content Patcher"},
			{"ModID": 2400, "Name": "Generic Mod Config Menu"},
			{"ModID": 0, "Name": "SMAPI"}
		]
	}`))
	if len(items) != 3 {
		t.Fatalf("got %d items", len(items))
	}
	if items[0].ModID != 1915 || items[0].Name != "Content Patcher" || items[0].External {
		t.Fatalf("first: %+v", items[0])
	}
	if !items[2].External || items[2].Name != "SMAPI" {
		t.Fatalf("external: %+v", items[2])
	}
}

func TestParsePageRequirementsEmpty(t *testing.T) {
	t.Parallel()
	if parsePageRequirements([]byte(`{"fetched":"2026-01-01T00:00:00Z","value":[]}`)) != nil {
		t.Fatal("empty cache should yield no items")
	}
	if parsePageRequirements([]byte(`not json`)) != nil {
		t.Fatal("invalid json should yield no items")
	}
}

func TestMarkRequirementPresence(t *testing.T) {
	t.Parallel()
	items := markRequirementPresence(
		[]requirementItem{
			{Name: "Content Patcher", ModID: 1915},
			{Name: "GMCM", ModID: 2400},
			{Name: "SMAPI", External: true},
		},
		[]int{1915, 99},
	)
	if !items[0].Present || items[1].Present || items[2].Present {
		t.Fatalf("presence: %+v", items)
	}
}

func TestRequirementsRequest(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	cache := []byte(`{"fetched":"2026-01-01T00:00:00Z","value":[{"ModID":1915,"Name":"Content Patcher"},{"ModID":2400,"Name":"GMCM"},{"ModID":0,"Name":"SMAPI"}]}`)
	if err := os.MkdirAll(filepath.Join(dir, "cache"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cache", "nexus-requirements-2400.json"), cache, 0o600); err != nil {
		t.Fatal(err)
	}
	pdir := filepath.Join(dir, "profiles", "stardew", "p1")
	if err := os.MkdirAll(pdir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pdir, "profile.json"), []byte(`{"name":"Main","entries":[{"source":{"kind":"nexus","modId":1915}}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"lastProfile":{"stardew":"p1"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := Serve(bytes.NewReader(frame(t, request{Type: "requirements", Game: "stardewvalley", ModID: 2400})), &out, func(string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	var n uint32
	if err := binary.Read(&out, binary.NativeEndian, &n); err != nil {
		t.Fatal(err)
	}
	var rep reply
	if err := json.Unmarshal(out.Next(int(n)), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Error != "" {
		t.Fatal(rep.Error)
	}
	if len(rep.Requirements) != 3 {
		t.Fatalf("got %+v", rep.Requirements)
	}
	if !rep.Requirements[0].Present || rep.Requirements[1].Present || rep.Requirements[2].Present {
		t.Fatalf("presence %+v", rep.Requirements)
	}
	if rep.Accent == "" {
		t.Fatal("accent missing")
	}
}

func TestRequirementsRequestNoCache(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	var out bytes.Buffer
	if err := Serve(bytes.NewReader(frame(t, request{Type: "requirements", Game: "stardewvalley", ModID: 1})), &out, func(string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	var n uint32
	if err := binary.Read(&out, binary.NativeEndian, &n); err != nil {
		t.Fatal(err)
	}
	var rep reply
	if err := json.Unmarshal(out.Next(int(n)), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Requirements != nil {
		t.Fatalf("want omit, got %+v", rep.Requirements)
	}
}
