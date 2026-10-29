package settings

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestExportOmitsSecretsAndMachineFields(t *testing.T) {
	s := Defaults()
	s.Language = "en"
	s.Accent = "moss"
	s.NexusName = "Farmer"
	s.NexusUserID = 99
	s.NexusPremium = true
	s.GameFolders = map[string]string{"stardew": "/games/Stardew Valley"}
	s.GameStores = map[string]string{"stardew": "steam"}
	s.Loaders = map[string]string{"stardew": "4.5.2"}
	s.LastProfile = map[string]string{"stardew": "abc"}
	s.LastPlayed = map[string]Played{"stardew": {Profile: "abc", At: "2026-01-01T00:00:00Z"}}
	s.BackgroundImage = "/home/u/wall.png"
	s.OverlayEnabled = true
	s.OverlayPort = 9000
	s.OverlayToken = "overlay-secret-token"
	b, err := MarshalExport(s)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["version"] != float64(exportVersion) {
		t.Fatalf("version = %v", m["version"])
	}
	global, _ := m["global"].(map[string]any)
	if global["accent"] != "moss" {
		t.Fatalf("accent = %v", global["accent"])
	}
	if global["language"] != "en" {
		t.Fatalf("language = %v", global["language"])
	}
	for _, k := range []string{
		"nexusName", "nexusUserId", "nexusPremium", "gameFolders", "gameStores",
		"lastProfile", "lastPlayed", "backgroundImage", "dismissed",
		"nxmHandled", "nxmPreviousHandlers", "nxmAsked", "lanStarted",
		"overlayToken", "overlayEnabled", "overlayPort",
	} {
		if strings.Contains(string(b), `"`+k+`"`) {
			t.Fatalf("exported %s: %s", k, b)
		}
	}
	if _, ok := global["loaders"]; ok {
		t.Fatalf("installed loader versions exported: %s", b)
	}
	raw := string(b)
	for _, secret := range []string{"Farmer", "/games/Stardew", "abc", "/home/u/wall", "overlay-secret-token"} {
		if strings.Contains(raw, secret) {
			t.Fatalf("export still holds %q: %s", secret, raw)
		}
	}
}

func TestLanguagePortableRoundTrip(t *testing.T) {
	p, present, err := ParseExport([]byte(`{"version":1,"global":{"language":"en"}}`))
	if err != nil {
		t.Fatal(err)
	}
	cur := Defaults()
	ApplyExport(&cur, p, present)
	if cur.Language != "en" {
		t.Fatalf("language = %q", cur.Language)
	}
}

