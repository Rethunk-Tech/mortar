package profile

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestEnableTurnsOnRequiredDisabledDependencies(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "core", map[string]string{"manifest.json": `{"Name":"Core","Author":"me","Version":"1.0.0","UniqueID":"Me.Core"}`})
	e.item(t, "user", map[string]string{"manifest.json": `{"Name":"User","Author":"me","Version":"1.0.0","UniqueID":"Me.User","Dependencies":[{"UniqueID":"Me.Core"},{"UniqueID":"Me.Opt","IsRequired":false}]}`})
	e.item(t, "opt", map[string]string{"manifest.json": `{"Name":"Opt","Author":"me","Version":"1.0.0","UniqueID":"Me.Opt"}`})
	p := mustCreate(t, e, "Farm")
	e.addZip(t, p.ID, "core")
	e.addZip(t, p.ID, "opt")
	e.addZip(t, p.ID, "user")
	e.disable(t, p.ID, "smapi:Me.Core")
	e.disable(t, p.ID, "smapi:Me.Opt")
	e.disable(t, p.ID, "smapi:Me.User")
	got, also, err := e.enableMod("stardew", p.ID, "", "smapi:Me.User", true)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(also, []string{"Core"}) {
		t.Fatalf("also = %v", also)
	}
	on := enabledByID(got)
	if !on["Me.User"] || !on["Me.Core"] || on["Me.Opt"] {
		t.Fatalf("enabled = %+v disabled entries=%+v", on, got.Entries)
	}
}

func TestEnableSkipsRequiredWhenSettingIsNever(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	st, err := settings.OpenIn(e.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	var applyErr error
	if _, err := st.Update(func(s *settings.Settings) {
		applyErr = settings.ApplyKeyGame(s, "enableRequirements", settings.EnableReqNever, "stardew")
	}); err != nil {
		t.Fatal(err)
	}
	if applyErr != nil {
		t.Fatal(applyErr)
	}
	e.settings = st
	e.item(t, "core", map[string]string{"manifest.json": `{"Name":"Core","Author":"me","Version":"1.0.0","UniqueID":"Me.Core"}`})
	e.item(t, "user", map[string]string{"manifest.json": `{"Name":"User","Author":"me","Version":"1.0.0","UniqueID":"Me.User","Dependencies":[{"UniqueID":"Me.Core"}]}`})
	p := mustCreate(t, e, "Farm")
	e.addZip(t, p.ID, "core")
	e.addZip(t, p.ID, "user")
	e.disable(t, p.ID, "smapi:Me.Core")
	e.disable(t, p.ID, "smapi:Me.User")
	got, also, err := e.enableMod("stardew", p.ID, "", "smapi:Me.User", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(also) != 0 {
		t.Fatalf("also = %v", also)
	}
	on := enabledByID(got)
	if !on["Me.User"] || on["Me.Core"] {
		t.Fatalf("enabled = %+v", on)
	}
}

// enabledByID says, by local id, whether each mod of p is on.
func enabledByID(p Profile) map[string]bool {
	on := map[string]bool{}
	for _, en := range p.Entries {
		for _, m := range en.Mods {
			on[m.ID.Local()] = !hasID(en.Disabled, m.ID)
		}
	}
	return on
}
