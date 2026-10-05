package stardew

import "testing"

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
