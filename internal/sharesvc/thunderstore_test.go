package sharesvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

func TestThunderstoreRefsQueueAsPackages(t *testing.T) {
	r := &resolver{game: "lethal-company", target: []profile.Entry{
		{Source: profile.Source{Kind: profile.KindThunderstore, Name: "Alice-Had", Version: "1.0.0"}},
	}}
	m := r.thunderstore(share.Ref{Package: "Alice-MoreCompany", Version: "1.2.3"})
	if m.State != StateDownload || m.Name != "MoreCompany" || m.Author != "Alice" || m.PageURL == "" {
		t.Fatalf("mod %+v", m)
	}
	m.Implicit = true
	req := requestFor("lethal-company", "p1", m)
	if req.Package != "Alice-MoreCompany" || req.Version != "1.2.3" || req.Kind != queue.KindInstall || req.ModID != 0 || !req.Implicit {
		t.Errorf("request %+v", req)
	}
	if had := r.thunderstore(share.Ref{Package: "alice-had", Version: "1.0.0"}); had.State != StateInstalled {
		t.Errorf("an installed package is %s", had.State)
	}
	if modToken(m) != "t:alice-morecompany" {
		t.Errorf("token %q", modToken(m))
	}
}

func TestThunderstoreRowCarriesTheIndexIcon(t *testing.T) {
	const icon = "https://gcdn.thunderstore.io/live/repository/icons/Alice-MoreCompany-1.2.3.png"
	r := &resolver{game: "lethal-company", packageIcons: func(names []string) map[string]string {
		if len(names) != 2 {
			t.Errorf("icons asked per ref, not once: %v", names)
		}
		return map[string]string{"alice-morecompany": icon}
	}}
	refs := []share.Ref{{Package: "Alice-MoreCompany", Version: "1.2.3"}, {Package: "Bob-NoIcon"}}
	r.checkPackages(t.Context(), refs)
	if got := r.thunderstore(refs[0]).Icon; got != icon {
		t.Errorf("icon %q", got)
	}
	if got := r.thunderstore(refs[1]).Icon; got != "" {
		t.Errorf("a package the index lacks has icon %q", got)
	}
}

func TestAnInstalledRowShowsThePictureItsEntryHas(t *testing.T) {
	const pic = "https://staticdelivery.nexusmods.com/mods/1/images/7/a.png"
	r := &resolver{target: []profile.Entry{{Source: profile.Source{Kind: profile.KindPatreon, Name: "123", Picture: pic}}}}
	mods, _ := r.resolve(t.Context(), []share.Ref{{Patreon: "123"}, {Patreon: "456"}})
	if len(mods) != 2 || mods[0].Icon != pic || mods[1].Icon != "" {
		t.Fatalf("mods %+v", mods)
	}
}
