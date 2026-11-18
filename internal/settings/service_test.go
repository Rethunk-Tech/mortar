package settings

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

type stubHost struct {
	winhost.Host
	open, save string
	openErr    error
	emits      int
}

func (h *stubHost) Emit(string, any)                        { h.emits++ }
func (h *stubHost) OpenFile(winhost.Dialog) (string, error) { return h.open, h.openErr }
func (h *stubHost) SaveFile(winhost.Dialog) (string, error) { return h.save, nil }

func TestServiceSettersPersist(t *testing.T) {
	store, _ := open(t)
	svc := NewService(store)
	host := &stubHost{}
	svc.App = host

	if err := errors.Join(
		svc.SetLanguage("en"),
		svc.SetAccent("moss"),
		svc.SetKeepInTray(true),
		svc.SetLanSharing(true),
		svc.SetLanPort(1234),
		svc.SetIncludeBetaReleases(true),
		svc.SetIncludePrereleaseModVersions(true),
		svc.SetCheckOnlyEnabledMods(true),
		svc.SetEnableModsWhenInstalled(false),
		svc.SetBackground(BackgroundSolid),
		svc.RememberNexusDownloadServers([]string{"cdn", "cdn"}),
		svc.SetNexusPreferredDownloadServer("cdn"),
		svc.SetNxmRedirectOtherGames(false),
		svc.SetListSort("author", "desc"),
		svc.SetListGroupBy("tag"),
		svc.SetTipsSeen([]string{"mods", "saves"}),
		svc.SetLoaderToastAt("stardew", "smapi", "2026-01-01T00:00:00Z"),
		svc.SetLastGame("stardew"),
		svc.SetLastProfile("stardew", "p1"),
		svc.SetShortcuts(nil),
	); err != nil {
		t.Fatal(err)
	}
	got := svc.Get()
	if got.Language != "en" || got.Accent != "moss" || !got.KeepInTray || !got.LanSharing || got.LanPort != 1234 ||
		!got.IncludeBetaReleases || !got.IncludePrereleaseModVersions || !got.CheckOnlyEnabledMods ||
		got.EnableModsWhenInstalled == nil || *got.EnableModsWhenInstalled || got.Background != BackgroundSolid ||
		got.NexusPreferredDownloadServer != "cdn" || !slices.Contains(got.NexusSeenDownloadServers, "cdn") ||
		got.NxmRedirectOtherGames == nil || *got.NxmRedirectOtherGames || got.ListSortColumn != "author" ||
		got.ListSortDir != "desc" || got.ListGroupBy != "tag" || !slices.Equal(got.TipsSeen, []string{"mods", "saves"}) ||
		got.LoaderPrefs[LoaderKey("stardew", "smapi")].ToastAt != "2026-01-01T00:00:00Z" || got.LastGame != "stardew" ||
		got.LastProfile["stardew"] != "p1" || got.Shortcuts == nil {
		t.Fatalf("setters = %+v", got)
	}
	if host.emits == 0 {
		t.Fatal("setters did not emit")
	}
	if specs := svc.PrefSpecs(); len(specs) == 0 {
		t.Fatal("PrefSpecs empty")
	}
	if _, err := svc.InstalledFonts(); err != nil {
		t.Fatal(err)
	}
	_ = svc.SystemAccent()
	if svc.AntivirusProduct().Text == "" {
		t.Fatal("AntivirusProduct empty")
	}
	if svc.CorruptSettingsPath() != "" {
		t.Fatalf("CorruptSettingsPath = %q", svc.CorruptSettingsPath())
	}
	if err := svc.ShowCorruptSettings(); err != nil {
		t.Fatal(err)
	}
}

