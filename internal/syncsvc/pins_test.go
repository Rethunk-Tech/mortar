package syncsvc

import (
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

// payload is a profile of one Nexus package holding mods a and b, with the named ones switched off.
func payload(t *testing.T, off ...string) string {
	t.Helper()
	e := profile.Entry{
		Key:    "k",
		Source: profile.Source{Kind: profile.KindNexus, ModID: 1, FileID: 2},
		Mods:   []profile.Component{{ID: "smapi:A.One", Name: "One", Folder: "A"}, {ID: "smapi:B.Two", Name: "Two", Folder: "B"}},
	}
	for _, id := range off {
		e.Disabled = append(e.Disabled, mod.ID("smapi:"+id))
	}
	var b strings.Builder
	if _, err := share.Write(&b, "stardew", profile.Profile{Name: "Main", Entries: []profile.Entry{e}}, "", share.OwnInclude()); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func disabled(t *testing.T, p string) []string {
	t.Helper()
	pv, err := share.ReadBytes([]byte(p))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range pv.Entries {
		for _, id := range r.Disabled {
			out = append(out, string(id))
		}
	}
	return out
}

func TestPinnedModStaysOffTheFolderAndSurvivesAnApply(t *testing.T) {
	ctx := t.Context()
	folder := t.TempDir()
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a, ma := newMachine(t, folder, "Desktop", &clock)
	b, mb := newMachine(t, folder, "Laptop", &clock)

	ma.set("main", payload(t))
	scan(t, a)
	offers := scan(t, b)
	if err := b.Resolve(ctx, "stardew", "main", offers[0].Revision, Theirs); err != nil {
		t.Fatal(err)
	}
	if err := b.SetPin("stardew", "local-Main", PinMod, "smapi:B.Two", true); err != nil {
		t.Fatal(err)
	}

	// The laptop switches the mod off for itself: nothing new reaches the folder.
	mb.set("local-Main", payload(t, "B.Two"))
	before := shared1(t, folder)
	scan(t, b)
	if shared1(t, folder) != before || len(disabled(t, shared1(t, folder))) != 0 {
		t.Fatalf("a pinned mod's state reached the sync folder: %v", disabled(t, shared1(t, folder)))
	}
	if len(scan(t, a)) != 0 {
		t.Fatal("the desktop was offered a revision the laptop did not write")
	}

	// The desktop switches the other mod off; applying that on the laptop keeps its own state for the pinned one.
	ma.set("main", payload(t, "A.One"))
	scan(t, a)
	offers = scan(t, b)
	if len(offers) != 1 {
		t.Fatalf("offers = %+v", offers)
	}
	if err := b.Resolve(ctx, "stardew", "main", offers[0].Revision, Theirs); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(disabled(t, mb.profiles["local-Main"]), ","); !strings.Contains(got, "A.One") || !strings.Contains(got, "B.Two") {
		t.Fatalf("after apply the laptop has %q off, want both", got)
	}
	if len(scan(t, b)) != 0 || len(disabled(t, shared1(t, folder))) != 1 {
		t.Fatalf("applying must not push the pin: %v", disabled(t, shared1(t, folder)))
	}

	// Keep mine over a conflicting desktop revision keeps the pin local too.
	ma.set("main", payload(t, "A.One"))
	mb.set("local-Main", payload(t, "B.Two"))
	scan(t, a)
	offers = scan(t, b)
	if len(offers) != 1 || !offers[0].Conflict {
		t.Fatalf("offers = %+v", offers)
	}
	if err := b.Resolve(ctx, "stardew", "main", offers[0].Revision, Mine); err != nil {
		t.Fatal(err)
	}
	if got := disabled(t, shared1(t, folder)); len(got) != 0 {
		t.Fatalf("keep mine must write my A.One on and leave the pinned B.Two as the folder had it, wrote off: %v", got)
	}
	if len(scan(t, b)) != 0 {
		t.Fatal("keep mine left an offer behind")
	}
}

func TestPinNeedsASyncedProfile(t *testing.T) {
	clock := time.Now()
	s, m := newMachine(t, t.TempDir(), "Desktop", &clock)
	m.set("main", payload(t))
	if err := s.SetPin("stardew", "main", PinMod, "smapi:A.One", true); err == nil {
		t.Fatal("a profile that never synced has no revision to pin against")
	}
}
