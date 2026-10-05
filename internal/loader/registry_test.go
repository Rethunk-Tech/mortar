package loader

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

type fakeLoader struct {
	Loader
	id string
}

func (f fakeLoader) ID() string { return f.id }

func TestRegistryContract(t *testing.T) {
	Register(fakeLoader{id: "contract-a"})
	Register(fakeLoader{id: "contract-b"})
	for id, want := range map[string]bool{"contract-a": true, "CONTRACT-B": true, "contract-none": false} {
		if _, ok := Get(id); ok != want {
			t.Errorf("Get(%q) = %v, want %v", id, ok, want)
		}
	}
	g := components.GameInfo{Loaders: []components.GameLoader{{ID: "contract-b"}, {ID: "contract-none"}, {ID: "contract-a"}}}
	got := For(g)
	if len(got) != 2 || got[0].ID() != "contract-b" || got[1].ID() != "contract-a" {
		t.Errorf("For lists catalog order and skips a loader with no driver: %v", got)
	}
}
