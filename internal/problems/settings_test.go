package problems

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompatibilitySettingSuggestionWhenOff(t *testing.T) {
	pack := settingPack(t, `{"Enabled":{"Default":false}}`, `[{"Action":"Load","Target":"a","When":{"Enabled":true,"HasMod":"Other.Mod"}}]`, `{"Enabled":false}`)
	hints := compatibilitySettings([]Installed{pack, settingMod()})
	if len(hints) != 1 || hints[0].Field != "Enabled" || len(hints[0].Suggested) != 1 || hints[0].Suggested[0] != "true" {
		t.Fatalf("hints = %#v", hints)
	}
}

func TestCompatibilitySettingAlreadyActiveIsNotSuggested(t *testing.T) {
	pack := settingPack(t, `{"Enabled":{"Default":false}}`, `[{"Action":"Load","Target":"a","When":{"Enabled":true,"HasMod":"Other.Mod"}}]`, `{"Enabled":true}`)
	if got := compatibilitySettings([]Installed{pack, settingMod()}); len(got) != 0 {
		t.Fatalf("hints = %#v", got)
	}
}

func TestCompatibilitySettingMultipleValuesOneActiveIsNotSuggested(t *testing.T) {
	pack := settingPack(t, `{"Modes":{"Default":"off","AllowMultiple":true}}`, `[{"Action":"Load","Target":"a","When":{"Modes":"high, low","HasMod":"Other.Mod"}}]`, `{"Modes":"off, low"}`)
	if got := compatibilitySettings([]Installed{pack, settingMod()}); len(got) != 0 {
		t.Fatalf("hints = %#v", got)
	}
}

func TestCompatibilitySettingUnion(t *testing.T) {
	pack := settingPack(t, `{"Mode":{"Default":"off"}}`, `[{"Action":"Load","Target":"a","When":{"Mode":"high","HasMod":"Other.Mod"}},{"Action":"Load","Target":"b","When":{"Mode":"low","HasMod":"Other.Mod"}}]`, `{"Mode":"off"}`)
	hints := compatibilitySettings([]Installed{pack, settingMod()})
	if len(hints) != 1 || len(hints[0].Suggested) != 2 || hints[0].Suggested[0] != "high" || hints[0].Suggested[1] != "low" {
		t.Fatalf("hints = %#v", hints)
	}
}

func TestCompatibilitySettingMissingConfigUsesDefault(t *testing.T) {
	pack := settingPack(t, `{"Enabled":{"Default":false,"Description":"Use the compatibility patch"}}`, `[{"Action":"Load","Target":"a","When":{"Enabled":true,"HasMod":"Other.Mod"}}]`, "")
	hints := compatibilitySettings([]Installed{pack, settingMod()})
	if len(hints) != 1 || hints[0].Current != "false" || hints[0].Description != "Use the compatibility patch" {
		t.Fatalf("hints = %#v", hints)
	}
}

func TestCompatibilitySettingFalseHasModIsIgnored(t *testing.T) {
	pack := settingPack(t, `{"Enabled":{"Default":false}}`, `[{"Action":"Load","Target":"a","When":{"Enabled":true,"HasMod |contains=Other.Mod":false}}]`, `{"Enabled":false}`)
	if got := compatibilitySettings([]Installed{pack, settingMod()}); len(got) != 0 {
		t.Fatalf("hints = %#v", got)
	}
}

func settingMod() Installed {
	return Installed{Enabled: true, UniqueID: "Other.Mod", Name: "Other"}
}

func settingPack(t *testing.T, schema, changes, config string) Installed {
	t.Helper()
	dir := t.TempDir()
	manifestJSON := `{"Name":"Compatibility Pack","Author":"Test","Version":"1.0.0","UniqueID":"Pack.Compat","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher","MinimumVersion":"1.0.0"}}`
	contentJSON := `{"ConfigSchema":` + schema + `,"Changes":` + changes + `}`
	for name, body := range map[string]string{"manifest.json": manifestJSON, "content.json": contentJSON} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if config != "" {
		if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(config), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return Installed{Key: "pack", Enabled: true, Folder: dir, UniqueID: "Pack.Compat", Name: "Compatibility Pack"}
}
