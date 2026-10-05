package profile

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestEnableTurnsOnRequiredDisabledDependencies(t *testing.T) {
	e := newEnv(t)
	e.item(t, "core", map[string]string{"manifest.json": `{"Name":"Core","Author":"me","Version":"1.0.0","UniqueID":"Me.Core"}`})
	e.item(t, "user", map[string]string{"manifest.json": `{"Name":"User","Author":"me","Version":"1.0.0","UniqueID":"Me.User","Dependencies":[{"UniqueID":"Me.Core"},{"UniqueID":"Me.Opt","IsRequired":false}]}`})
	e.item(t, "opt", map[string]string{"manifest.json": `{"Name":"Opt","Author":"me","Version":"1.0.0","UniqueID":"Me.Opt"}`})
	p := mustCreate(t, e, "Farm")
	if _, err := e.AddEntry("stardew", p.ID, "core", Source{Kind: KindLocal, Name: "core.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "opt", Source{Kind: KindLocal, Name: "opt.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "user", Source{Kind: KindLocal, Name: "user.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "", "Me.Core", false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "", "Me.Opt", false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "", "Me.User", false); err != nil {
		t.Fatal(err)
	}
	got, also, err := e.enableMod("stardew", p.ID, "", "Me.User", true)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(also, []string{"Core"}) {
		t.Fatalf("also = %v", also)
	}
	on := map[string]bool{}
	for _, en := range got.Entries {
		for _, m := range en.Mods {
			on[m.UniqueID] = !hasID(en.Disabled, m.UniqueID)
		}
	}
	if !on["Me.User"] || !on["Me.Core"] || on["Me.Opt"] {
		t.Fatalf("enabled = %+v disabled entries=%+v", on, got.Entries)
	}
}

func TestEnableSkipsRequiredWhenSettingIsNever(t *testing.T) {
	e := newEnv(t)
	st, err := settings.Open()
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
	if _, err := e.AddEntry("stardew", p.ID, "core", Source{Kind: KindLocal, Name: "core.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "user", Source{Kind: KindLocal, Name: "user.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "", "Me.Core", false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "", "Me.User", false); err != nil {
		t.Fatal(err)
	}
	got, also, err := e.enableMod("stardew", p.ID, "", "Me.User", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(also) != 0 {
		t.Fatalf("also = %v", also)
	}
	on := map[string]bool{}
	for _, en := range got.Entries {
		for _, m := range en.Mods {
			on[m.UniqueID] = !hasID(en.Disabled, m.UniqueID)
		}
	}
	if !on["Me.User"] || on["Me.Core"] {
		t.Fatalf("enabled = %+v", on)
	}
}
