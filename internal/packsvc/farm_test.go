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
			if r.Kind != queue.KindUpdate || r.CurrentKey != "nexus-3-3" || r.FileID != 4 || r.Latest {
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
		if _, err := ParseFarm(text); err == nil {
			t.Errorf("accepted %q", text)
		}
	}
	if _, err := (&Service{Profiles: &fakeProfiles{}}).ExportFarm("lethal-company", "p"); err == nil {
		t.Error("exported a list for another game")
	}
}
