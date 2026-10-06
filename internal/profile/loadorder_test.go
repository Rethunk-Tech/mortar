package profile

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestLoadOrderNeverReportsAnOptionalDependencyMissing(t *testing.T) {
	smapi := func(id string) mod.ID { return mod.NewID(mod.FormatSMAPI, id) }
	lib, gmcm := smapi("SMAPI.Lib"), smapi("spacechase0.GenericModConfigMenu")
	rows := loadorder.Resolve(loadOrderInput([]Mod{
		{
			ID: smapi("BetterRanching"), Name: "Better Ranching", Enabled: true,
			Needs: []mod.ID{lib, gmcm}, Optional: []mod.ID{smapi("spacechase0.genericmodconfigmenu")},
		},
		{ID: smapi("Off"), Name: "Off", Enabled: false},
	}))
	if len(rows) != 1 {
		t.Fatalf("rows = %+v", rows)
	}
	r := rows[0]
	if len(r.MissingRequired) != 1 || r.MissingRequired[0] != lib {
		t.Fatalf("missing = %v", r.MissingRequired)
	}
	if len(r.Optional) != 1 || len(r.Required) != 1 {
		t.Fatalf("required = %v, optional = %v", r.Required, r.Optional)
	}
}
