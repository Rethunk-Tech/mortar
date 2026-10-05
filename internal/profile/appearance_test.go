package profile

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestSetAppearance(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	got, err := s.SetAppearance("stardew", p.ID, " teal ", "sprout", "  co-op Fridays  ")
	if err != nil || got.Color != "teal" || got.Icon != "sprout" || got.Description != "co-op Fridays" {
		t.Fatalf("set = %+v, %v", got, err)
	}
	listed, err := s.List("stardew")
	if err != nil || listed[0].Color != "teal" || listed[0].Icon != "sprout" || listed[0].Description != "co-op Fridays" {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	raw, err := fsx.ReadFile(filepath.Join(s.root, "stardew", p.ID, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"color": "teal"`) || !strings.Contains(string(raw), `"icon": "sprout"`) {
		t.Fatalf("json: %s", raw)
	}
	cleared, err := s.SetAppearance("stardew", p.ID, "", "", "")
	if err != nil || cleared.Color != "" || cleared.Icon != "" || cleared.Description != "" {
		t.Fatalf("clear = %+v, %v", cleared, err)
	}
}

func TestSetAppearanceRejects(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	if _, err := s.SetAppearance("stardew", p.ID, "neon", "", ""); err == nil {
		t.Fatal("unknown colour accepted")
	}
	if _, err := s.SetAppearance("stardew", p.ID, "", "dragon", ""); err == nil {
		t.Fatal("unknown icon accepted")
	}
	if _, err := s.SetAppearance("stardew", p.ID, "", "", strings.Repeat("x", MaxDescription+1)); err == nil {
		t.Fatal("long description accepted")
	}
	ok := strings.Repeat("é", MaxDescription)
	if _, err := s.SetAppearance("stardew", p.ID, "rose", "star", ok); err != nil {
		t.Fatalf("max runes rejected: %v", err)
	}
}

func TestLoadSanitizesAppearance(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	path := filepath.Join(s.root, "stardew", p.ID, fileName)
	raw, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["color"] = "neon"
	doc["icon"] = "dragon"
	doc["description"] = strings.Repeat("x", MaxDescription+1)
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := s.List("stardew")
	if err != nil || len(got) != 1 {
		t.Fatalf("list = %+v, %v", got, err)
	}
	if got[0].Color != "" || got[0].Icon != "" || got[0].Description != "" {
		t.Fatalf("unsanitized = %+v", got[0])
	}
}
