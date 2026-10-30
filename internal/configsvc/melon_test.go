package configsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-cfgedit"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// The layout is what MelonPreferences_Entry.Save writes through Tomlet: the description as a preceding comment, the
// optional inline comment after the value.
const melonSample = `[MelonLoader]
# Mod load order
load_order = "default" # inline

[BetterHands]
# Turns the feature on
Enabled = true
# Multiplier
Speed = 2.5
Count = 4
Names = ["a", "b"]
Label = "Hello # not a comment"
`

func TestMelonCategoryOwnership(t *testing.T) {
	if !ownsMelonCategory([]string{"thunderstore:Ada-Better_Hands"}, "BetterHands") || ownsMelonCategory([]string{"thunderstore:Ada-Other"}, "BetterHands") {
		t.Fatal("category ownership")
	}
}

func TestMelonRefusedValueNamesItsProblem(t *testing.T) {
	_, err := cfgedit.ParseMelon("[A]\nT = { a = 1 }\n").Set("A", "T", "{ x = ")
	if detail, _ := usererr.DetailOf(valueErr(err)).(*valueError); detail == nil || detail.Problem != "toml" || usererr.KindOf(valueErr(err)) != usererr.Invalid {
		t.Errorf("raw: %v", err)
	}
}

func TestMelonSchemaKeepsOnlyTheModsCategories(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "UserData"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "UserData", melonName), []byte(melonSample), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Service{Profiles: fakeProfiles{dir: dir}}
	f := ConfigFile{Name: melonName, Format: FormatMelon}
	mine, err := s.melonSchema("g", "p", "thunderstore:Ada-Better_Hands", f)
	if err != nil || len(mine.Sections) != 1 || mine.Sections[0].Name != "BetterHands" {
		t.Fatalf("one mod: %+v %v", mine.Sections, err)
	}
	all, err := s.melonSchema("g", "p", "", f)
	if err != nil || len(all.Sections) != 2 {
		t.Fatalf("whole profile: %+v %v", all.Sections, err)
	}
}
