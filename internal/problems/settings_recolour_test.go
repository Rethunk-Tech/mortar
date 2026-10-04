package problems

import "testing"

func earthy(enabled bool) Installed {
	return Installed{Key: "earthy", Enabled: enabled, UniqueID: "DaisyNiko.EarthyRecolour", Name: "Earthy Recolour"}
}

const (
	foliageSchema  = `{"Foliage":{"Default":"Vanilla","AllowValues":"Vanilla, Earthy, VPR, Starblue"}}`
	foliageChanges = `[{"Action":"Load","Target":"a","FromFile":"assets/{{Foliage}}.png"}]`
)

func TestRecolourChoiceFollowsTheEnabledRecolour(t *testing.T) {
	pack := settingPack(t, foliageSchema, foliageChanges, `{"Foliage":"Vanilla"}`)
	hints := compatibilitySettings([]Installed{pack, earthy(true)})
	if len(hints) != 1 || hints[0].Suggested[0] != "Earthy" || hints[0].ForNames[0] != "Earthy Recolour" || !hints[0].Variant {
		t.Fatalf("hints = %#v", hints)
	}
	if got := compatibilitySettings([]Installed{pack, earthy(false)}); len(got) != 0 {
		t.Fatalf("disabled recolour suggested: %#v", got)
	}
	matched := settingPack(t, foliageSchema, foliageChanges, `{"Foliage":"Earthy"}`)
	if got := compatibilitySettings([]Installed{matched, earthy(true)}); len(got) != 0 {
		t.Fatalf("matching choice flagged: %#v", got)
	}
}

func TestRecolourChoiceForAMissingRecolourFallsBackToVanilla(t *testing.T) {
	pack := settingPack(t, foliageSchema, foliageChanges, `{"Foliage":"Starblue"}`)
	hints := compatibilitySettings([]Installed{pack})
	if len(hints) != 1 || hints[0].Suggested[0] != "Vanilla" || hints[0].CurrentFor != "Starblue Valley" {
		t.Fatalf("hints = %#v", hints)
	}
}

func TestRecolourChoiceLeftToThePack(t *testing.T) {
	auto := settingPack(t, `{"Style":{"Default":"Auto","AllowValues":"Auto, Vanilla, Earthy"}}`, foliageChanges, "")
	mapped := settingPack(t, foliageSchema, `[{"Action":"Load","Target":"a","When":{"Foliage":"Earthy","HasMod":"DaisyNiko.EarthyRecolour"}}]`, `{"Foliage":"Vanilla"}`)
	for _, pack := range []Installed{auto, mapped} {
		if got := compatibilitySettings([]Installed{pack, earthy(true)}); len(got) != 0 {
			t.Fatalf("choice the pack makes itself flagged: %#v", got)
		}
	}
}
