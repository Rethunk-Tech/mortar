package profile

import (
	"encoding/json"
	"testing"
)

func TestSetConfigValuePreservesFieldsAndWritesBoolean(t *testing.T) {
	t.Parallel()
	manifest := manifestJSON("me.a")
	content := `{"ConfigSchema":{"Enabled":{"Default":false}}}`
	e, p := updEnv(t,
		map[string]string{
			"A/manifest.json": manifest,
			"A/content.json":  content,
			"A/config.json":   "{\n  \"other\":42, // retained\n  \"ENABLED\":false,\n}",
		},
		map[string]string{"A/manifest.json": manifest + " "},
	)
	svc := NewService(e.Store, t.TempDir(), nil)
	if err := svc.SetConfigValue("stardew", p.ID, "a-1", "smapi:me.a", "enabled", "true"); err != nil {
		t.Fatal(err)
	}
	raw, err := svc.ReadConfig("stardew", p.ID, "a-1", "smapi:me.a")
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		t.Fatal(err)
	}
	var enabled bool
	if err := json.Unmarshal(values["ENABLED"], &enabled); err != nil || !enabled {
		t.Fatalf("ENABLED = %s, want bool true", values["ENABLED"])
	}
	var other int
	if err := json.Unmarshal(values["other"], &other); err != nil || other != 42 {
		t.Fatalf("other = %s, want 42", values["other"])
	}
}
