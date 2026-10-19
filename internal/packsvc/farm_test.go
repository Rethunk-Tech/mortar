package packsvc

import (
	"encoding/json"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
)

func farmEntry(key, id, version string, modID, fileID int) profile.Entry {
	return profile.Entry{
		Key:    key,
		Source: profile.Source{Kind: profile.KindNexus, Name: id + ".zip", ModID: modID, FileID: fileID, Version: version, Picture: "p.png"},
		Mods:   []profile.Component{{ID: mod.SMAPI(id), Name: id, Version: version}},
	}
}

func TestFarmListChecksAndFixesAGuest(t *testing.T) {
	off := farmEntry("nexus-9-9", "Host.Off", "1.0.0", 9, 9)
	off.Disabled = []mod.ID{mod.SMAPI("Host.Off")}
	local := profile.Entry{Key: "local-x", Source: profile.Source{Kind: profile.KindLocal, Name: "x.zip"}, Mods: []profile.Component{{ID: mod.SMAPI("Host.Local"), Name: "Local", Version: "1.0.0"}}}
	bundled := profile.Entry{Key: "smapi", Source: profile.Source{Kind: profile.SourceSMAPI}, Mods: []profile.Component{{ID: mod.SMAPI("SMAPI.ConsoleCommands"), Version: "4.0.0"}}}
	host := profile.Profile{ID: "h", Name: "Our farm", Entries: []profile.Entry{
		farmEntry("nexus-1-2", "Host.Same", "1.0.0", 1, 2), farmEntry("nexus-3-4", "Host.Old", "2.0.0", 3, 4),
		farmEntry("nexus-5-6", "Host.New", "1.0.0", 5, 6), off, local, bundled,
	}}
	guest := profile.Profile{ID: "g", Name: "Mine", Entries: []profile.Entry{
		farmEntry("nexus-1-2", "Host.Same", "1.0", 1, 2), farmEntry("nexus-3-3", "Host.Old", "1.9.0", 3, 3),
		farmEntry("nexus-7-7", "Mine.Only", "1.0.0", 7, 7),
	}}
	hostSvc := &Service{Profiles: &fakeProfiles{list: []profile.Profile{host}}}
	list, err := hostSvc.ExportFarm("stardew", "h")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Mods) != 4 || list.Mods[0].Source.Picture != "" {
		t.Fatalf("list holds %d mods, want the 4 enabled non-bundled ones, stripped of pictures: %+v", len(list.Mods), list.Mods)
	}
	text, err := json.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}

	q := &fakeQueue{}
	guestSvc := &Service{Profiles: &fakeProfiles{list: []profile.Profile{guest}}, Queue: q}
	check, err := guestSvc.CheckFarm("stardew", "g", string(text))
	if err != nil {
		t.Fatal(err)
	}
	got := map[mod.ID]FarmRow{}
	for _, r := range check.Rows {
		got[r.ID] = r
	}
	if len(got) != 3 || got[mod.SMAPI("Host.Old")].State != FarmDifferent || got[mod.SMAPI("Host.Old")].Mine != "1.9.0" ||
		got[mod.SMAPI("Host.New")].State != FarmMissing || !got[mod.SMAPI("Host.New")].Fixable ||
		got[mod.SMAPI("Host.Local")].State != FarmMissing || got[mod.SMAPI("Host.Local")].Fixable {
		t.Fatalf("rows = %+v", check.Rows)
	}

	fix, err := guestSvc.FixFarm(t.Context(), "stardew", "g", string(text), nil)
	if err != nil {
		t.Fatal(err)
	}
	if fix.Queued != 2 || len(fix.Manual) != 1 || fix.Manual[0] != "Local" || len(q.got) != 2 {
		t.Fatalf("fix = %+v, queued %+v", fix, q.got)
	}
	only := &fakeQueue{}
	guestSvc.Queue = only
	if fix, err = guestSvc.FixFarm(t.Context(), "stardew", "g", string(text), []mod.ID{mod.SMAPI("Host.New")}); err != nil || fix.Queued != 1 || only.got[0].ModID != 5 {
		t.Fatalf("one mod: %+v, %v, %+v", fix, err, only.got)
	}
	for _, r := range q.got {
		switch r.ModID {
		case 3:
			if r.Kind != queue.KindUpdate || r.CurrentKey != "nexus-3-3" || r.FileID != 4 || r.Latest || !r.Exact {
				t.Errorf("update = %+v", r)
			}
		case 5:
			if r.Kind != queue.KindInstall || r.FileID != 6 || r.Profile != "g" {
				t.Errorf("install = %+v", r)
			}
		default:
			t.Errorf("unexpected request %+v", r)
		}
	}
}

