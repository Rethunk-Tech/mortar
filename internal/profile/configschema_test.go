package profile

import (
	"encoding/json"
	"strings"
	"testing"
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
