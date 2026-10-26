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
