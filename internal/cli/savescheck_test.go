package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/savessvc"
)

func TestSavesCheckAndProfileFromSave(t *testing.T) {
	results := map[string]any{
		"saves.check": savessvc.SaveCheck{
			Folder: "Farm_1", Farm: "Sunny", LastProfileID: "p1", LastProfileExists: true,
			Missing: []savessvc.Lack{{UniqueID: "A.Mod", Name: "Alpha"}, {UniqueID: "B.Mod", Name: "Beta"}},
		},
		"profile.fromSave": savessvc.FromSaveResult{
			Profile: profile.Profile{ID: "new1", Name: "Sunny"},
			Added:   []string{"local-a"},
			Queued:  []string{"B.Mod"},
		},
	}
	r := invoke(t, results, "saves", "check", "stardew", "Farm_1", "Main")
	if r.code != 0 || r.calls[0].method != "saves.check" || r.calls[0].params.Name != "Farm_1" || r.calls[0].params.Profile != "Main" {
		t.Fatalf("check call: %+v", r)
	}
	if !strings.Contains(r.out, "2 mods") || !strings.Contains(r.out, "Alpha") {
		t.Fatalf("check out: %q", r.out)
	}
	r = invoke(t, results, "profile", "from-save", "stardew", "Farm_1")
	if r.code != 0 || r.calls[0].method != "profile.fromSave" || r.calls[0].params.Name != "Farm_1" {
		t.Fatalf("from-save call: %+v", r)
	}
	if !strings.Contains(r.out, "new1") || !strings.Contains(r.out, "Sunny") {
		t.Fatalf("from-save out: %q", r.out)
	}
}
