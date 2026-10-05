package mod

import "testing"

func TestIDParts(t *testing.T) {
	id := NewID(FormatSMAPI, "Pathoschild.ContentPatcher")
	if id != "smapi:Pathoschild.ContentPatcher" || id.Format() != "smapi" || id.Local() != "Pathoschild.ContentPatcher" {
		t.Fatalf("got %q %q %q", id, id.Format(), id.Local())
	}
	tc := NewID(FormatThunderstore, "BepInEx-BepInExPack:5.4")
	if tc.Format() != "thunderstore" || tc.Local() != "BepInEx-BepInExPack:5.4" {
		t.Errorf("only the first colon splits: %q %q", tc.Format(), tc.Local())
	}
	if bare := ID("plain"); bare.Format() != "" || bare.Local() != "plain" {
		t.Errorf("no colon: %q %q", bare.Format(), bare.Local())
	}
}

func TestEqualRules(t *testing.T) {
	cases := []struct {
		a, b ID
		want bool
	}{
		{"smapi:Au.One", "smapi:au.one", true},
		{"smapi: Au.One ", "smapi:au.one", true},
		{"smapi:au.one", "smapi:au.two", false},
		{"smapi:au.one", "bepinex:au.one", false},
		{"bepinex:Au.One", "bepinex:au.one", false},
		{"", "", true},
	}
	for _, c := range cases {
		if got := Equal(c.a, c.b); got != c.want {
			t.Errorf("Equal(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
