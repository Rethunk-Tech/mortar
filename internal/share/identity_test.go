package share

import "testing"

func TestIdentityVSModDBKeepsModAndVersionApartFromNexus(t *testing.T) {
	a := Ref{VSModDB: "alpha", Version: "1.2.0"}
	if got := a.Identity(); got != "v:alpha@1.2.0" {
		t.Fatalf("Identity() = %q, want v:alpha@1.2.0", got)
	}
	for _, other := range []Ref{{VSModDB: "beta", Version: "1.2.0"}, {VSModDB: "alpha", Version: "1.3.0"}, {}} {
		if a.Identity() == other.Identity() {
			t.Fatalf("%+v and %+v share Identity %q", a, other, a.Identity())
		}
	}
}
