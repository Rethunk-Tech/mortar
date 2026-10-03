package modconfig

import "testing"

func TestParseConfigSchema(t *testing.T) {
	schema, err := Parse([]byte(`{
		"Format": "2.0.0",
		"ConfigSchema": {
			"Season": {
				"AllowValues": "Spring, Summer, Fall, Winter",
				"AllowMultiple": true,
				"AllowBlank": true,
				"Default": "Spring",
				"Description": "Which seasons",
				"Section": "World"
			}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	f, ok := schema.Lookup("season")
	if !ok || f.Name != "Season" || !f.AllowMultiple || !f.AllowBlank || f.Default != "Spring" || f.Section != "World" {
		t.Fatalf("%+v", f)
	}
	if got := len(f.AllowValues); got != 4 || f.AllowValues[1] != "Summer" {
		t.Fatalf("AllowValues = %v", f.AllowValues)
	}
	if err := schema.Validate("Season", "Spring, Autumn"); err == nil {
		t.Fatal("expected Autumn rejected")
	}
	if err := schema.Validate("Season", "Spring, Summer"); err != nil {
		t.Fatal(err)
	}
}

func TestIsContentPack(t *testing.T) {
	if !IsContentPack([]byte(`{"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`)) {
		t.Fatal("CP pack")
	}
	if IsContentPack([]byte(`{"UniqueID":"Some.Mod"}`)) {
		t.Fatal("code mod")
	}
}
