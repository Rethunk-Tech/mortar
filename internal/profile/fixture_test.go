package profile

import "testing"

type profileCreator interface {
	Create(game, name string) (Profile, error)
}

func mustCreate(t testing.TB, s profileCreator, name string) Profile {
	t.Helper()
	p, err := s.Create("stardew", name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