func TestImportIgnoresUnknownAndSanitizesLikeLoad(t *testing.T) {
	raw := []byte(`{"version":1,"global":{"accent":"neon","mystery":true,"gameFolders":{"stardew":"/nope"},"storeRetentionDays":30},"sources":{"nexus":{"name":"x"}}}`)
	p, present, err := ParseExport(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Accent != "sand" {
		t.Fatalf("invalid accent as on load: %q", p.Accent)
	}
	if p.StoreRetentionDays != 30 {
		t.Fatalf("storeRetentionDays = %d", p.StoreRetentionDays)
	}
	cur := Defaults()
	cur.NexusName = "keep"
	cur.NexusUserID = 7
	cur.GameFolders["stardew"] = "/keep"
	cur.Loaders["smapi"] = "4.5.2"
	cur.LastProfile["stardew"] = "abc"
	ApplyExport(&cur, p, present)
	if cur.NexusName != "keep" || cur.NexusUserID != 7 {
		t.Fatalf("nexus display changed: %+v", cur)
	}
	if cur.GameFolders["stardew"] != "/keep" || cur.Loaders["smapi"] != "4.5.2" || cur.LastProfile["stardew"] != "abc" {
		t.Fatalf("machine fields changed: %+v", cur)
	}
	if cur.StoreRetentionDays != 30 {
		t.Fatalf("storeRetentionDays = %d", cur.StoreRetentionDays)
	}
	if cur.Accent != "sand" {
		t.Fatalf("accent = %q", cur.Accent)
	}
}

func TestImportRejectsMissingOrUnknownVersion(t *testing.T) {
	if _, _, err := ParseExport([]byte(`{"global":{"accent":"moss"}}`)); err == nil {
		t.Fatal("missing version accepted")
	}
	if _, _, err := ParseExport([]byte(`{"version":2,"global":{"accent":"moss"}}`)); err == nil {
		t.Fatal("version 2 accepted")
	}
}

func TestImportPreviewGroupsBySectionAndNotesUnknown(t *testing.T) {
	raw := []byte(`{"version":1,"global":{"accent":"sky","storeRetentionDays":60,"parallelDownloads":2,"mystery":1},"sources":{"nexus":{"name":"x"}}}`)
	got, err := PreviewImport(Defaults(), raw)
	if err != nil {
		t.Fatal(err)
	}
	var secs []string
	for _, s := range got.Sections {
		secs = append(secs, s.Section)
	}
	if strings.Join(secs, ",") != "appearance,downloads,storage" {
		t.Fatalf("sections = %+v", got.Sections)
	}
	if c := got.Sections[0].Changes[0]; c.Key != "accent" || c.To != "sky" || c.Label != "Accent" {
		t.Fatalf("change = %+v", c)
	}
	if strings.Join(got.Ignored, ",") != "mystery,nexusName" {
		t.Fatalf("ignored = %v", got.Ignored)
	}
	if _, err := PreviewImport(Defaults(), []byte(`{"version":2}`)); err == nil {
		t.Fatal("version 2 accepted")
	}
}

func TestApplyImportTakesOnlyChosenSections(t *testing.T) {
	raw := []byte(`{"version":1,"global":{"accent":"sky","storeRetentionDays":60}}`)
	cur := Defaults()
	if err := ApplyImport(&cur, raw, []string{SectionStorage}); err != nil {
		t.Fatal(err)
	}
	if cur.StoreRetentionDays != 60 || cur.Accent != Defaults().Accent {
		t.Fatalf("storeRetentionDays=%d accent=%q", cur.StoreRetentionDays, cur.Accent)
	}
	if err := ApplyImport(&cur, raw, nil); err != nil || cur.Accent != Defaults().Accent {
		t.Fatalf("no sections changed accent: %v", err)
	}
}

// A field naming an account, a token, a machine path or a command on this computer must stay out of an export.
func TestPortableFieldsNameNoAccountOrMachineState(t *testing.T) {
	for _, k := range []string{
		"nexusUserId", "nexusName", "nexusPremium", "itchName", "overlayToken", "overlayEnabled", "overlayPort",
		"gameFolders", "gameStores", "launcherRoots", "launchersConfirmed", "loaders", "lastProfile", "lastPlayed", "hiddenGames",
		"backgroundImage", "lanStarted", "lanAddresses", "antivirusSocket", "antivirusCommand",
		"nxmHandled", "nxmPreviousHandlers", "nxmAsked", "nxmPreviousName", "alvrProgram", "autoInstallFolders",
		"lastModUpdateDigest", "lastModUpdateDigestAt", "dismissed", "shortcuts",
		"downloadFolder", "watchFolders", "syncFolder", "vortexFolder", "mo2Folder", "packFolder", "lanAllowAnyAddress",
	} {
		if slices.Contains(portableFields, k) {
			t.Errorf("portableFields exports %s", k)
		}
	}
}

func TestImportKeepsFoldersAndNetworkLocalAndNeverDisablesAntivirus(t *testing.T) {
	cur := Defaults()
	cur.DownloadFolder = "/home/u/dl"
	cur.LanAllowAnyAddress = false
	cur.Antivirus = AntivirusClamd
	file := `{"version":1,"global":{"downloadFolder":"/etc","watchFolders":"/etc","syncFolder":"/etc","vortexFolder":"/etc","packFolder":"/etc","lanSharing":true,"lanAllowAnyAddress":true,"antivirus":"off"}}`
	exported, present, err := ParseExport([]byte(file))
	if err != nil {
		t.Fatal(err)
	}
	ApplyExport(&cur, exported, present)
	if cur.DownloadFolder != "/home/u/dl" || cur.WatchFolders != "" || cur.SyncFolder != "" || cur.VortexFolder != "" || cur.PackFolder != "" || cur.LanAllowAnyAddress {
		t.Fatalf("local-only field imported: %+v", cur)
	}
	if cur.Antivirus != AntivirusClamd {
		t.Fatalf("antivirus = %q", cur.Antivirus)
	}
}

func TestImportKeepsLocalProgressAndAnExportOmitsIt(t *testing.T) {
	other := Defaults()
	other.Games = map[string]*GameSettings{"stardew": {
		LastDownloadsSeen: 9999999999999, LastSweepGameVersion: "1.6", NxmDefaultProfile: "theirs",
		DevProjects: []DevProject{{Dir: "/home/alice/secret-project", Name: "X"}}, SyncOwnFolder: "/home/alice/Sync",
		LaunchPresetTemplates: []LaunchPresetTemplate{{Name: "p", Env: "API_TOKEN=hunter2", Prefix: "/opt/x"}},
	}}
	other.LoaderPrefs = map[string]LoaderPrefs{"stardew/smapi": {Pin: "4.0", LastSweepVersion: "9", ToastAt: "2026-01-01T00:00:00Z"}}
	b, err := MarshalExport(other)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"/home/alice", "hunter2", "/opt/x", "theirs", "9999999999999"} {
		if strings.Contains(string(b), w) {
			t.Errorf("export carries %q", w)
		}
	}
	if other.Games["stardew"].SyncOwnFolder == "" || other.Games["stardew"].LaunchPresetTemplates[0].Env == "" {
		t.Fatal("exporting changed the settings it was given")
	}
	// A file from before the export fix still carries the games' values.
	old, _ := json.Marshal(map[string]any{"version": 1, "games": other.Games})
	cur0 := Defaults()
	cur0.Games = map[string]*GameSettings{"stardew": {LastDownloadsSeen: 100, NxmDefaultProfile: "mine", LastSweepGameVersion: "1.5"}}
	if err := ApplyImport(&cur0, old, ImportSections()); err != nil {
		t.Fatal(err)
	}
	if g := cur0.GamePrefs("stardew"); g.LastDownloadsSeen != 100 || g.NxmDefaultProfile != "mine" || g.LastSweepGameVersion != "1.5" {
		t.Errorf("games = %+v", g)
	}
	cur := Defaults()
	cur.LoaderPrefs = map[string]LoaderPrefs{"stardew/smapi": {LastSweepVersion: "8"}}
	if err := ApplyImport(&cur, b, ImportSections()); err != nil {
		t.Fatal(err)
	}
	if p := cur.LoaderPrefs["stardew/smapi"]; p.Pin != "4.0" || p.LastSweepVersion != "8" || p.ToastAt != "" {
		t.Errorf("loader prefs = %+v", p)
	}
}

func TestImportedScannerModeNeedsALocalScanner(t *testing.T) {
	other := Defaults()
	other.Antivirus = AntivirusCommand
	b, _ := MarshalExport(other)
	cur := Defaults()
	cur.Antivirus = AntivirusAutomatic
	if err := ApplyImport(&cur, b, ImportSections()); err != nil {
		t.Fatal(err)
	}
	if cur.Antivirus != AntivirusAutomatic {
		t.Fatalf("antivirus = %q", cur.Antivirus)
	}
	cur.AntivirusCommand = "scan {path}"
	if err := ApplyImport(&cur, b, ImportSections()); err != nil || cur.Antivirus != AntivirusCommand {
		t.Fatalf("antivirus = %q, %v", cur.Antivirus, err)
	}
}
