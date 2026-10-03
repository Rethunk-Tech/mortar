package profile

import (
	"testing"
	"time"
)

func TestChangesSinceDecodesHistoryOnce(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	n := 0
	onHistoryDecode = func() { n++ }
	t.Cleanup(func() { onHistoryDecode = nil })
	if _, err := e.ChangesSince("stardew", p.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("ChangesSince decoded history %d times, want 1", n)
	}
}
