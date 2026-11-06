package bepinex5

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testthunderstore"
)

func TestReleasesComeFromTheGamesThunderstoreCommunity(t *testing.T) {
	pack, err := safefs.ReadFile(buildPack(t, "4.3.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	srv := testthunderstore.Serve(t, "some-game", `[{"name":"BepInExPack","owner":"BepInEx","versions":[
		{"version_number":"5.4.2100"},{"version_number":"5.4.2304"},{"version_number":"6.0.0"}]}]`,
		map[string][]byte{"/package/download/BepInEx/BepInExPack/5.4.2304/": pack}, true)
	l := Loader{Index: &thunderstore.Driver{HTTP: srv.Client(), URL: srv.URL, CacheDir: t.TempDir()}, HTTP: srv.Client()}
	var _ loader.Releases = l
	g := components.GameInfo{Name: "Some Game", Sources: []components.GameSource{{ID: "thunderstore", Key: "some-game"}}}
	ctx := t.Context()

	if v, err := l.Versions(ctx, g); err != nil || !slices.Equal(v, []string{"5.4.2304", "5.4.2100"}) {
		t.Fatalf("versions %v %v", v, err)
	}
	if v, err := l.Latest(ctx, g); err != nil || v != "5.4.2304" {
		t.Fatalf("latest %q %v", v, err)
	}
	dst := filepath.Join(t.TempDir(), "pack.zip")
	if err := l.Fetch(ctx, g, "5.4.2304", dst); err != nil {
		t.Fatal(err)
	}
	if got, _ := safefs.ReadFile(dst); string(got) != string(pack) {
		t.Fatal("downloaded file differs")
	}
	g.Loaders = []components.GameLoader{{ID: ID, Major: 6}}
	if v, err := l.Versions(ctx, g); err != nil || !slices.Equal(v, []string{"6.0.0"}) {
		t.Fatalf("BepInEx 6 versions %v %v", v, err)
	}
	if _, err := l.Versions(ctx, components.GameInfo{Name: "No Community"}); err == nil {
		t.Fatal("a game without a Thunderstore community must be refused")
	}
}