func TestFarmListRefusesWhatIsNotOne(t *testing.T) {
	for _, text := range []string{"", "nope", `{"game":"stardew","mods":[]}`, `{"game":"lethal-company","mods":[{"id":"smapi:A"}]}`} {
		if _, err := parseFarmFor("stardew", text); err == nil {
			t.Errorf("accepted %q", text)
		}
	}
	if _, err := (&Service{Profiles: &fakeProfiles{}}).ExportFarm("sims4", "p"); err == nil {
		t.Error("exported a list for a game with no mod ids to match")
	}
}

func TestFarmCheckReadsShareModListText(t *testing.T) {
	named := func(key, name, version string, modID int) profile.Entry {
		e := farmEntry(key, key, version, modID, 1)
		e.Mods[0].Name = name
		return e
	}
	guest := profile.Profile{ID: "g", Entries: []profile.Entry{
		named("a", "Content Patcher", "2.8.3", 1915), named("b", "Mod *[beta]* (new)_x", "0.9", 7),
		named("c", "Twin", "1.0", 8), named("d", "Twin", "2.0", 8),
	}}
	q := &fakeQueue{}
	svc := &Service{Profiles: &fakeProfiles{list: []profile.Profile{guest}}, Queue: q}
	lists := map[string]string{
		"markdown": "Enabled\n- [Content Patcher](https://www.nexusmods.com/stardewvalley/mods/1915) 2.8.3\n" +
			"- [Mod \\*\\[beta\\]\\* \\(new\\)\\_x](https://www.nexusmods.com/stardewvalley/mods/7) 1.0\n- [Twin](https://www.nexusmods.com/stardewvalley/mods/8) 2.0\n" +
			"- [Json Assets](https://github.com/spacechase0/JsonAssets) 1.2\n- Local Only 3.1\n\nDisabled\n- [Off](https://example.com/off) 2",
		"plain": "Enabled\nContent Patcher v2.8.3 - https://www.nexusmods.com/stardewvalley/mods/1915\n" +
			"Mod *[beta]* (new)_x v1.0 - https://www.nexusmods.com/stardewvalley/mods/7\nTwin v2.0 - https://www.nexusmods.com/stardewvalley/mods/8\n" +
			"Json Assets v1.2 - https://github.com/spacechase0/JsonAssets\nLocal Only v3.1\n\nDisabled\nOff v2 - https://example.com/off",
	}
	for form, text := range lists {
		check, err := svc.CheckFarm("stardew", "g", text)
		if err != nil {
			t.Fatalf("%s: %v", form, err)
		}
		got := map[string]FarmRow{}
		for _, r := range check.Rows {
			got[r.Name] = r
		}
		beta, assets, local := got["Mod *[beta]* (new)_x"], got["Json Assets"], got["Local Only"]
		if len(got) != 3 || beta.State != FarmDifferent || beta.Mine != "0.9" || !beta.Fixable ||
			assets.State != FarmMissing || !assets.Fixable || local.State != FarmMissing || local.Fixable {
			t.Errorf("%s: rows = %+v", form, check.Rows)
		}
		q.got = nil
		fix, err := svc.FixFarm(t.Context(), "stardew", "g", text, []mod.ID{beta.ID, assets.ID})
		if err != nil || fix.Queued != 2 || len(q.got) != 2 || q.got[0].ModID != 7 || q.got[0].CurrentKey != "b" || q.got[1].Repo != "spacechase0/JsonAssets" {
			t.Errorf("%s: fix = %+v, %v, queued %+v", form, fix, err, q.got)
		}
	}
	if _, err := parseFarmFor("stardew", "- [Elden Mod](https://www.nexusmods.com/eldenring/mods/3) 1.0"); err == nil {
		t.Error("accepted a list of another game's Nexus pages")
	}
}

