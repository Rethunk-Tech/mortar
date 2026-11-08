package share

import "testing"

func TestIdentityVSModDB(t *testing.T) {
	a := Ref{VSModDB: "alpha", FileID: 12}
	b := Ref{VSModDB: "beta", FileID: 12}
	if a.Identity() != "v:alpha:12" {
		t.Fatalf("Identity() = %q; want v:alpha:12", a.Identity())
	}
	if a.Identity() == b.Identity() {
		t.Fatalf("VSModDB refs with the same FileID share Identity %q", a.Identity())
	}
	n := Ref{ModID: 0, FileID: 12}
	if n.Identity() != "n:0:12" {
		t.Fatalf("Nexus Identity() = %q; want n:0:12", n.Identity())
	}
	if a.Identity() == n.Identity() {
		t.Fatalf("VSModDB Identity fell into Nexus %q", n.Identity())
	}
}
