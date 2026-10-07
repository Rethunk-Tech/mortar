package problems

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/curseforge"
)

// cfServer serves two CurseForge projects: 998265, whose newest file is 1.0.1 (the installed mod is at 1.0.0), and
// 998266, whose author forbids downloads outside CurseForge.
func cfServer(t *testing.T) curseforge.Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mods/998265":
			_, _ = w.Write([]byte(`{"data":{"id":998265,"name":"Bush","links":{"websiteUrl":"https://www.curseforge.com/stardewvalley/mods/bush"},"allowModDistribution":true}}`))
		case "/mods/998265/files":
			_, _ = w.Write([]byte(`{"data":[
				{"id":776,"modId":998265,"isAvailable":true,"displayName":"Bush 1.0.0.zip","fileName":"bush-1.0.0.zip","releaseType":1,"fileDate":"2026-01-01T00:00:00Z","downloadUrl":"https://edge/bush-1.0.0.zip"},
				{"id":777,"modId":998265,"isAvailable":true,"displayName":"Bush 1.0.1.zip","fileName":"bush-1.0.1.zip","releaseType":1,"fileDate":"2026-02-01T00:00:00Z","downloadUrl":"https://edge/bush-1.0.1.zip"}]}`))
		case "/mods/998266":
			_, _ = w.Write([]byte(`{"data":{"id":998266,"name":"Shut","links":{"websiteUrl":"https://www.curseforge.com/stardewvalley/mods/shut"},"allowModDistribution":false}}`))
		case "/mods/998266/files":
			_, _ = w.Write([]byte(`{"data":[{"id":881,"modId":998266,"isAvailable":true,"displayName":"Shut 2.0.zip","fileName":"shut-2.0.zip","releaseType":1,"fileDate":"2026-02-01T00:00:00Z","downloadUrl":"https://edge/shut-2.0.zip"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return curseforge.Driver{
		URL: srv.URL, Key: func() string { return "k" },
		GameSource: func(string) (components.GameSource, bool) {
			return components.GameSource{ID: "curseforge", Key: "4643", GameID: 669}, true
		},
	}
}

func nexusModWithCFKey(project, version string) (framework.Mod, Update) {
	x := inst("nexus-20619-123175", "Example.Bush", version, true)
	x.SourceKind, x.UpdateKeys = profile.KindNexus, []string{"Nexus:20619", "CurseForge:" + project}
	return x, Update{Key: x.Key, ID: x.ModID(), Installed: version, Version: "1.0.1", Source: "CurseForge", URL: "https://www.curseforge.com/stardewvalley/mods/bush"}
}

func TestANexusModIsOfferedItsNewerCurseForgeFileFromTheRealDriver(t *testing.T) {
	x, up := nexusModWithCFKey("998265", "1.0.0")
	got := (&Service{}).switchUpdates(t.Context(), cfServer(t), []framework.Mod{x}, []Update{up})
	if len(got) != 1 {
		t.Fatalf("updates = %+v", got)
	}
	u := got[0]
	if u.Package != "998265" || u.PackageSource != profile.KindCurseForge || u.PackageVersion != "777" || u.Version != "1.0.1" ||
		!u.Switch || u.FromSource != "Nexus Mods" || u.NotDistributable {
		t.Fatalf("update = %+v", u)
	}
}

func TestAForbiddenCurseForgeProjectIsAPageLinkNotADownload(t *testing.T) {
	x, up := nexusModWithCFKey("998266", "1.0.0")
	got := (&Service{}).switchUpdates(t.Context(), cfServer(t), []framework.Mod{x}, []Update{up})
	if len(got) != 1 || !got[0].NotDistributable || got[0].Package != "" || got[0].PackageVersion != "" || got[0].URL == "" {
		t.Fatalf("updates = %+v", got)
	}
}

func TestACurseForgeFileNotNewerThanInstalledIsNotOffered(t *testing.T) {
	x, up := nexusModWithCFKey("998265", "1.0.1")
	up.Installed = "1.0.1"
	if got := (&Service{}).switchUpdates(t.Context(), cfServer(t), []framework.Mod{x}, []Update{up}); len(got) != 0 {
		t.Fatalf("updates = %+v, want none", got)
	}
}
