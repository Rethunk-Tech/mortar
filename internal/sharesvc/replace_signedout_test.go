package sharesvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

func TestReplaceSignedOutKeepsTheProfilesMods(t *testing.T) {
	s, _ := newService(t, false)
	prof := mineProfile(t, s)
	if _, err := s.d.Profiles.InstallSource(t.Context(), "stardew", prof.ID, modZip(t, "A.Drop"), profile.Source{Kind: profile.KindNexus, ModID: 600, FileID: 6}); err != nil {
		t.Fatal(err)
	}
	text := link(t, "Cozy", share.Ref{ModID: 900, FileID: 9})
	pv, err := s.PreviewLink(t.Context(), "stardew", text, prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Replace(t.Context(), "stardew", pv.Session, prof.ID, nil, false)
	got, _ := s.find("stardew", prof.ID)
	if err == nil || len(got.Entries) == 0 {
		t.Fatalf("Replace err %v, entries %d: a refused replace must keep the profile mods", err, len(got.Entries))
	}
}
