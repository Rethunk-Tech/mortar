package cli

import "testing"

func TestModsMenuArgs(t *testing.T) {
	t.Parallel()
	game, profile, mod, set, err := modsMenuArgs([]string{"stardewvalley", "Default", "Example.Mod", "--set", "main/0=true"})
	if err != nil {
		t.Fatal(err)
	}
	if game != "stardewvalley" || profile != "Default" || mod != "Example.Mod" || set != "main/0=true" {
		t.Fatalf("%s %s %s %s", game, profile, mod, set)
	}
	if _, _, _, _, err := modsMenuArgs([]string{"stardewvalley"}); err == nil {
		t.Fatal("expected usage error")
	}
}
