package profile

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestCopyModsWithNeedsBringsWhatTheTargetLacks(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	const lc = "lethal-company"
	from, err := e.Create(lc, "From")
	if err != nil {
		t.Fatal(err)
	}
	to, err := e.Create(lc, "To")
	if err != nil {
		t.Fatal(err)
	}
	install := func(profileID, name, version, deps string) {
		zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{
			"manifest.json": `{"name":"` + name + `","version_number":"` + version + `","dependencies":[` + deps + `]}`, name + ".dll": version,
		})
		if _, err := e.InstallSource(lc, profileID, zip, Source{Kind: KindThunderstore, Name: "Ns-" + name, Version: version}); err != nil {
			t.Fatal(err)
		}
	}
	install(from.ID, "Mod", "1.0.0", `"Ns-Lib-1.2.0","Ns-Base-1.0.0"`)
	install(from.ID, "Lib", "1.2.0", `"Ns-Base-1.0.0"`)
	install(from.ID, "Base", "1.0.0", ``)
	install(to.ID, "Base", "1.1.0", ``)
	install(to.ID, "Lib", "1.1.0", ``)

	needs, err := e.NeedsToCopy(lc, from.ID, to.ID, []mod.ID{"thunderstore:Ns-Mod"})
	if err != nil {
		t.Fatal(err)
	}
	if len(needs) != 1 || needs[0].ID != "thunderstore:Ns-Lib" {
		t.Fatalf("needs = %+v, want only Lib: the target's Lib is older than Mod accepts and its Base is new enough", needs)
	}
	got, err := e.CopyModsWithNeeds(lc, from.ID, to.ID, []mod.ID{"thunderstore:Ns-Mod"})
	if err != nil {
		t.Fatal(err)
	}
	versions := map[mod.ID]string{}
	for _, en := range got.Entries {
		for _, m := range en.Mods {
			versions[m.ID] = m.Version
		}
	}
	want := map[mod.ID]string{"thunderstore:Ns-Mod": "1.0.0", "thunderstore:Ns-Lib": "1.2.0", "thunderstore:Ns-Base": "1.1.0"}
	for id, v := range want {
		if versions[id] != v {
			t.Fatalf("target = %v, want %v", versions, want)
		}
	}
}
