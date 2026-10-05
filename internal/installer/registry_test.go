package installer

import "testing"

type fakeInstaller struct {
	Installer
	id string
}

func (f fakeInstaller) ID() string { return f.id }

func TestRegistryContract(t *testing.T) {
	Register(fakeInstaller{id: "contract-a"})
	for id, want := range map[string]bool{"contract-a": true, "contract-none": false, "": false} {
		if _, ok := Get(id); ok != want {
			t.Errorf("Get(%q) = %v, want %v", id, ok, want)
		}
	}
	for _, id := range detectionOrder {
		if _, ok := Get(id); !ok {
			t.Errorf("the detection order names %q, which has no driver", id)
		}
	}
}
