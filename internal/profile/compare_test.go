package profile

import (
	"errors"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestCopyModsAddsFromStoreAndRespectsTheRunningLock(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": `{"Name":"B","Author":"me","Version":"2.0.0","UniqueID":"Me.B"}`})
	from := mustCreate(t, e, "From")
	to := mustCreate(t, e, "To")
	if _, err := e.AddEntry("stardew", from.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", from.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", from.ID, "local-b", "smapi:Me.B", false); err != nil {
		t.Fatal(err)
	}

	got, err := e.CopyMods("stardew", from.ID, to.ID, []mod.ID{"smapi:Me.A", "smapi:me.b"})
	if err != nil {
		t.Fatal(err)
	}
	src, err := e.load("stardew", from.ID)
	if err != nil {
		t.Fatal(err)
	}
	d := CompareProfilesCLI(src, got)
	if len(d.OnlyA) != 0 || len(d.OnlyB) != 0 || len(d.DifferentVersion) != 0 || len(d.DifferentEnabled) != 0 || len(d.DifferentSource) != 0 {
		t.Fatalf("after copy diff = %+v", d)
	}
	off := false
	for _, entry := range got.Entries {
		if entry.Key == "local-b" {
			off = !entry.Enabled("smapi:Me.B")
		}
	}
	if !off {
		t.Fatalf("copied disabled mod as enabled: %+v", got.Entries)
	}

	e.item(t, "local-c", map[string]string{"manifest.json": `{"Name":"C","Author":"me","Version":"1.0.0","UniqueID":"Me.C"}`})
	if _, err := e.AddEntry("stardew", from.ID, "local-c", Source{Kind: KindLocal, Name: "c.zip"}); err != nil {
		t.Fatal(err)
	}
	e.Running = func(_, id string) bool { return id == to.ID }
	var re *RunningError
	if _, err := e.CopyMods("stardew", from.ID, to.ID, []mod.ID{"smapi:Me.C"}); !errors.As(err, &re) {
		t.Fatalf("running dest = %v", err)
	}
}
