package tools

import "testing"

func TestExpandPlaceholders(t *testing.T) {
	ctx := Context{Game: "/g", Mods: "/m", Saves: "/s", Profile: "/p"}
	if expand("{game}/x", ctx) != "/g/x" {
		t.Fatalf("game placeholder")
	}
	if expandArgs([]string{"{mods}", "{profile}"}, ctx)[1] != "/p" {
		t.Fatalf("args")
	}
	if expand("{saves}", ctx) != "/s" {
		t.Fatalf("saves")
	}
}
