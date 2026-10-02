package updatesvc

import (
	"context"
	"testing"
)

type fakeModUpdateSource struct {
	updates []ModUpdate
}

func (f fakeModUpdateSource) ModUpdates(context.Context) ([]ModUpdate, error) {
	return f.updates, nil
}

func TestDecideModUpdateNotification(t *testing.T) {
	source := fakeModUpdateSource{updates: []ModUpdate{
		{Game: "stardew", ProfileID: "farm", ProfileName: "Farm", Count: 2},
		{Game: "stardew", ProfileID: "backup", ProfileName: "Backup", Count: 1},
	}}
	updates, err := source.ModUpdates(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	state := &modUpdateState{}
	got, ok := decideModUpdateNotification(state, updates)
	if !ok || got.ProfileName != "Farm" || got.Count != 2 {
		t.Fatalf("first notification = %#v, %t", got, ok)
	}

	got, ok = decideModUpdateNotification(state, []ModUpdate{
		{Game: "stardew", ProfileID: "farm", ProfileName: "Farm", Count: 2},
		{Game: "stardew", ProfileID: "backup", ProfileName: "Backup", Count: 2},
	})
	if !ok || got.ProfileName != "Backup" || got.Count != 2 {
		t.Fatalf("increased notification = %#v, %t", got, ok)
	}

	if _, ok = decideModUpdateNotification(state, []ModUpdate{
		{Game: "stardew", ProfileID: "farm", ProfileName: "Farm", Count: 1},
		{Game: "stardew", ProfileID: "backup", ProfileName: "Backup", Count: 2},
	}); ok {
		t.Fatal("unchanged or reduced counts should not notify")
	}
}

func TestModUpdateNotificationPluralizes(t *testing.T) {
	_, one := ModUpdateNotification(ModUpdate{Count: 1, ProfileName: "Farm"})
	if one != "1 mod update available for Farm" {
		t.Fatalf("one update = %q", one)
	}
	_, many := ModUpdateNotification(ModUpdate{Count: 2, ProfileName: "Farm"})
	if many != "2 mod updates available for Farm" {
		t.Fatalf("many updates = %q", many)
	}
}
