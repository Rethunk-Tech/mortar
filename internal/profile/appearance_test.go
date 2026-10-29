package profile

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestSetAppearance(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	got, err := s.SetAppearance("stardew", p.ID, " teal ", "sprout")
	if err != nil || got.Color != "teal" || got.Icon != "sprout" {
		t.Fatalf("set = %+v, %v", got, err)
	}
	listed, err := s.List("stardew")
	if err != nil || listed[0].Color != "teal" || listed[0].Icon != "sprout" {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	raw, err := safefs.ReadFile(filepath.Join(s.root, "stardew", p.ID, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"color": "teal"`) || !strings.Contains(string(raw), `"icon": "sprout"`) {
		t.Fatalf("json: %s", raw)
	}
	cleared, err := s.SetAppearance("stardew", p.ID, "", "")
	if err != nil || cleared.Color != "" || cleared.Icon != "" {
		t.Fatalf("clear = %+v, %v", cleared, err)
	}
}

func TestSetAppearanceRejects(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	if _, err := s.SetAppearance("stardew", p.ID, "neon", ""); err == nil {
		t.Fatal("unknown colour accepted")
	}
	if _, err := s.SetAppearance("stardew", p.ID, "", "dragon"); err == nil {
		t.Fatal("unknown icon accepted")
	}
}

func TestLoadSanitizesAppearance(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	path := filepath.Join(s.root, "stardew", p.ID, fileName)
	editProfileJSON(t, path, func(doc map[string]any) {
		doc["color"] = "neon"
		doc["icon"] = "dragon"
	})
	got, err := s.List("stardew")
	if err != nil || len(got) != 1 {
		t.Fatalf("list = %+v, %v", got, err)
	}
	if got[0].Color != "" || got[0].Icon != "" {
		t.Fatalf("unsanitized = %+v", got[0])
	}
}

func TestLoadFoldsDescriptionIntoNotes(t *testing.T) {
	t.Parallel()
	for notes, want := range map[string]string{"": "co-op Fridays", "Read me": "co-op Fridays\n\nRead me"} {
		s := newStore(t)
		p := mustCreate(t, s, "Farm")
		path := filepath.Join(s.root, "stardew", p.ID, fileName)
		editProfileJSON(t, path, func(doc map[string]any) {
			doc["notes"] = notes
			doc["description"] = "co-op Fridays"
		})
		got, err := s.Get("stardew", p.ID)
		if err != nil || got.Notes != want {
			t.Fatalf("notes %q: got %q, %v", notes, got.Notes, err)
		}
		if _, err := s.SetAppearance("stardew", p.ID, "teal", ""); err != nil {
			t.Fatal(err)
		}
		raw, err := safefs.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), `"description"`) || !strings.Contains(string(raw), `"notes": "co-op Fridays`) {
			t.Fatalf("rewritten: %s", raw)
		}
	}
}
