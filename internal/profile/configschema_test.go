package profile

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestSetConfigValueDottedPathPreservesOrder(t *testing.T) {
	t.Parallel()
	manifest := `{"Name":"P","UniqueID":"me.cp","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`
	content := `{"ConfigSchema":{"Season":{"AllowValues":"Spring, Summer, Fall","Default":"Spring"},"Enabled":{"Default":false}}}`
	e, p := updEnv(t,
		map[string]string{
			"A/manifest.json": manifest,
			"A/content.json":  content,
			"A/config.json":   "{\n  \"z\": 1,\n  \"nested\": {\"foo\": 1.50},\n  \"Season\": \"Spring\"\n}",
		},
		map[string]string{"A/manifest.json": manifest + " "},
	)
	svc := NewService(e.Store, t.TempDir(), nil)
	if err := svc.SetConfigValue("stardew", p.ID, "a-1", "smapi:me.cp", "nested.foo", "2.25"); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetConfigValue("stardew", p.ID, "a-1", "smapi:me.cp", "Season", "Fall"); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetConfigValue("stardew", p.ID, "a-1", "smapi:me.cp", "Season", "Nope"); err == nil {
		t.Fatal("invalid CP value accepted")
	}
	raw, err := svc.ReadConfig("stardew", p.ID, "a-1", "smapi:me.cp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, `"z"`) || strings.Index(raw, `"z"`) > strings.Index(raw, `"nested"`) {
		t.Fatalf("key order lost: %s", raw)
	}
	if !strings.Contains(raw, "2.25") {
		t.Fatalf("number format lost: %s", raw)
	}
	fields, err := svc.ListConfigFields("stardew", p.ID, "a-1", "smapi:me.cp")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range fields {
		if f.Path == "Season" {
			found = f.Value == "Fall" && len(f.AllowValues) == 3
		}
	}
	if !found {
		t.Fatalf("fields = %+v", fields)
	}
	schema, err := svc.ReadContentSchema("stardew", p.ID, "a-1", "smapi:me.cp")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(schema), &doc); err != nil || doc["Season"] == nil {
		t.Fatalf("schema %s", schema)
	}
}

func TestSetConfigValueKeepsEachFileKind(t *testing.T) {
	t.Parallel()
	cpManifest := `{"Name":"P","UniqueID":"me.cp","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`
	cpContent := `{"ConfigSchema":{"Mist":{"AllowValues":"true, false","Default":"true"},"Fog":{"AllowValues":"true, false","Default":"true"}}}`
	smManifest := `{"Name":"S","UniqueID":"me.smapi"}`
	e, p := updEnv(t,
		map[string]string{
			"A/manifest.json": cpManifest,
			"A/content.json":  cpContent,
			"A/config.json":   `{"Mist":"true","Fog":"true"}`,
			"B/manifest.json": smManifest,
			"B/config.json":   `{"On":true,"Count":3,"Flag":false,"Name":"x"}`,
		},
		map[string]string{"A/manifest.json": cpManifest + " "},
	)
	// The shipped files hold the defaults; the profile's copies have lost some values.
	for id, text := range map[mod.ID]string{
		"smapi:me.cp":    `{"Mist":"true","Fog":null}`,
		"smapi:me.smapi": `{"On":true,"Count":null,"Flag":null,"Name":null}`,
	} {
		if err := e.WriteConfig("stardew", p.ID, "a-1", id, text); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(e.Store, t.TempDir(), nil)
	set := func(key string, id mod.ID, field, value string) {
		t.Helper()
		if err := svc.SetConfigValue("stardew", p.ID, key, id, field, value); err != nil {
			t.Fatal(err)
		}
	}
	set("a-1", "smapi:me.cp", "Mist", "false")
	set("a-1", "smapi:me.cp", "Fog", "false")
	set("a-1", "smapi:me.smapi", "On", "false")
	set("a-1", "smapi:me.smapi", "Count", "4")
	set("a-1", "smapi:me.smapi", "Flag", "true")
	set("a-1", "smapi:me.smapi", "Name", "y")
	cp, _ := svc.ReadConfig("stardew", p.ID, "a-1", "smapi:me.cp")
	sm, _ := svc.ReadConfig("stardew", p.ID, "a-1", "smapi:me.smapi")
	for _, want := range []string{`"Mist": "false"`, `"Fog": "false"`} {
		if !strings.Contains(cp, want) {
			t.Errorf("content pack config lacks %s: %s", want, cp)
		}
	}
	for _, want := range []string{`"On": false`, `"Count": 4`, `"Flag": true`, `"Name": "y"`} {
		if !strings.Contains(sm, want) {
			t.Errorf("SMAPI config lacks %s: %s", want, sm)
		}
	}
}
