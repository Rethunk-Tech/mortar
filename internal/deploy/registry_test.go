package deploy

import "testing"

type fakeDeployer struct {
	Deployer
	id string
}

func (f fakeDeployer) ID() string { return f.id }

func TestRegistryContract(t *testing.T) {
	Register(fakeDeployer{id: "contract-a"})
	for id, want := range map[string]bool{"contract-a": true, "contract-none": false, "": false} {
		if _, ok := Get(id); ok != want {
			t.Errorf("Get(%q) = %v, want %v", id, ok, want)
		}
	}
}
