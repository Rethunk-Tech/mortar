package framework

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

type fakeFramework struct{ id mod.ID }

func (f fakeFramework) ID() mod.ID           { return f.id }
func (f fakeFramework) Matches(m Mod) bool   { return m.Key == string(f.id) }
func (fakeFramework) Analyze(Input) Findings { return Findings{} }

func TestRegistryContract(t *testing.T) {
	Register(fakeFramework{id: "smapi:Contract.A"})
	present := func(ids ...mod.ID) []Framework {
		mods := make([]Mod, len(ids))
		for i, id := range ids {
			mods[i] = Mod{Key: string(id)}
		}
		return Present(mods)
	}
	if got := present("smapi:Contract.A"); len(got) != 1 || got[0].ID() != "smapi:Contract.A" {
		t.Errorf("a framework is present when an enabled mod matches it: %v", got)
	}
	if got := present("smapi:Contract.None"); len(got) != 0 {
		t.Errorf("an unknown mod turns no framework on: %v", got)
	}
}
