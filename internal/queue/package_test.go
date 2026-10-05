package queue

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

func TestPackageInstallsWithItsDependencies(t *testing.T) {
	f := newFixture(t)
	c, _ := f.s.d.Client()
	f.s.d.Closure = func(_ context.Context, game string, roots []thunderstore.Ref) ([]thunderstore.Resolved, error) {
		if game != "riskofrain2" || len(roots) != 1 || roots[0].Name != "Mod" {
			t.Errorf("closure asked %q %+v", game, roots)
		}
		url := c.BaseURL + "/cdn/file.zip"
		return []thunderstore.Resolved{
			{Namespace: "BepInEx", Name: "BepInExPack", Version: "5.4.2100", URL: url},
			{Namespace: "Me", Name: "Mod", Version: "1.0.0", URL: url},
		}, nil
	}
	f.s.d.InstallPackage = f.s.d.Install
	f.start()
	if _, err := f.s.Add([]Request{{Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "Me-Mod"}}); err != nil {
		t.Fatal(err)
	}
	st := f.wait("both done", func(st State) bool {
		return len(st.Items) == 2 && st.Items[0].State == StateDone && st.Items[1].State == StateDone
	})
	if st.Items[0].Kind != KindDependency || st.Items[1].Kind != KindInstall {
		t.Errorf("kinds %s %s", st.Items[0].Kind, st.Items[1].Kind)
	}
	if len(f.installs) != 2 || f.installs[0].Kind != sourceThunderstore {
		t.Errorf("installed %+v", f.installs)
	}
	if _, err := f.s.Add([]Request{{Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "not a package"}}); err == nil {
		t.Error("a malformed package was queued")
	}
}
