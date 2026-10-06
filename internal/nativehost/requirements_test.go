package nativehost

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
)

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

func TestModReplyCarriesRequirements(t *testing.T) {
	dir := listenControl(t, `{"global":{"lastProfile":{"stardew":"p1"}}}`)
	cache := []byte(`{"fetched":"2026-01-01T00:00:00Z","value":{"page":{"requirements":[{"modId":1915,"name":"Content Patcher"},{"modId":2400,"name":"GMCM"},{"name":"SMAPI","external":true}]},"partial":true}}`)
	path := filepath.Join(dir, "cache", filepath.FromSlash(nexussvc.PageName("stardewvalley", 2400)))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, cache, 0o600); err != nil {
		t.Fatal(err)
	}
	pdir := filepath.Join(dir, "profiles", "stardew", "p1")
	if err := os.MkdirAll(pdir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pdir, "profile.json"), []byte(`{"name":"Main","entries":[{"source":{"kind":"nexus","modId":1915}}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := ServeFrom(nil, bytes.NewReader(frame(t, request{Type: "mod", Source: "nexus", SourceGameKey: "stardewvalley", ModID: 2400})), &out, func(string) error { return nil }); err != nil {
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

func TestModReplyOmitsRequirementsWithoutCache(t *testing.T) {
	listenControl(t, `{"global":{"lastProfile":{"stardew":"p1"}}}`)
	var out bytes.Buffer
	if err := ServeFrom(nil, bytes.NewReader(frame(t, request{Type: "mod", Source: "nexus", SourceGameKey: "stardewvalley", ModID: 1})), &out, func(string) error { return nil }); err != nil {
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
