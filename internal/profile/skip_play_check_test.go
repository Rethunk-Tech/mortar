package profile

import (
	"testing"
)

func TestSkipPlayCheckDefaultsFalseAndPersists(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if p.Overrides["skipPlayCheck"] != "" {
		t.Fatal("new profile skips the pre-Play check")
	}

	got, err := s.SetSkipPlayCheck("stardew", p.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Overrides["skipPlayCheck"] != "true" {
		t.Fatal("setter did not skip the pre-Play check")
	}

	listed, err := s.List("stardew")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Overrides["skipPlayCheck"] != "true" {
		t.Fatalf("listed profile = %+v", listed)
	}

	got, err = s.SetSkipPlayCheck("stardew", p.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Overrides["skipPlayCheck"] != "false" {
		t.Fatal("setter did not restore the pre-Play check")
	}
}
