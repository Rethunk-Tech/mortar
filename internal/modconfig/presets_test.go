package modconfig

import (
	"fmt"
	"strings"
	"testing"
)

func TestSaveListApplyDeleteAndNames(t *testing.T) {
	data := t.TempDir()
	if err := SavePreset(data, "stardew", "Me.Mod", "farm", []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	got, err := ListPresets(data, "stardew", "Me.Mod")
	if err != nil || len(got) != 1 || got[0] != "farm" {
		t.Fatalf("list = %v, %v", got, err)
	}
	body, err := LoadPreset(data, "stardew", "Me.Mod", "farm")
	if err != nil || !strings.Contains(string(body), `"a"`) {
		t.Fatalf("load = %s, %v", body, err)
	}
	if err := DeletePreset(data, "stardew", "Me.Mod", "farm"); err != nil {
		t.Fatal(err)
	}
	got, err = ListPresets(data, "stardew", "Me.Mod")
	if err != nil || len(got) != 0 {
		t.Fatalf("after delete = %v, %v", got, err)
	}
}

func TestPresetNameValidation(t *testing.T) {
	data := t.TempDir()
	long := strings.Repeat("n", maxPresetName+1)
	for _, name := range []string{"", "a/b", `a\b`, ".", "..", long} {
		if err := SavePreset(data, "stardew", "Me.Mod", name, []byte(`{}`)); err == nil {
			t.Errorf("%q accepted", name)
		}
	}
	if err := SavePreset(data, "stardew", "Me.Mod", strings.Repeat("n", maxPresetName), []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := SavePreset(data, "stardew", "Me.Mod", "bad", []byte(`{`)); err == nil {
		t.Fatal("invalid JSON accepted")
	}
}

func TestAtMost32Presets(t *testing.T) {
	data := t.TempDir()
	for i := range maxPresetCount {
		if err := SavePreset(data, "stardew", "Cap.Mod", fmt.Sprintf("p%d", i), []byte(`{}`)); err != nil {
			t.Fatal(err)
		}
	}
	if err := SavePreset(data, "stardew", "Cap.Mod", "overflow", []byte(`{}`)); err == nil {
		t.Fatal("33rd preset accepted")
	}
	if err := SavePreset(data, "stardew", "Cap.Mod", "p0", []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
}
