package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/problems"
)

func TestUpdatesChangelogPrintsEntriesUnderEachMod(t *testing.T) {
	results := map[string]any{
		"updates": problems.UpdatesResult{
			Updates: []problems.Update{
				{Name: "Content Patcher", Installed: "1.0.0", Version: "2.0.0", Source: "Nexus", NexusID: 1915, URL: "https://nexus"},
				{Name: "Local", Installed: "1.0.0", Version: "1.1.0", Source: "local"},
				{Name: "Gh Mod", Installed: "1.0.0", Version: "1.2.0", Source: "GitHub", GitHubRepo: "me/gh"},
			},
		},
		"changelog": []nexus.Changelog{
			{Version: "2.0.0", Notes: []string{"new API"}},
			{Version: "1.5.0", Body: "release body line"},
		},
	}
	r := invoke(t, results, "updates", "stardew", "Farm", "--changelog")
	if r.code != 0 {
		t.Fatalf("code %d stderr %q", r.code, r.errOut)
	}
	if len(r.calls) != 3 || r.calls[0].method != "updates" || r.calls[1].method != "changelog" {
		t.Fatalf("calls %+v", r.calls)
	}
	if c := r.calls[1]; c.params.Game != "stardew" || c.params.Source != "nexus" || c.params.ID != "1915" || c.params.Name != "1.0.0" || c.params.Value != "2.0.0" {
		t.Fatalf("changelog params %+v", r.calls[1].params)
	}
	if c := r.calls[2]; c.params.Source != "github" || c.params.ID != "me/gh" {
		t.Fatalf("github changelog params %+v", c.params)
	}
	if !strings.Contains(r.out, "Gh Mod") || !strings.Contains(r.out, "release body line") {
		t.Fatalf("out %q", r.out)
	}
	if !strings.Contains(r.out, "Content Patcher") || !strings.Contains(r.out, "2.0.0") || !strings.Contains(r.out, "new API") {
		t.Fatalf("out %q", r.out)
	}
}
