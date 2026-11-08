package profile

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestModSetHashIgnoresDisabledAndCacheSetRoundTrip(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Ns-ModA", "Ns-ModB"} {
		src := t.TempDir()
		writeFile(t, src, "manifest.json", `{"name":"`+name+`","version_number":"1.0.0"}`)
		writeFile(t, src, name+".dll", name)
		key := store.PackageKey(name, "1.0.0")
		if err := addAnyDir(t, e.items, "lethal-company", key, src); err != nil {
			t.Fatal(err)
		}
		if _, err := e.AddEntry("lethal-company", p.ID, key, Source{Kind: KindThunderstore, Name: name, Version: "1.0.0"}); err != nil {
			t.Fatal(err)
		}
	}
	const modA = "thunderstore:Unknown-Ns-ModA"
	first, err := e.ModSetHash("lethal-company", p.ID)
	if err != nil || first == "" {
		t.Fatalf("enabled set: %q, %v", first, err)
	}
	if _, err := e.SetModEnabled("lethal-company", p.ID, "", modA, false); err != nil {
		t.Fatal(err)
	}
	disabled, err := e.ModSetHash("lethal-company", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if disabled == first {
		t.Fatal("disabling a package left ModSetHash unchanged")
	}
	if _, err := e.SetModEnabled("lethal-company", p.ID, "", modA, true); err != nil {
		t.Fatal(err)
	}
	again, err := e.ModSetHash("lethal-company", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Fatalf("re-enabled hash %q, want %q", again, first)
	}
	if got := e.CacheSet("lethal-company", p.ID); got != "" {
		t.Fatalf("CacheSet before record = %q", got)
	}
	if err := e.SetCacheSet("lethal-company", p.ID, first); err != nil {
		t.Fatal(err)
	}
	if got := e.CacheSet("lethal-company", p.ID); got != first {
		t.Fatalf("CacheSet = %q, want %q", got, first)
	}
}
