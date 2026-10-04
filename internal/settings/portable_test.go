package settings

import (
	"encoding/json"
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
	if m["accent"] != "moss" {
		t.Fatalf("accent = %v", m["accent"])
	}
	if m["language"] != "en" {
		t.Fatalf("language = %v", m["language"])
	}
	for _, k := range []string{
		"nexusName", "nexusUserId", "nexusPremium", "gameFolders", "gameStores", "loaders",
		"lastProfile", "lastPlayed", "backgroundImage", "dismissed",
		"nxmHandled", "nxmPrevious", "nxmAsked",
		"overlayToken", "overlayEnabled", "overlayPort",
	} {
		if _, ok := m[k]; ok {
			t.Fatalf("exported %s: %s", k, b)
		}
	}
	raw := string(b)
	for _, secret := range []string{"Farmer", "/games/Stardew", "abc", "/home/u/wall", "overlay-secret-token"} {
		if strings.Contains(raw, secret) {
			t.Fatalf("export still holds %q: %s", secret, raw)
		}
	}
}

func TestLanguagePortableRoundTrip(t *testing.T) {
	p, present, err := ParseExport([]byte(`{"version":1,"language":"en"}`))
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
	raw := []byte(`{"version":1,"accent":"neon","mystery":true,"nexusName":"x","gameFolders":{"stardew":"/nope"},"backupsKept":12}`)
	p, present, err := ParseExport(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Accent != "sand" {
		t.Fatalf("invalid accent as on load: %q", p.Accent)
	}
	if p.BackupsKept != 12 {
		t.Fatalf("backupsKept = %d", p.BackupsKept)
	}
	cur := Defaults()
	cur.NexusName = "keep"
	cur.NexusUserID = 7
	cur.GameFolders["stardew"] = "/keep"
	cur.Loaders["stardew"] = "4.5.2"
	cur.LastProfile["stardew"] = "abc"
	ApplyExport(&cur, p, present)
	if cur.NexusName != "keep" || cur.NexusUserID != 7 {
		t.Fatalf("nexus display changed: %+v", cur)
	}
	if cur.GameFolders["stardew"] != "/keep" || cur.Loaders["stardew"] != "4.5.2" || cur.LastProfile["stardew"] != "abc" {
		t.Fatalf("machine fields changed: %+v", cur)
	}
	if cur.BackupsKept != 12 {
		t.Fatalf("backupsKept = %d", cur.BackupsKept)
	}
	if cur.Accent != "sand" {
		t.Fatalf("accent = %q", cur.Accent)
	}
}

func TestImportRejectsMissingOrUnknownVersion(t *testing.T) {
	if _, _, err := ParseExport([]byte(`{"accent":"moss"}`)); err == nil {
		t.Fatal("missing version accepted")
	}
	if _, _, err := ParseExport([]byte(`{"version":2,"accent":"moss"}`)); err == nil {
		t.Fatal("version 2 accepted")
	}
}

func TestImportPreviewGroupsBySectionAndNotesUnknown(t *testing.T) {
	raw := []byte(`{"version":1,"accent":"sky","backupsKept":9,"parallelDownloads":2,"mystery":1,"nexusName":"x"}`)
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
	raw := []byte(`{"version":1,"accent":"sky","backupsKept":9}`)
	cur := Defaults()
	if err := ApplyImport(&cur, raw, []string{SectionStorage}); err != nil {
		t.Fatal(err)
	}
	if cur.BackupsKept != 9 || cur.Accent != Defaults().Accent {
		t.Fatalf("backupsKept=%d accent=%q", cur.BackupsKept, cur.Accent)
	}
	if err := ApplyImport(&cur, raw, nil); err != nil || cur.Accent != Defaults().Accent {
		t.Fatalf("no sections changed accent: %v", err)
	}
}
