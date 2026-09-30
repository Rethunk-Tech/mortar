package settings

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExportOmitsSecretsAndMachineFields(t *testing.T) {
	s := Defaults()
	s.Accent = "moss"
	s.NexusName = "NOMAD"
	s.NexusUserID = 99
	s.NexusPremium = true
	s.GameFolders = map[string]string{"stardew": "/games/Stardew Valley"}
	s.Loaders = map[string]string{"stardew": "4.5.2"}
	s.LastProfile = map[string]string{"stardew": "abc"}
	s.LastPlayed = map[string]Played{"stardew": {Profile: "abc", At: "2026-01-01T00:00:00Z"}}
	s.BackgroundImage = "/home/u/wall.png"
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
	for _, k := range []string{
		"nexusName", "nexusUserId", "nexusPremium", "gameFolders", "loaders",
		"lastProfile", "lastPlayed", "backgroundImage", "dismissed",
		"nxmHandled", "nxmPrevious", "nxmAsked",
	} {
		if _, ok := m[k]; ok {
			t.Fatalf("exported %s: %s", k, b)
		}
	}
	raw := string(b)
	for _, secret := range []string{"NOMAD", "/games/Stardew", "abc", "/home/u/wall"} {
		if strings.Contains(raw, secret) {
			t.Fatalf("export still holds %q: %s", secret, raw)
		}
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

func TestImportPreviewListsChanges(t *testing.T) {
	cur := Defaults()
	raw := []byte(`{"version":1,"accent":"sky","backupsKept":9}`)
	p, present, err := ParseExport(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := previewChanges(cur, p, present)
	if len(got) != 2 {
		t.Fatalf("changes = %+v", got)
	}
}
