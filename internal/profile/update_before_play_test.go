package profile

import (
	"testing"
)

func TestUpdateBeforePlayDefaultsFalseAndPersists(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if p.UpdateBeforePlay {
		t.Fatal("new profile enables update before Play")
	}

	got, err := s.SetUpdateBeforePlay("stardew", p.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.UpdateBeforePlay {
		t.Fatal("setter did not enable update before Play")
	}

	listed, err := s.List("stardew")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || !listed[0].UpdateBeforePlay {
		t.Fatalf("listed profile = %+v", listed)
	}

	got, err = s.SetUpdateBeforePlay("stardew", p.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.UpdateBeforePlay {
		t.Fatal("setter did not disable update before Play")
	}
}
