package stardew

import "testing"

func TestStardewVersionFromLog(t *testing.T) {
	t.Parallel()
	const header = "SMAPI 4.5.2 with Stardew Valley 1.6.15 build 24356 on Unix 6.16.8-200.fc42.x86_64"
	if got := StardewVersionFromLog(header + "\n[14:00:00 INFO SMAPI] Mods go here: /tmp/mods\n"); got != "1.6.15" {
		t.Fatalf("got %q", got)
	}
}

func TestGameVersionChanged(t *testing.T) {
	t.Parallel()
	if GameVersionChanged("", "1.6.15") {
		t.Fatal("empty recorded is not a change")
	}
	if GameVersionChanged("1.6.15", "") {
		t.Fatal("empty installed is not a change")
	}
	if GameVersionChanged("1.6.15", "1.6.15") {
		t.Fatal("same version")
	}
	if !GameVersionChanged("1.6.14", "1.6.15") {
		t.Fatal("want change")
	}
}
