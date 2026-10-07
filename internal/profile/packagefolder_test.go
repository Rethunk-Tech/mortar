package profile

import (
	"errors"
	"testing"
)

func TestPackageEntryHasStateButNoModFolder(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.InstallSource(t.Context(), "lethal-company", p.ID, tsZip(t, "1.0.0"), Source{Kind: KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	got, _ := e.read("lethal-company", p.ID)
	key := got.Entries[0].Key
	st, err := e.ModState("lethal-company", p.ID, key, "thunderstore:Ns-Mod")
	if err != nil || st.Config != ConfigNone {
		t.Fatalf("state = %+v, %v", st, err)
	}
	if _, err := e.ModFolder("lethal-company", p.ID, key, "thunderstore:Ns-Mod"); !errors.Is(err, ErrNoModFolder) {
		t.Fatalf("a package's mod folder: %v", err)
	}
}
