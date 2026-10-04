package profile

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/modconfig"
)

func TestApplyConfigPresetWritesAndRecordsHistory(t *testing.T) {
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m, "A/config.json": `{"z":1}`}, map[string]string{"A/manifest.json": m + " "})
	svc := NewService(e.Store, t.TempDir(), nil)
	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := modconfig.SavePreset(root, "stardew", "me.a", "farm", []byte(`{"z":9}`)); err != nil {
		t.Fatal(err)
	}
	if err := svc.ApplyConfigPreset("stardew", p.ID, "a-1", "me.a", "farm"); err != nil {
		t.Fatal(err)
	}
	got, err := svc.ReadConfig("stardew", p.ID, "a-1", "me.a")
	if err != nil || !strings.Contains(got, `"z": 9`) {
		t.Fatalf("config = %s, %v", got, err)
	}
	events, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 || events[0].Kind != historyConfigPreset {
		t.Fatalf("history = %+v", events)
	}
}
