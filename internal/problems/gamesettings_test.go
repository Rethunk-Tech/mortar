package problems

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

var modSettings = []components.RequiredSetting{
	{Path: "options", Key: "ModsEnabled", Value: "1", Message: "custom content is off"},
	{Path: "options", Key: "ScriptMods", Value: "1", Message: "script mods are off"},
}

func TestSettingFailuresReportOnlyAKeyWithAnotherValue(t *testing.T) {
	got := settingFailures("[Options]\n; a comment\nmodsenabled = 0\nOther=1\n", modSettings)
	if len(got) != 1 || got[0].Kind != KindGameSetting || got[0].Plugin != "ModsEnabled" || got[0].Message != "custom content is off" {
		t.Fatalf("got %+v", got)
	}
	if got := settingFailures("modsenabled=1\nscriptmods = 1\n", modSettings); len(got) != 0 {
		t.Fatalf("switched on settings are not findings: %+v", got)
	}
	if got := settingFailures("", modSettings); len(got) != 0 {
		t.Fatalf("keys the file lacks keep the game's default: %+v", got)
	}
}

func TestCheckingTheSettingsFileNeverChangesIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Options.ini")
	body := "[Options]\r\nModsEnabled = 0\r\nScriptMods = 0\r\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(path)
	if got := settingFailures(body, modSettings); len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	after, _ := fsx.ReadFile(path)
	stat, _ := os.Stat(path)
	if string(after) != body || !stat.ModTime().Equal(before.ModTime()) {
		t.Fatal("the file must be untouched")
	}
}
