package control

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestTypedIDNamesAModOfAnyFormatByItsBareID(t *testing.T) {
	lethal := []mod.ID{"thunderstore:AinaVT-LethalConfig", "thunderstore:Rune580-LethalCompany_InputUtils", "smapi:Pathoschild.ContentPatcher"}
	cases := []struct {
		known []mod.ID
		typed string
		want  mod.ID
	}{
		{lethal, "AinaVT-LethalConfig", "thunderstore:AinaVT-LethalConfig"},
		// SMAPI ids compare without case, as SMAPI does.
		{lethal, " pathoschild.contentpatcher ", "smapi:Pathoschild.ContentPatcher"},
		{lethal, "thunderstore:Someone-Else", "thunderstore:Someone-Else"},
		// Not installed: the format most of the profile's mods use.
		{lethal, "Someone-Else", "thunderstore:Someone-Else"},
		{nil, "Esca.FarmTypeManager", "smapi:Esca.FarmTypeManager"},
	}
	for _, c := range cases {
		if got, err := typedID(c.known, c.typed); err != nil || got != c.want {
			t.Errorf("typedID(%q) = %q, want %q", c.typed, got, c.want)
		}
	}
	both := []mod.ID{"folder:examplemod", "modinfo:examplemod"}
	if _, err := typedID(both, "examplemod"); err == nil || !strings.Contains(err.Error(), "ambiguous id examplemod") {
		t.Errorf("bare id of two formats: err = %v", err)
	}
	if got, err := typedID(both, "modinfo:examplemod"); err != nil || got != "modinfo:examplemod" {
		t.Errorf("full id = %q, %v", got, err)
	}
	if idMatches("ainavt-lethalconfig", "thunderstore:AinaVT-LethalConfig") {
		t.Error("a Thunderstore id matched in another case")
	}
}
