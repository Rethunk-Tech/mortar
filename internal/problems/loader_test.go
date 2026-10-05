package problems

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestLoaderFailuresAttributeLogFindingsToPackages(t *testing.T) {
	log, err := fsx.ReadFile(filepath.Join("..", "loader", "bepinex5", "testdata", "logoutput-missing-dependency.log"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	testfs.WriteFile(t, dir, "BepInEx/LogOutput.log", string(log))
	boombox := inst("boombox", "YoutubeBoombox", "1.0", true)
	known := map[string]framework.Mod{"youtube boombox 1.5.0": boombox}
	for name, owners := range map[string]map[string]framework.Mod{"known": known, "unknown": {}} {
		got := loaderFailures(bepinex5.Loader{}, loader.ProfileView{Dir: dir}, func() map[string]framework.Mod { return owners })
		if len(got) != 2 || got[0].Plugin != "Youtube Boombox 1.5.0" || got[0].Kind != bepinex5.KindMissingDependency {
			t.Fatalf("%s: failures = %+v", name, got)
		}
		wantKey := ""
		if name == "known" {
			wantKey = "boombox"
		}
		if got[0].Key != wantKey || got[0].Name != boombox.Name && wantKey != "" {
			t.Fatalf("%s: owner = %q %q", name, got[0].Key, got[0].Name)
		}
	}
}
