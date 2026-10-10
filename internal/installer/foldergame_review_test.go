package installer

import "testing"

func TestFolderGameLayoutRoutesTrayFilesAndKeepsTheDepthCap(t *testing.T) {
	t.Parallel()
	g := Game{Loaders: []string{"folder"}, Targets: []Target{
		{ID: TargetMods, MaxDepth: map[string]int{"package": 2}},
		{ID: "tray", Shared: true, Extensions: []string{"trayitem"}},
	}}
	l, err := (plain{}).Layout(extracted(t, "k", map[string]string{"a/b.package": "x", "House.TRAYITEM": "y"}), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range l.Files {
		got[f.Rel] = f.Target
	}
	if got["a/b.package"] != TargetMods || got["House.TRAYITEM"] != "tray" || !g.IsShared("tray") || g.IsShared(TargetMods) {
		t.Fatalf("layout = %v", got)
	}
	if _, err := (plain{}).Layout(extracted(t, "k", map[string]string{"a/b/c/d.package": "x"}), g, nil); err == nil {
		t.Fatal("a package deeper than the target allows was accepted")
	}
}
