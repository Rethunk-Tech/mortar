package profile

import "testing"

func TestBundledModsAreHiddenAndKept(t *testing.T) {
	e := newEnv(t)
	e.item(t, "smapi-1.0.0", bundle())
	e.item(t, "local-x", map[string]string{"manifest.json": manifestJSON("Other")})
	p, err := e.Create("stardew", "a")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ApplyBundled("stardew", smapiBundle("smapi-1.0.0")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-x", Source{Kind: KindLocal, Name: "x.zip"}); err != nil {
		t.Fatal(err)
	}
	user, err := e.UserMods("stardew", p.ID)
	if err != nil || len(user) != 1 || user[0].UniqueID != "Other" {
		t.Fatalf("user mods = %+v, %v", user, err)
	}
	all, err := e.Mods("stardew", p.ID)
	if err != nil || len(all) != 3 {
		t.Fatalf("installed mods = %+v, %v", all, err)
	}
	if _, err := e.RemoveEntry("stardew", p.ID, "smapi-1.0.0"); err == nil {
		t.Fatal("removing the bundled entry succeeded")
	}
}

func smapiBundle(key string) Bundle {
	return Bundle{Key: key, Source: Source{Kind: SourceSMAPI, Name: "SMAPI"}}
}

func TestBridgeEntryIsHiddenKeptAndNotRemovable(t *testing.T) {
	e := newEnv(t)
	e.item(t, "smapi-1.0.0", bundle())
	e.item(t, "bridge-1.0.0", map[string]string{"manifest.json": manifestJSON("Rethunk.MortarSmapiBridge")})
	e.item(t, "bridge-2.0.0", map[string]string{"manifest.json": manifestJSON("Rethunk.MortarSmapiBridge")})
	p, err := e.Create("stardew", "a")
	if err != nil {
		t.Fatal(err)
	}
	bridge := func(key string) Bundle { return Bundle{Key: key, Source: Source{Kind: SourceMortar, Name: "Mortar"}} }
	for _, b := range []Bundle{smapiBundle("smapi-1.0.0"), bridge("bridge-1.0.0")} {
		if err := e.ApplyBundled("stardew", b); err != nil {
			t.Fatal(err)
		}
	}
	// A new bridge replaces the old one and leaves SMAPI's entry alone.
	if err := e.ApplyBundled("stardew", bridge("bridge-2.0.0")); err != nil {
		t.Fatal(err)
	}
	got, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, en := range got.Entries {
		keys = append(keys, en.Key)
	}
	if len(keys) != 2 || keys[0] != "smapi-1.0.0" || keys[1] != "bridge-2.0.0" {
		t.Fatalf("entries = %v", keys)
	}
	if user, err := e.UserMods("stardew", p.ID); err != nil || len(user) != 0 {
		t.Fatalf("user mods = %+v, %v", user, err)
	}
	if _, err := e.RemoveEntry("stardew", p.ID, "bridge-2.0.0"); err == nil {
		t.Fatal("removing the bridge entry succeeded")
	}
}
