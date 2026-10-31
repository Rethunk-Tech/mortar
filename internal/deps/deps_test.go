package deps

import "testing"

func TestCheck(t *testing.T) {
	for _, c := range []struct {
		scheme, version, constraint string
		want                        Status
	}{
		{SemverSMAPI, "1.2.0", "1.10", Unmet},
		{SemverSMAPI, "2.0-beta", "2.0", Unmet},
		{SemverSMAPI, "garbage", "1.0", Unknown},
		{SemverSMAPI, "1.0", "garbage", Unknown},
		{SemverSMAPI, "1.0", "", Met},
		{SemverStrict, "1.10.0", "1.9.9", Met},
		{SemverStrict, "1.0.0", "1.0.1", Unmet},
		{SemverStrict, "1.0", "1.0.1", Unmet},
		{SemverStrict, "v0.15.1", "0.16.0", Unmet},
		{SemverStrict, "1.2.3.4", "1.2.3", Met},
		{Opaque, "r12", "r12", Met},
		{Opaque, "r13", "r12", Unknown},
	} {
		if got := Check(c.scheme, c.version, c.constraint); got != c.want {
			t.Errorf("Check(%s, %q, %q) = %v, want %v", c.scheme, c.version, c.constraint, got, c.want)
		}
	}
}

func TestVocabularies(t *testing.T) {
	d, err := Thunderstore("BepInEx-BepInExPack-5.4.2100")
	if err != nil || d.Target.Package != "thunderstore:BepInEx-BepInExPack" || d.Constraint != "5.4.2100" || d.Relation != Required {
		t.Fatalf("thunderstore = %+v, %v", d, err)
	}
	if _, err := Thunderstore("nodash"); err == nil {
		t.Fatal("malformed accepted")
	}
	if d := SMAPI("A.B", "1.0", false); d.Target.Mod != "smapi:A.B" || d.Relation != Optional {
		t.Fatalf("smapi = %+v", d)
	}
}

func TestThunderstoreRefNeedsExactlyNamespaceNameVersion(t *testing.T) {
	if pkg, v, ok := ThunderstoreRef("Ns-Name-1.0.0"); !ok || pkg != "Ns-Name" || v != "1.0.0" {
		t.Fatalf("%q %q %v", pkg, v, ok)
	}
	for _, bad := range []string{"Ns-Name", "A-B-C-D", "-Name-1.0.0"} {
		if _, _, ok := ThunderstoreRef(bad); ok {
			t.Errorf("%q accepted", bad)
		}
	}
}