func TestFarmListMatchesThunderstoreAndVintageStoryMods(t *testing.T) {
	pkg := func(key, id, version string) profile.Entry {
		return profile.Entry{
			Key:    key,
			Source: profile.Source{Kind: profile.KindThunderstore, Name: id, Version: version},
			Mods:   []profile.Component{{ID: mod.NewID(mod.FormatThunderstore, id), Name: id, Version: version}},
		}
	}
	vs := func(key, version string) profile.Entry {
		return profile.Entry{
			Key:    key,
			Source: profile.Source{Kind: profile.KindVSModDB, Name: "7-8", FileID: 9, Version: version},
			Mods:   []profile.Component{{ID: mod.NewID(mod.FormatFolder, key), Name: "Carry", Version: version, ModID: "carryon"}},
		}
	}
	host := profile.Profile{ID: "h", Entries: []profile.Entry{pkg("a", "Ns-Same", "1.0.0"), pkg("b", "Ns-Old", "2.0.0"), pkg("c", "Ns-New", "1.0.0")}}
	guest := profile.Profile{ID: "g", Entries: []profile.Entry{pkg("a", "ns-same", "1.0.0"), pkg("b", "Ns-Old", "3.0.0")}}
	svc := &Service{Profiles: &fakeProfiles{list: []profile.Profile{host, guest}}, Queue: &fakeQueue{}}
	list, err := svc.ExportFarm("lethal-company", "h")
	if err != nil {
		t.Fatal(err)
	}
	text, _ := json.Marshal(list)
	check, err := svc.CheckFarm("lethal-company", "g", string(text))
	if err != nil {
		t.Fatal(err)
	}
	if len(check.Rows) != 2 || check.Rows[0].State != FarmDifferent || check.Rows[0].Mine != "3.0.0" || check.Rows[1].State != FarmMissing {
		t.Errorf("rows = %+v", check.Rows)
	}
	if _, err := svc.CheckFarm("valheim", "g", `{"game":"stardew","mods":[{"id":"smapi:A"}]}`); err == nil {
		t.Error("checked a list made for another game")
	}

	svc.Profiles = &fakeProfiles{list: []profile.Profile{{ID: "h", Entries: []profile.Entry{vs("x", "1.2.0")}}, {ID: "g", Entries: []profile.Entry{vs("y", "1.1.0")}}}}
	list, err = svc.ExportFarm("vintage-story", "h")
	if err != nil || len(list.Mods) != 1 || list.Mods[0].ID != mod.NewID(mod.FormatModInfo, "carryon") {
		t.Fatalf("vintage story list = %+v, %v", list, err)
	}
	text, _ = json.Marshal(list)
	if check, err = svc.CheckFarm("vintage-story", "g", string(text)); err != nil || len(check.Rows) != 1 || check.Rows[0].Mine != "1.1.0" || !check.Rows[0].Fixable {
		t.Errorf("vintage story rows = %+v, %v", check.Rows, err)
	}
}

func TestFarmListDropsSourcesTheGameDoesNotOffer(t *testing.T) {
	text := `{"game":"riskofrain2","name":"x","mods":[` +
		`{"id":"thunderstore:Ns-A","name":"A","version":"1.0.0","source":{"kind":"thunderstore","name":"Ns-A","version":"1.0.0"}},` +
		`{"id":"thunderstore:Ns-B","name":"B","version":"1.0.0","source":{"kind":"github","repo":"evil/repo"}},` +
		`{"id":"thunderstore:Ns-C","name":"C","version":"1.0.0","source":{"kind":"nexus","modId":5}}]}`
	l, err := parseFarmFor("riskofrain2", text)
	if err != nil {
		t.Fatal(err)
	}
	if l.Mods[0].Source.Kind != profile.KindThunderstore || l.Mods[1].Source.Kind != "" || l.Mods[2].Source.Kind != "" {
		t.Fatalf("sources = %+v", l.Mods)
	}
}
