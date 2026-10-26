package devsvc

import "testing"

func TestArgsRefuseControlCharacters(t *testing.T) {
	f := Form{all: []Param{{ID: "description", Label: "Description"}}}
	for _, v := range []string{"a\tb", "a\x1fb", "a\nb"} {
		if _, _, err := f.args(map[string]string{"description": v}, nil); err == nil {
			t.Fatalf("%q was accepted", v)
		}
	}
}

func TestArgsRefuseAnUnmetRequirement(t *testing.T) {
	f := Form{all: []Param{
		{
			ID: "audience", Label: "Audience", Kind: "choice", Default: "everyone",
			Choices: []Choice{{Value: "everyone", Label: "Everyone"}, {Value: "host"}},
		},
		{ID: "netcode", Label: "Netcode", Kind: "bool", Requires: map[string]string{"audience": "everyone"}},
	}}
	cases := []struct {
		values map[string]string
		ok     bool
	}{
		{map[string]string{"netcode": "true"}, true},
		{map[string]string{"netcode": "true", "audience": "everyone"}, true},
		{map[string]string{"netcode": "false", "audience": "host"}, true},
		{map[string]string{"netcode": "true", "audience": "host"}, false},
	}
	for _, c := range cases {
		if _, _, err := f.args(c.values, nil); (err == nil) != c.ok {
			t.Errorf("%v: err %v, want ok %v", c.values, err, c.ok)
		}
	}
}
