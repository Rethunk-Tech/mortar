package profile

import (
	"errors"
	"strings"
	"testing"
)

func TestPickHistoryItem(t *testing.T) {
	t.Parallel()
	items := []HistoryItem{
		{Kind: diffAdded, Name: "Alpha", Key: "k1"},
		{Kind: diffDisabled, Name: "alpha", Key: "k2"},
		{Kind: diffConfig, Name: "Beta", Key: "k3"},
		{Kind: diffConfig, Name: "Beta", Key: "k4"},
	}
	for _, tc := range []struct {
		want     string
		wantKind string
		errPart  string
	}{
		{"Alpha", diffDisabled, ""},
		{"k1", diffAdded, ""},
		{"Beta", "", "matches 2 changes"},
		{"Gamma", "", "not found"},
		{"", "", "not found"},
	} {
		got, err := pickHistoryItem(items, tc.want)
		if tc.errPart != "" {
			if err == nil || !strings.Contains(err.Error(), tc.errPart) {
				t.Errorf("pick %q err = %v, want %q", tc.want, err, tc.errPart)
			}
			continue
		}
		if err != nil || got.Kind != tc.wantKind {
			t.Errorf("pick %q = %+v, %v, want kind %s", tc.want, got, err, tc.wantKind)
		}
	}
}

// Reverting a removal puts back the mod and the overlay laid over it.
func TestRevertHistoryItemRemovedRestoresOverlay(t *testing.T) {
	t.Parallel()
	e, p, baseKey, _ := installOverlayPair(t)
	if _, err := e.RemoveEntry("stardew", p.ID, baseKey); err != nil {
		t.Fatal(err)
	}
	if cur, _ := e.read("stardew", p.ID); len(cur.Entries) != 0 {
		t.Fatalf("after remove: %+v", cur.Entries)
	}
	evs := e.mustHistory(t, p.ID)
	if _, err := e.RevertHistoryItem("stardew", p.ID, evs[0].ID, baseKey); err != nil {
		t.Fatal(err)
	}
	cur, err := e.read("stardew", p.ID)
	if err != nil || len(cur.Entries) != 2 || cur.Entries[1].OverlayOf != baseKey {
		t.Fatalf("after revert: %+v %v", cur.Entries, err)
	}
	if got := readLive(t, e, p.ID, baseKey, overlayDir+"/assets/a.png"); got != "A-opt" {
		t.Fatalf("overlaid file = %q", got)
	}
}

// A config edit reverts to the old bytes and records a "reverted" event; a running game refuses it.
func TestRevertHistoryItemConfig(t *testing.T) {
	t.Parallel()
	e, p := updEnv(t, map[string]string{"manifest.json": manifestJSON("Me.A"), "config.json": `{"v":1}`}, nil)
	if err := e.WriteConfig("stardew", p.ID, "a-1", "smapi:Me.A", `{"v":2}`); err != nil {
		t.Fatal(err)
	}
	edit := e.mustHistory(t, p.ID)[0]
	read := func() string {
		b, err := e.ReadConfig("stardew", p.ID, "a-1", "smapi:Me.A")
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	if !strings.Contains(read(), "2") {
		t.Fatalf("edit not written: %s", read())
	}

	e.Running = func(_, id string) bool { return id == p.ID }
	if _, err := e.RevertHistoryItem("stardew", p.ID, edit.ID, "Me.A"); err == nil {
		t.Fatal("config revert while running")
	} else if _, ok := errors.AsType[*RunningError](err); !ok {
		t.Fatalf("err = %v, want RunningError", err)
	}
	e.Running = nil

	if _, err := e.RevertHistoryItem("stardew", p.ID, edit.ID, "Me.A"); err != nil {
		t.Fatal(err)
	}
	if got := read(); !strings.Contains(got, "1") || strings.Contains(got, "2") {
		t.Fatalf("config after revert = %s", got)
	}
	head := e.mustHistory(t, p.ID)[0]
	if head.Kind != historyReverted || head.Change != ChangeRestoredFile {
		t.Fatalf("head event = %+v", head)
	}
}
