package queue

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

func TestPackageInstallsWithItsDependencies(t *testing.T) {
	f := newFixture(t)
	c, _ := f.s.d.Client()
	f.s.d.Closure = func(_ context.Context, game string, roots []thunderstore.Ref) ([]thunderstore.Resolved, error) {
		if game != "riskofrain2" || len(roots) != 2 || roots[0].Name != "Mod" || roots[1].Name != "Other" {
			t.Errorf("closure asked %q %+v", game, roots)
		}
		url := c.BaseURL + "/cdn/file.zip"
		return []thunderstore.Resolved{
			{Namespace: "BepInEx", Name: "BepInExPack", Version: "5.4.2100", URL: url},
			{Namespace: "Me", Name: "Held", Version: "2.0.0", URL: url},
			{Namespace: "Me", Name: "Mod", Version: "1.0.0", URL: url, Icon: "https://ccdn.thunderstore.io/live/repository/icons/Me-Mod-1.0.0.png", Category: "Tools"},
			{Namespace: "Me", Name: "Other", Version: "1.0.0", URL: url},
		}, nil
	}
	f.s.d.Held = func(_, _, pkg string) string {
		if pkg == "Me-Held" {
			return "2.1.0"
		}
		return ""
	}
	f.s.d.InstallPackage = f.s.d.Install
	f.start()
	if _, err := f.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "Me-Mod"}, {Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "Me-Other"}}); err != nil {
		t.Fatal(err)
	}
	st := f.wait("both done", func(st State) bool {
		return len(st.Items) == 2 && st.Items[0].State == StateDone && st.Items[1].State == StateDone
	})
	if st.Items[0].Package != "Me-Mod" || st.Items[1].Package != "Me-Other" || st.Items[0].Kind != KindInstall {
		t.Errorf("a held dependency, the loader pack or a root was mishandled: %+v", st.Items)
	}
	if len(f.installs) != 2 || f.installs[0].Kind != profile.KindThunderstore || f.installs[0].Picture != "https://ccdn.thunderstore.io/live/repository/icons/Me-Mod-1.0.0.png" || f.installs[0].Category != "Tools" {
		t.Errorf("installed %+v", f.installs)
	}
	if _, err := f.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "not a package"}}); err == nil {
		t.Error("a malformed package was queued")
	}
}
