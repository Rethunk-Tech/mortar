package contentpatcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/testenv/packs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestCompatibilitySettingSuggestionWhenOff(t *testing.T) {
	pack := settingPack(t, `{"Enabled":{"Default":false}}`, `[{"Action":"Load","Target":"a","When":{"Enabled":true,"HasMod":"Other.Mod"}}]`, `{"Enabled":false}`)
	hints := compatibilitySettings([]framework.Mod{pack, settingMod()})
	if len(hints) != 1 || hints[0].Field != "Enabled" || len(hints[0].Suggested) != 1 || hints[0].Suggested[0] != "true" {
		t.Fatalf("hints = %#v", hints)
	}
}

func TestCompatibilitySettingAlreadyActiveIsNotSuggested(t *testing.T) {
	pack := settingPack(t, `{"Enabled":{"Default":false}}`, `[{"Action":"Load","Target":"a","When":{"Enabled":true,"HasMod":"Other.Mod"}}]`, `{"Enabled":true}`)
	if got := compatibilitySettings([]framework.Mod{pack, settingMod()}); len(got) != 0 {
		t.Fatalf("hints = %#v", got)
	}
}

func TestCompatibilitySettingMultipleValuesOneActiveIsNotSuggested(t *testing.T) {
	pack := settingPack(t, `{"Modes":{"Default":"off","AllowMultiple":true}}`, `[{"Action":"Load","Target":"a","When":{"Modes":"high, low","HasMod":"Other.Mod"}}]`, `{"Modes":"off, low"}`)
	if got := compatibilitySettings([]framework.Mod{pack, settingMod()}); len(got) != 0 {
		t.Fatalf("hints = %#v", got)
	}
}

func TestCompatibilitySettingUnion(t *testing.T) {
	pack := settingPack(t, `{"Mode":{"Default":"off"}}`, `[{"Action":"Load","Target":"a","When":{"Mode":"high","HasMod":"Other.Mod"}},{"Action":"Load","Target":"b","When":{"Mode":"low","HasMod":"Other.Mod"}}]`, `{"Mode":"off"}`)
	hints := compatibilitySettings([]framework.Mod{pack, settingMod()})
	if len(hints) != 1 || len(hints[0].Suggested) != 2 || hints[0].Suggested[0] != "high" || hints[0].Suggested[1] != "low" {
		t.Fatalf("hints = %#v", hints)
	}
}

func TestCompatibilitySettingMissingConfigUsesDefault(t *testing.T) {
	pack := settingPack(t, `{"Enabled":{"Default":false,"Description":"Use the compatibility patch"}}`, `[{"Action":"Load","Target":"a","When":{"Enabled":true,"HasMod":"Other.Mod"}}]`, "")
	hints := compatibilitySettings([]framework.Mod{pack, settingMod()})
	if len(hints) != 1 || hints[0].Current != "false" || hints[0].Description != "Use the compatibility patch" {
		t.Fatalf("hints = %#v", hints)
	}
}

func TestCompatibilitySettingFalseHasModIsIgnored(t *testing.T) {
	pack := settingPack(t, `{"Enabled":{"Default":false}}`, `[{"Action":"Load","Target":"a","When":{"Enabled":true,"HasMod |contains=Other.Mod":false}}]`, `{"Enabled":false}`)
	if got := compatibilitySettings([]framework.Mod{pack, settingMod()}); len(got) != 0 {
		t.Fatalf("hints = %#v", got)
	}
}

func settingMod() framework.Mod {
	return framework.Mod{Enabled: true, UniqueID: "Other.Mod", Name: "Other"}
}

func settingPack(t *testing.T, schema, changes, config string) framework.Mod {
	t.Helper()
	dir := t.TempDir()
	manifestJSON := `{"Name":"Compatibility Pack","Author":"Test","Version":"1.0.0","UniqueID":"Pack.Compat","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher","MinimumVersion":"1.0.0"}}`
	contentJSON := `{"ConfigSchema":` + schema + `,"Changes":` + changes + `}`
	for name, body := range map[string]string{"manifest.json": manifestJSON, "content.json": contentJSON} {
		testfs.WriteFile(t, dir, name, body)
	}
	if config != "" {
		testfs.WriteFile(t, dir, "config.json", config)
	}
	return packs.FromDisk(framework.Mod{Key: "pack", Enabled: true, Folder: dir, UniqueID: "Pack.Compat", Name: "Compatibility Pack"})
}

func TestCompatibilitySettingContainsForm(t *testing.T) {
	schema := `{"Patch":{"Default":"on","AllowValues":"on, off"}}`
	active := settingPack(t, schema, `[{"Action":"Load","Target":"a","When":{"Patch|contains=on":true,"HasMod":"Other.Mod"}}]`, `{"Patch":"on"}`)
	if got := compatibilitySettings([]framework.Mod{active, settingMod()}); len(got) != 0 {
		t.Fatalf("contains=true already met: %#v", got)
	}
	off := settingPack(t, schema, `[{"Action":"Load","Target":"a","When":{"Patch |contains=off":false,"HasMod":"Other.Mod"}}]`, `{"Patch":"off"}`)
	hints := compatibilitySettings([]framework.Mod{off, settingMod()})
	if len(hints) != 1 || len(hints[0].Suggested) != 1 || hints[0].Suggested[0] != "on" {
		t.Fatalf("contains=false should suggest the other value: %#v", hints)
	}
}

