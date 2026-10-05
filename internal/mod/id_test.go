package mod

import "testing"

const bare = "plain"

func TestIDTextFormAndEquality(t *testing.T) {
	if id := NewID(FormatSMAPI, "Pathoschild.ContentPatcher"); id != "smapi:Pathoschild.ContentPatcher" {
		t.Errorf("NewID = %q", id)
	}
	for _, c := range []struct {
		id            ID
		format, local string
	}{
		{"smapi:Pathoschild.ContentPatcher", "smapi", "Pathoschild.ContentPatcher"},
		{"thunderstore:BepInEx-BepInExPack:5.4", "thunderstore", "BepInEx-BepInExPack:5.4"},
		{ID(bare), "", "plain"},
	} {
		if c.id.Format() != c.format || c.id.Local() != c.local {
			t.Errorf("%q splits to %q, %q", c.id, c.id.Format(), c.id.Local())
		}
	}
	for _, c := range []struct {
		a, b ID
		want bool
	}{
		{"smapi:Au.One", "smapi:au.one", true},
		{"smapi: Au.One ", "smapi:au.one", true},
		{"smapi:au.one", "smapi:au.two", false},
		{"smapi:au.one", "bepinex:au.one", false},
		{"bepinex:Au.One", "bepinex:au.one", false},
		{"", "", true},
	} {
		if got := Equal(c.a, c.b); got != c.want {
			t.Errorf("Equal(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
	if Parse("Au.One", FormatSMAPI) != "smapi:Au.One" || Parse("bepinex:Au.One", FormatSMAPI) != "bepinex:Au.One" {
		t.Error("Parse must default only bare ids")
	}
}
