package queue

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/curseforge"
)

// cfDirect is queue.ResolveDirect's CurseForge branch against a fake API whose files are served from cdn.
func cfDirect(t *testing.T, cdn string) func(context.Context, string, string, string, []string) (DirectFile, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mods/998265":
			_, _ = w.Write([]byte(`{"data":{"id":998265,"name":"Bush","links":{"websiteUrl":"https://www.curseforge.com/stardewvalley/mods/bush"},"allowModDistribution":true}}`))
		case "/mods/998265/files/777":
			_, _ = w.Write([]byte(`{"data":{"id":777,"modId":998265,"isAvailable":true,"displayName":"Bush 1.0.1.zip","fileName":"bush-1.0.1.zip","releaseType":1,"fileDate":"2026-02-01T00:00:00Z","fileLength":13,"downloadUrl":"` + cdn + `/cdn/file.zip"}}`))
		case "/mods/998266":
			_, _ = w.Write([]byte(`{"data":{"id":998266,"name":"Shut","links":{"websiteUrl":"https://www.curseforge.com/stardewvalley/mods/shut"},"allowModDistribution":false}}`))
		case "/mods/998266/files/881":
			_, _ = w.Write([]byte(`{"data":{"id":881,"modId":998266,"isAvailable":true,"displayName":"Shut 2.0.zip","fileName":"shut-2.0.zip","releaseType":1,"downloadUrl":"` + cdn + `/cdn/file.zip"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	driver := curseforge.Driver{
		URL: srv.URL, Key: func() string { return "k" },
		GameSource: func(string) (components.GameSource, bool) {
			return components.GameSource{ID: "curseforge", Key: "4643", GameID: 669}, true
		},
	}
	return func(ctx context.Context, _, id, version string, _ []string) (DirectFile, error) {
		r, err := driver.Resolve(ctx, id, version)
		if err != nil {
			return DirectFile{}, err
		}
		fileID, _ := strconv.Atoi(r.VersionID)
		return DirectFile{ID: id, Name: r.Name, Version: r.Version, FileID: fileID, FileName: r.FileName, URL: r.URL, SizeKB: r.Size >> 10}, nil
	}
}

func TestACurseForgeUpdateOfANexusEntryDownloadsAndInstallsFromCurseForge(t *testing.T) {
	f := newFixture(t)
	c, _ := f.s.d.Client()
	f.s.d.Direct = cfDirect(t, c.BaseURL)
	f.s.d.InstallPackage = f.s.d.Install
	f.start()
	// The request problems.Update{Package, PackageSource, PackageVersion: file id} becomes.
	if _, err := f.s.Add(t.Context(), []Request{{
		Kind: KindUpdate, Game: "stardew", Profile: "p1", Name: "Bush", CurrentKey: "nexus-20619-123175",
		Package: "998265", Source: "curseforge", Version: "777",
	}}); err != nil {
		t.Fatal(err)
	}
	st := f.wait("done", f.item(StateDone))
	if it := st.Items[0]; it.Package != "998265" || it.FileName != "bush-1.0.1.zip" || it.State != StateDone {
		t.Fatalf("item = %+v", it)
	}
	if len(f.installs) != 1 || f.installs[0].Kind != profile.KindCurseForge {
		t.Fatalf("installs = %+v, want one CurseForge install", f.installs)
	}
}

func TestACurseForgeUpdateOfAForbiddenProjectOpensItsPageInsteadOfDownloading(t *testing.T) {
	f := newFixture(t)
	c, _ := f.s.d.Client()
	f.s.d.Direct = cfDirect(t, c.BaseURL)
	f.s.d.InstallPackage = f.s.d.Install
	f.s.d.OpenURL = func(u string) error {
		f.mu.Lock()
		f.opened = append(f.opened, u)
		f.mu.Unlock()
		return nil
	}
	f.start()
	_, err := f.s.Add(t.Context(), []Request{{
		Kind: KindUpdate, Game: "stardew", Profile: "p1", Name: "Shut", CurrentKey: "nexus-1-2",
		Package: "998266", Source: "curseforge", Version: "881",
	}})
	if err == nil {
		t.Fatal("a forbidden file was queued for download")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.opened) != 1 || f.opened[0] != "https://www.curseforge.com/stardewvalley/mods/shut" || len(f.installs) != 0 {
		t.Fatalf("opened %v, installs %v", f.opened, f.installs)
	}
}
