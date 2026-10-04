package profile

import (
	"testing"
	"time"
)

func TestChangesSinceDecodesHistoryOnce(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
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
