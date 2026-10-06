package profile

import "testing"

func TestEnabledPackagesGiveAFilePackageItsManifestVersion(t *testing.T) {
	e := newEnv(t)
	p := mustCreate(t, e, "Lobby")
	e.item(t, "local-a", map[string]string{"BepInEx/plugins/Ns-A/A.dll": "a"})
	e.item(t, "pkg-b", map[string]string{"BepInEx/plugins/Ns-B/B.dll": "b"})
	if _, err := e.update("stardew", p.ID, func(p *Profile, _ string) error {
		p.Entries = append(p.Entries,
			Entry{Key: "local-a", Package: true, Source: Source{Kind: KindLocal, Name: "Ns-A-1.2.3.zip"}, Mods: []Component{{ID: "thunderstore:Ns-A", Name: "A", Version: "1.2.3"}}},
			Entry{Key: "pkg-b", Package: true, Source: Source{Kind: KindThunderstore, Name: "Ns-B", Version: "2.0.0"}, Mods: []Component{{ID: "thunderstore:Ns-B", Name: "B", Version: "1.9.9"}}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	got, err := e.EnabledPackages("stardew", p.ID)
	if err != nil || len(got) != 2 || got[0].Version != "1.2.3" || got[1].Version != "2.0.0" {
		t.Fatalf("got %+v, %v", got, err)
	}
}
