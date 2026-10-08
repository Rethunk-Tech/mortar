package settings

import (
	"encoding/json"
	"testing"
)

func TestToggleDefaultsOn(t *testing.T) {
	d := Defaults()
	if d.CheckModUpdatesOnStart == nil || !*d.CheckModUpdatesOnStart {
		t.Fatal("checkModUpdatesOnStart default is on")
	}
	if d.TellWhenSmapiOut == nil || !*d.TellWhenSmapiOut {
		t.Fatal("tellWhenSmapiOut default is on")
	}
	if d.EnableModsWhenInstalled == nil || !*d.EnableModsWhenInstalled {
		t.Fatal("enableModsWhenInstalled default is on")
	}
	if !d.LanSharing {
		t.Fatal("lanSharing default is on")
	}
	if d.LanPort != 0 || len(d.LanAddresses) != 0 {
		t.Fatalf("LAN defaults = port %d, addresses %v", d.LanPort, d.LanAddresses)
	}
}

func TestOmittedTogglesNormalizeOn(t *testing.T) {
	var s Settings
	if err := json.Unmarshal([]byte(`{"accent":"sand"}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.CheckModUpdatesOnStart != nil || s.TellWhenSmapiOut != nil {
		t.Fatal("omitted keys must stay nil until normalize")
	}
	normalizeToggles(&s)
	if !*s.CheckModUpdatesOnStart || !*s.TellWhenSmapiOut {
		t.Fatal("omitted toggles become on")
	}
	if !*s.EnableModsWhenInstalled {
		t.Fatal("omitted enableModsWhenInstalled becomes on")
	}
}

func TestExplicitFalseTogglesKept(t *testing.T) {
	var s Settings
	if err := json.Unmarshal([]byte(`{"checkModUpdatesOnStart":false,"tellWhenSmapiOut":false}`), &s); err != nil {
		t.Fatal(err)
	}
	normalizeToggles(&s)
	if *s.CheckModUpdatesOnStart || *s.TellWhenSmapiOut {
		t.Fatal("explicit false must stay off")
	}
}
