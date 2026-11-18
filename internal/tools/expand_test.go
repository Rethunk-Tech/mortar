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

func TestEmptyPlaceholderAndArgListStayEmpty(t *testing.T) {
	ctx := Context{Game: "/g"}
	if got := expand("", ctx); got != "" {
		t.Fatalf("empty string = %q", got)
	}
	if got := expandArgs(nil, ctx); got != nil {
		t.Fatalf("nil args = %#v", got)
	}
	if got := expandArgs([]string{}, ctx); got != nil {
		t.Fatalf("empty args = %#v", got)
	}
}