func TestCompatibilitySettingVariantFieldIsNotSuggested(t *testing.T) {
	pack := settingPack(t, `{"Room":{"Default":"None","AllowValues":"Plain, Glass, None"}}`, `[{"Action":"Load","Target":"a","When":{"Room":"Plain","HasMod":"Other.Mod"}}]`, `{"Room":"None"}`)
	if got := compatibilitySettings([]framework.Mod{pack, settingMod()}); len(got) != 0 {
		t.Fatalf("variant field suggested: %#v", got)
	}
}

func TestCompatibilitySettingOnIncludedPropertyOnlyPatch(t *testing.T) {
	pack := settingPack(t, `{"Animals":{"Default":"off","AllowValues":"on, off"}}`, `[{"Action":"Include","FromFile":"cc.json","When":{"HasMod":"Other.Mod"}}]`, "")
	cc := `{"Changes":[{"Action":"EditMap","Target":"Maps/X","MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Back","SetProperties":{"A":"b"}}],"When":{"Animals":"on"}}]}`
	if err := os.WriteFile(filepath.Join(pack.Folder, "cc.json"), []byte(cc), 0o600); err != nil {
		t.Fatal(err)
	}
	hints := compatibilitySettings([]framework.Mod{pack, settingMod()})
	if len(hints) != 1 || hints[0].Field != "Animals" || hints[0].Suggested[0] != "on" {
		t.Fatalf("hints = %#v", hints)
	}
}

func TestCompatibilitySettingFieldActiveForAnyModIsNotSuggested(t *testing.T) {
	changes := `[{"Action":"Load","Target":"a","When":{"Shift":true,"HasMod":"Other.Mod"}},{"Action":"Load","Target":"b","When":{"Shift":false,"HasMod":"Other.Mod","HasMod |contains=Third.Mod":true}}]`
	third := settingMod()
	third.UniqueID, third.Key = "Third.Mod", "third"
	pack := settingPack(t, `{"Shift":{"Default":true}}`, changes, `{"Shift":true}`)
	if got := compatibilitySettings([]framework.Mod{pack, settingMod(), third}); len(got) != 0 {
		t.Fatalf("hints = %#v", got)
	}
}

func TestConfigOffPatchDoesNotConflict(t *testing.T) {
	changes := `[{"Action":"EditImage","Target":"Maps/X","ToArea":{"X":0,"Y":0,"Width":8,"Height":8},"When":{"Recolor":true}}]`
	off := settingPack(t, `{"Recolor":{"Default":true}}`, changes, `{"Recolor":false}`)
	other := settingPack(t, `{"Recolor":{"Default":true}}`, changes, `{"Recolor":true}`)
	other.Key, other.UniqueID, other.Name = "other", "Other.Pack", "Other Pack"
	if got := assetConflicts([]framework.Mod{off, other}); len(got) != 0 {
		t.Fatalf("switched-off patch conflicted: %+v", got)
	}
	off2 := settingPack(t, `{"Recolor":{"Default":true}}`, changes, `{"Recolor":true}`)
	if got := assetConflicts([]framework.Mod{off2, other}); len(got) != 1 {
		t.Fatalf("both on should conflict: %+v", got)
	}
}

func TestConflictOffersTheSettingThatRemovesOnePack(t *testing.T) {
	changes := `[{"Action":"EditMap","Target":"Maps/Desert","FromFile":"cart.tmx","ToArea":{"X":2,"Y":38,"Width":2,"Height":2},"When":{"DesertMinecart":true}}` +
		strings.Repeat(`,{"Action":"EditData","Target":"Data/Objects","Entries":{"Cart":"x"}}`, 5) + `]`
	cart := settingPack(t, `{"DesertMinecart":{"Default":true,"AllowValues":"true, false"}}`, changes, "")
	desert := settingPack(t, `{}`, `[{"Action":"EditMap","Target":"Maps/Desert","FromFile":"d.tmx","ToArea":{"X":0,"Y":0,"Width":60,"Height":156}}]`, "")
	desert.Key, desert.UniqueID, desert.Name = "desert", "Desert.Expansion", "Desert Expansion"
	got := assetConflicts([]framework.Mod{cart, desert})
	if len(got) != 1 || len(got[0].Fixes) != 1 {
		t.Fatalf("conflicts %+v", got)
	}
	if f := got[0].Fixes[0]; f.ID != "smapi:Pack.Compat" || f.Field != "DesertMinecart" || f.Value != "false" || f.Current != "true" {
		t.Fatalf("fix %+v", f)
	}
}
