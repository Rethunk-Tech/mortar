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
	if err := e.ApplyBundled("stardew", "smapi-1.0.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-x", Source{Kind: "local", Name: "x.zip"}); err != nil {
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
