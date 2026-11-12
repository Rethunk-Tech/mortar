package mod

import (
	"reflect"
	"testing"
)

func TestIDHelpers(t *testing.T) {
	ids := []ID{SMAPI(" Au.One "), "bepinex:Au.Two", "plain"}
	if got := Strings(ids); !reflect.DeepEqual(got, []string{"smapi: Au.One ", "bepinex:Au.Two", "plain"}) {
		t.Errorf("Strings = %q", got)
	}
	if got := Locals(ids); !reflect.DeepEqual(got, []string{" Au.One ", "Au.Two", "plain"}) {
		t.Errorf("Locals = %q", got)
	}
	for id, want := range map[ID]string{SMAPI(" Au.One "): "smapi:au.one", "bepinex:Au.Two": "bepinex:Au.Two"} {
		if id.Fold() != want {
			t.Errorf("Fold(%q) = %q, want %q", id, id.Fold(), want)
		}
	}
	if got := LooseName("Content-Patcher 2.0!"); got != "contentpatcher20" {
		t.Errorf("LooseName = %q", got)
	}
}

func TestDependencyIDRoundTrip(t *testing.T) {
	for _, id := range []ID{SMAPI("Au.One"), "thunderstore:Ns-Name"} {
		d := NewDependency(id, "1.0", true)
		if d.ModID() != id {
			t.Errorf("ModID = %q, want %q", d.ModID(), id)
		}
	}
	if d := NewDependency(SMAPI("Au.One"), "", false); d.Format != "" {
		t.Errorf("SMAPI dependency stores format %q, want empty", d.Format)
	}
}

func TestUnmarshalManifestAcceptsBOM(t *testing.T) {
	var v struct{ Name string }
	if err := UnmarshalManifest([]byte(BOM+`{"Name":"x"}`), &v); err != nil || v.Name != "x" {
		t.Fatalf("got %+v, %v", v, err)
	}
	if got := StripBOM([]byte("abc")); string(got) != "abc" {
		t.Errorf("StripBOM changed BOM-less input: %q", got)
	}
}
