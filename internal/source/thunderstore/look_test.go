package thunderstore

import "testing"

func TestCategoryPrefersWhatThePackageIs(t *testing.T) {
	for want, cats := range map[string][]string{
		"Tools":     {"Mods", "Tools", "Libraries", "BepInEx"},
		"Libraries": {"BepInEx", "Client-side", "Libraries"},
		"Mods":      {"Mods", "BepInEx"},
		"":          nil,
	} {
		if got := Category(cats); got != want {
			t.Errorf("Category(%v) = %q, want %q", cats, got, want)
		}
	}
}

func TestLooksComeFromTheIndexAndTheCacheAlone(t *testing.T) {
	f := newFake(t)
	icon := "https://ccdn.thunderstore.io/live/repository/icons/Ns-Gen-1.0.0.png"
	l := custom("Ns", "Gen", ver{VersionNumber: "1.0.0", Icon: icon})
	l["categories"] = []string{"Mods", "Tools"}
	f.chunk0 = append(f.chunk0, l)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	got, err := d.Closure(t.Context(), "lethal-company", []Ref{{"Ns", "Gen", ""}}, "1.2.3")
	if err != nil || len(got) != 1 || got[0].Look != (Look{Icon: icon, Category: "Tools"}) {
		t.Fatalf("closure %+v %v", got, err)
	}
	f.srv.Close()
	looks, err := d.CachedLooks("lethal-company", []string{"NS-GEN", "Ns-Missing"})
	if err != nil || len(looks) != 1 || looks["ns-gen"] != (Look{Icon: icon, Category: "Tools"}) {
		t.Fatalf("cached looks %+v %v", looks, err)
	}
}
