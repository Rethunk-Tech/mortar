package share

import (
	"encoding/json"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestAParadoxRefRoundTripsMatchesItsEntryAndRefusesOtherShapes(t *testing.T) {
	ref := Ref{Paradox: "74604", FileID: 40, Version: "1.8.0"}
	b, err := json.Marshal(ref)
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseRef(b)
	if err != nil || got.Paradox != ref.Paradox || got.FileID != ref.FileID || got.Version != ref.Version || !got.valid() {
		t.Fatalf("%s read back as %+v, %v", b, got, err)
	}
	e := profile.Entry{Source: profile.Source{Kind: profile.KindParadox, Name: "74604", FileID: 40, Version: "1.8.0"}}
	if !got.MatchesEntry(e) {
		t.Fatal("the ref does not match the entry it came from")
	}
	if r, ok := identityOf(e); !ok || r.Paradox != "74604" {
		t.Fatalf("entry ref = %+v, %v", r, ok)
	}
	for _, bad := range []Ref{
		{Paradox: "../x", FileID: 1, Version: "1"},
		{Paradox: "74604", Version: "1"},
		{Paradox: "74604", FileID: 1},
		{Paradox: "74604", FileID: 1, Version: "a b"},
		{Paradox: "74604", FileID: 1, Version: "1", VSModDB: "1-2"},
		{Paradox: "74604", FileID: 1, Version: "1", ModID: 3},
	} {
		if bad.valid() {
			t.Fatalf("%+v was accepted", bad)
		}
	}
}
