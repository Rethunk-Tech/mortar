package deps

import "testing"

func TestSatisfies(t *testing.T) {
	for _, c := range []struct {
		scheme, version, constraint string
		want                        bool
	}{
		{SemverSMAPI, "1.2.0", "1.10", false},
		{SemverSMAPI, "2.0-beta", "2.0", false},
		{SemverSMAPI, "garbage", "1.0", true},
		{SemverSMAPI, "1.0", "", true},
		{SemverStrict, "1.10.0", "1.9.9", true},
		{SemverStrict, "1.0.0", "1.0.1", false},
		{SemverStrict, "1.0", "1.0.1", true},
		{Opaque, "r12", "r12", true},
		{Opaque, "r13", "r12", false},
	} {
		if got := Satisfies(c.scheme, c.version, c.constraint); got != c.want {
			t.Errorf("Satisfies(%s, %q, %q) = %v", c.scheme, c.version, c.constraint, got)
		}
	}
}

func TestVocabularies(t *testing.T) {
	d, err := Thunderstore("BepInEx-BepInExPack-5.4.2100")
	if err != nil || d.Target.Package != "thunderstore:bepinex-bepinexpack" || d.Constraint != "5.4.2100" || d.Relation != Required {
		t.Fatalf("thunderstore = %+v, %v", d, err)
	}
	if _, err := Thunderstore("nodash"); err == nil {
		t.Fatal("malformed accepted")
	}
	if d := Nexus(NexusRequirement{ModID: 7}); d.Target.String() != "nexus:7" {
		t.Fatalf("nexus = %+v", d)
	}
	if d := SMAPI("A.B", "1.0", false); d.Target.String() != "smapi:A.B" || d.Relation != Optional {
		t.Fatalf("smapi = %+v", d)
	}
}