func TestServiceExtraModsDevProjectsAndLaunchers(t *testing.T) {
	store, _ := open(t)
	svc := NewService(store)
	p := DevProject{Dir: "/src", Name: "M", Profile: "dev", Output: "/out", Untrusted: true}
	if err := svc.AddDevProject("stardew", p); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(
		svc.SetExtraModsProfile("stardew", "/out", "p1"),
		svc.SetDevProjectWatch("stardew", p.Dir, true),
		svc.TrustDevProject("stardew", p.Dir),
	); err != nil {
		t.Fatal(err)
	}
	gp := svc.Get().GamePrefs("stardew")
	if gp.ExtraModsProfiles["/out"] != "p1" || len(gp.DevProjects) != 1 || !gp.DevProjects[0].Watch || gp.DevProjects[0].Untrusted {
		t.Fatalf("game prefs = %+v", gp)
	}
	if err := svc.SetExtraModsProfile("stardew", "/out", ""); err != nil {
		t.Fatal(err)
	}
	if _, ok := svc.Get().GamePrefs("stardew").ExtraModsProfiles["/out"]; ok {
		t.Fatal("empty profile should drop the extra-mods mapping")
	}
	if err := svc.ReplaceDevProject("stardew", p, DevProject{Dir: "/src2", Name: "N", Profile: "dev2", Output: "/out"}); err != nil {
		t.Fatal(err)
	}

	svc.ValidateLauncherRoot = func(launcher, dir string) error {
		if dir == "/bad" {
			return os.ErrInvalid
		}
		return nil
	}
	if err := svc.AddLauncherRoot("steam", "/bad"); err == nil {
		t.Fatal("invalid launcher root accepted")
	}
	if err := svc.AddLauncherRoot("steam", "/a"); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddLauncherRoot("steam", "/a"); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddLauncherRoot("steam", "/b"); err != nil {
		t.Fatal(err)
	}
	if got := svc.Get().LauncherRoots["steam"]; !slices.Equal(got, []string{"/a", "/b"}) {
		t.Fatalf("roots = %v", got)
	}
	if err := svc.RemoveLauncherRoot("steam", "/a"); err != nil {
		t.Fatal(err)
	}
	if err := svc.RemoveLauncherRoot("steam", "/b"); err != nil {
		t.Fatal(err)
	}
	if _, ok := svc.Get().LauncherRoots["steam"]; ok {
		t.Fatal("empty launcher list should drop the key")
	}

	if err := svc.SetGameStore("stardew", "nope"); err == nil {
		t.Fatal("unknown store accepted")
	}
	if err := svc.SetGameStore("stardew", "steam"); err != nil || svc.Get().GameStores["stardew"] != "steam" {
		t.Fatalf("set store: %v %v", err, svc.Get().GameStores)
	}
	if err := svc.SetGameStore("stardew", ""); err != nil {
		t.Fatal(err)
	}
	if _, ok := svc.Get().GameStores["stardew"]; ok {
		t.Fatal("empty store should drop the key")
	}
}

func TestServiceDialogsAndImport(t *testing.T) {
	store, _ := open(t)
	svc := NewService(store)
	host := &stubHost{}
	svc.App = host

	if err := svc.ChooseBackgroundImage(); err != nil {
		t.Fatal(err)
	}
	host.openErr = os.ErrPermission
	if err := svc.ChooseBackgroundImage(); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("open err = %v", err)
	}
	host.openErr = nil
	host.open = "/wall.png"
	svc.ValidateImage = func(path string) error {
		if path != "/wall.png" {
			return os.ErrInvalid
		}
		return nil
	}
	if err := svc.ChooseBackgroundImage(); err != nil || svc.Get().BackgroundImage != "/wall.png" {
		t.Fatalf("choose image: %v %q", err, svc.Get().BackgroundImage)
	}

	host.open = ""
	if err := svc.ChooseGameFolder("stardew"); err != nil {
		t.Fatal(err)
	}
	host.open = "/game"
	if err := svc.ChooseGameFolder("stardew"); err != nil || svc.Get().GameFolders["stardew"] != filepath.Clean("/game") {
		t.Fatalf("choose folder: %v %v", err, svc.Get().GameFolders)
	}

	host.save = filepath.Join(t.TempDir(), "mortar-settings.json")
	path, err := svc.ExportSettings()
	if err != nil || path != host.save {
		t.Fatalf("export: %q %v", path, err)
	}
	foreign := filepath.Join(t.TempDir(), "in.json")
	if err := os.WriteFile(foreign, []byte(`{"version":1,"global":{"accent":"sky"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewImport(foreign)
	if err != nil || len(got.Sections) == 0 {
		t.Fatalf("preview: %+v %v", got, err)
	}
	if err := svc.ApplyImport(foreign, []string{SectionAppearance}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PreviewImport(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("missing import")
	}
	if err := svc.ApplyImport(filepath.Join(t.TempDir(), "missing.json"), []string{SectionAppearance}); err == nil {
		t.Fatal("missing apply")
	}
	host.open = path
	if picked, err := svc.PickImportFile(); err != nil || picked != path {
		t.Fatalf("pick: %q %v", picked, err)
	}
}
