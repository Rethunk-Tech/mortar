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
	req := requestFor("lethal-company", "p1", m)
	if req.Package != "Alice-MoreCompany" || req.Version != "1.2.3" || req.Kind != queue.KindInstall || req.ModID != 0 {
		t.Errorf("request %+v", req)
	}
	if had := r.thunderstore(share.Ref{Package: "alice-had", Version: "1.0.0"}); had.State != StateInstalled {
		t.Errorf("an installed package is %s", had.State)
	}
	if modToken(m) != "t:alice-morecompany" {
		t.Errorf("token %q", modToken(m))
	}
}
