package sharesvc

import (
	"bytes"
	"context"
	"encoding/base64"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

type fakeDismissals struct {
	have map[string][]string
}

func (f *fakeDismissals) DismissedTokens(game, id string) []string { return f.have[game+"/"+id] }
func (f *fakeDismissals) AdoptDismissed(game, id string, tokens []string) error {
	f.have[game+"/"+id] = append(f.have[game+"/"+id], tokens...)
	return nil
}

func storeMod(t *testing.T, items *store.Store, uniqueID string) string {
	t.Helper()
	src := t.TempDir()
	body := `{"Name":"` + uniqueID + `","Author":"a","Version":"1.0.0","UniqueID":"` + uniqueID + `","EntryDll":"m.dll"}`
	if err := fsx.WriteFile(filepath.Join(src, "manifest.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	key, err := store.HashDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := items.AddDir("stardew", key, src); err != nil {
		t.Fatal(err)
	}
	return key
}

func TestProblemChoicesTravelInAFileAndApplyOnImport(t *testing.T) {
	s, _ := newService(t, true)
	items, _ := testenv.Stores(t)
	s.d.Stored = func(game, k string) bool { _, err := items.Path(game, k); return err == nil }
	dis := &fakeDismissals{have: map[string][]string{"stardew/sender": {"broken\tme.old"}}}
	s.d.Dismissals = dis
	win, lose := storeMod(t, items, "Me.Win"), storeMod(t, items, "Me.Lose")
	winner, loser := mod.SMAPI("Me.Win"), mod.SMAPI("Me.Lose")
	sender := profile.Profile{Name: "Desk", Entries: []profile.Entry{
		{Key: win, Source: profile.Source{Kind: profile.KindLocal, Name: "win.zip"}, Mods: []profile.Component{{ID: winner, Folder: ".", LoadAfter: []mod.ID{loser}}}},
		{Key: lose, Source: profile.Source{Kind: profile.KindLocal, Name: "lose.zip"}, Mods: []profile.Component{{ID: loser, Folder: "."}}},
	}}
	send := func(inc share.Include) string {
		inc.LocalFiles = true
		inc.Dismissed = dis.DismissedTokens("stardew", "sender")
		var buf bytes.Buffer
		if _, err := share.Write(&buf, "stardew", sender, t.TempDir(), inc); err != nil {
			t.Fatal(err)
		}
		return base64.RawStdEncoding.EncodeToString(buf.Bytes())
	}

	without, err := s.PreviewData(context.Background(), "stardew", send(share.Include{Notes: true}), "")
	if err != nil || without.Choices != 0 {
		t.Fatalf("a file sent without problem choices previews %d choices, %v", without.Choices, err)
	}

	encoded := send(share.OwnInclude())
	pv, err := s.PreviewData(context.Background(), "stardew", encoded, "")
	if err != nil || pv.Choices != 2 {
		t.Fatalf("preview choices = %d, %v; want the dismissal and the win", pv.Choices, err)
	}
	res, err := s.Import(context.Background(), "stardew", pv.Session, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.find("stardew", res.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(got.Entries, func(e profile.Entry) bool { return e.Key == win })
	if i < 0 || !slices.ContainsFunc(got.Entries[i].Mods[0].LoadAfter, func(id mod.ID) bool { return mod.Equal(id, loser) }) {
		t.Fatalf("the winner does not load after the loser on the receiver: %+v", got.Entries)
	}
	if !slices.Contains(dis.have["stardew/"+res.Profile.ID], "broken\tme.old") {
		t.Fatalf("the dismissal did not reach the receiving profile: %v", dis.have)
	}
}

func TestSharedChoicesKeepTheReceiversOwnDecision(t *testing.T) {
	a, b := mod.SMAPI("Me.A"), mod.SMAPI("Me.B")
	p := profile.Profile{Entries: []profile.Entry{
		{Key: "ka", Source: profile.Source{Kind: profile.KindNexus, ModID: 1, FileID: 1}, Mods: []profile.Component{{ID: a, Folder: "."}}},
		{Key: "kb", Source: profile.Source{Kind: profile.KindNexus, ModID: 2, FileID: 2}, Mods: []profile.Component{{ID: b, Folder: ".", LoadAfter: []mod.ID{a}}}},
	}}
	win := share.Win{Ref: share.Ref{ModID: 1, FileID: 1}, Winner: a, Loser: b}
	if key, ok := winTarget(p, win); ok {
		t.Fatalf("a win the receiver already reversed applied to %q", key)
	}
	p.Entries[1].Mods[0].LoadAfter = nil
	if key, ok := winTarget(p, win); !ok || key != "ka" {
		t.Fatalf("winTarget = %q, %v", key, ok)
	}
	if _, ok := winTarget(profile.Profile{Entries: p.Entries[:1]}, win); ok {
		t.Fatal("a win applied although the receiver lacks the loser")
	}
}
