package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/datasvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
	"github.com/Rethunk-Tech/mortar/internal/templates"
)

func TestWaveCommandsSendTheirArguments(t *testing.T) {
	results := map[string]any{
		"templates":                 []templates.Template{{Name: "Base"}},
		"templates.save":            templates.Template{Name: "Base"},
		"history.usage":             []profile.HistoryUsage{{ProfileID: "p1", ProfileName: "Farm", Events: 5}},
		"history.trim":              profile.HistoryUsage{ProfileName: "Farm", Events: 2},
		"backups.usage":             savessvc.BackupsUsage{TotalBytes: 10, PerSave: []savessvc.SaveUsage{{Save: "Farm_1", Count: 2}}},
		"backups.trim":              savessvc.TrimResult{Removed: 3},
		"library.old-files":         []profile.OldFiles{{Key: "k", Label: "Mod"}},
		"library.old-files.resolve": nil,
		"library.strays":            profile.GameModsPreview{Mods: []profile.GameModPreview{{Name: "Stray", Folder: "/g/Mods/Stray"}}},
		"library.strays.move":       profile.GameModsResult{Imported: 1},
		"queue.retry-failed":        queue.RetryAllResult{Requeued: 2},
		"data.location":             datasvc.DataLocation{Dir: "/d", Portable: true},
		"archive.preview":           archive.Preview{Manifests: []archive.PreviewManifest{{ID: "smapi:A.B", Name: "Ab"}}},
	}
	cases := []struct {
		args   []string
		method string
		check  func(control.Params) bool
		out    string
	}{
		{[]string{"templates", "list", "stardew"}, "templates", func(p control.Params) bool { return p.Game == "stardew" }, "Base"},
		{[]string{"templates", "save", "stardew", "Farm", "My", "Base"}, "templates.save", func(p control.Params) bool { return p.Profile == "Farm" && p.Name == "My Base" }, "Saved template"},
		{[]string{"history", "usage", "stardew"}, "history.usage", func(p control.Params) bool { return p.Game == "stardew" }, "Farm"},
		{[]string{"history", "trim", "stardew", "Farm", "--keep", "2"}, "history.trim", func(p control.Params) bool { return p.Keep == 2 && p.Profile == "Farm" }, "keeps 2"},
		{[]string{"backups", "usage"}, "backups.usage", func(p control.Params) bool { return true }, "Farm_1"},
		{[]string{"backups", "trim", "--keep", "1"}, "backups.trim", func(p control.Params) bool { return p.Keep == 1 }, "Deleted 3"},
		{[]string{"library", "old-files", "stardew", "Farm"}, "library.old-files", func(p control.Params) bool { return p.Profile == "Farm" }, "Mod"},
		{[]string{"library", "old-files", "stardew", "Farm", "--delete", "k"}, "library.old-files.resolve", func(p control.Params) bool { return p.Key == "k" && !p.Set }, "Deleted"},
		{[]string{"library", "strays", "stardew"}, "library.strays", func(p control.Params) bool { return p.Game == "stardew" }, "Stray"},
		{[]string{"library", "strays", "stardew", "Farm", "--move", "A", "--move", "B"}, "library.strays.move", func(p control.Params) bool { return len(p.IDs) == 2 && p.Profile == "Farm" }, "Moved 1"},
		{[]string{"queue", "retry-failed"}, "queue.retry-failed", func(p control.Params) bool { return true }, "Requeued 2"},
		{[]string{"data", "location"}, "data.location", func(p control.Params) bool { return true }, "Portable"},
		{[]string{"archive", "preview", "/tmp/a.zip"}, "archive.preview", func(p control.Params) bool { return p.Path == "/tmp/a.zip" }, "A.B"},
	}
	for _, tc := range cases {
		r := invoke(t, results, tc.args...)
		if r.code != 0 || r.calls[len(r.calls)-1].method != tc.method || !tc.check(r.calls[len(r.calls)-1].params) || !strings.Contains(r.out, tc.out) {
			t.Errorf("%v: %+v", tc.args, r)
		}
	}
	if r := invoke(t, results, "history", "trim", "stardew", "Farm"); r.code != 2 || len(r.calls) != 0 {
		t.Errorf("trim without --keep: %+v", r)
	}
	r := invoke(t, results, "backups", "usage", "--json")
	if r.code != 0 || !strings.Contains(r.out, `"totalBytes": 10`) {
		t.Errorf("--json: %+v", r)
	}
}
