package launchsvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestPickCulpritTakesFirstFindingOfAnEnabledMod(t *testing.T) {
	mods := []profile.Mod{
		{Key: "a", Name: "LethalLib", ID: "EvaisaDev-LethalLib", Enabled: false},
		{Key: "b", Name: "More Company", ID: "notnotnotswipez-MoreCompany", Enabled: true},
		{Key: "c", Name: "Hydrogen", ID: "Owner-Hydrogen", Enabled: true},
	}
	found := []loader.Finding{
		{Plugin: "BepInEx", Message: "framework"},
		{Plugin: "LethalLib", Message: "disabled mods are not blamed"},
		{Plugin: "me.swipez.melonloader.morecompany", Message: "load failed"},
		{Plugin: "Hydrogen", Message: "later"},
	}
	m, f, ok := pickCulprit(found, mods)
	if !ok || m.Key != "b" || f.Message != "load failed" {
		t.Fatalf("culprit = %+v, %+v, %v", m, f, ok)
	}
	if _, _, ok := pickCulprit([]loader.Finding{{Plugin: "Unknown"}, {Plugin: "ab"}}, mods); ok {
		t.Fatal("a plugin that is no installed mod blames nothing")
	}
}

func TestPickCulpritShortTokenDoesNotMatchLongerMod(t *testing.T) {
	mods := []profile.Mod{
		{Key: "a", Name: "LethalLib", ID: "EvaisaDev-LethalLib", Enabled: true},
		{Key: "b", Name: "Lethal Lib Extended", ID: "Owner-LethalLibExtended", Enabled: true},
	}
	if m, _, ok := pickCulprit([]loader.Finding{{Plugin: "Lib"}}, mods); ok {
		t.Fatalf("plugin Lib blamed %q", m.Key)
	}
	m, _, ok := pickCulprit([]loader.Finding{{Plugin: "evaisa.lethallib"}}, mods)
	if !ok || m.Key != "a" {
		t.Fatalf("culprit = %+v, %v", m, ok)
	}
}
