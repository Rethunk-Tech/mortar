package profile

import (
	"strings"
	"testing"
)

func TestSetLaunchOptions(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.SetLaunchOptions("stardew", p.ID, `--developer-mode "one two"`)
	if err != nil || got.LaunchOptions != `--developer-mode "one two"` {
		t.Fatalf("set = %+v, %v", got, err)
	}
	listed, err := s.List("stardew")
	if err != nil || listed[0].LaunchOptions != `--developer-mode "one two"` {
		t.Fatalf("list = %+v, %v", listed, err)
	}
}

func TestSetLaunchOptionsDeniesMortarFlags(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetLaunchOptions("stardew", p.ID, "--mods-path /tmp"); err == nil || !strings.Contains(err.Error(), "is set by Mortar") {
		t.Fatalf("err = %v", err)
	}
}
