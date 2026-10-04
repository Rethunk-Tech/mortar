package nativehost

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/datadir"
)

func frame(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	n, err := frameLen(len(b))
	if err != nil {
		t.Fatal(err)
	}
	_ = binary.Write(&buf, binary.NativeEndian, n)
	buf.Write(b)
	return buf.Bytes()
}

// readReply reads one framed reply off out.
func readReply(t *testing.T, out *bytes.Buffer) reply {
	t.Helper()
	var n uint32
	if err := binary.Read(out, binary.NativeEndian, &n); err != nil {
		t.Fatal(err)
	}
	var r reply
	if err := json.Unmarshal(out.Next(int(n)), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func listenControl(t *testing.T, settingsJSON string) string {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listen addr = %T", ln.Addr())
	}
	if err := os.WriteFile(filepath.Join(dir, "control.json"), []byte(`{"port":`+strconv.Itoa(addr.Port)+`}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if settingsJSON != "" {
		if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(settingsJSON), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestServeHandsEachLinkOverAndReplies(t *testing.T) {
	in := append(frame(t, request{Link: "nxm://a"}), frame(t, request{Link: "nxm://bad"})...)
	var got []string
	var out bytes.Buffer
	err := Serve(bytes.NewReader(in), &out, func(link string) error {
		got = append(got, link)
		if link == "nxm://bad" {
			return errors.New("refused")
		}
		return nil
	})
	if err != nil || len(got) != 2 || got[0] != "nxm://a" {
		t.Fatalf("Serve = %v, links %v", err, got)
	}
	var replies []reply
	for out.Len() > 0 {
		replies = append(replies, readReply(t, &out))
	}
	if len(replies) != 2 || !replies[0].OK || replies[1].OK || replies[1].Error != "refused" {
		t.Fatalf("replies = %+v", replies)
	}
}

func TestServeAnswersInstalledMods(t *testing.T) {
	in := frame(t, request{Type: "installed", Game: "stardewvalley"})
	var out bytes.Buffer
	err := serve(bytes.NewReader(in), &out, func(string) error {
		t.Fatal("installed request opened a link")
		return nil
	}, func(game string) []int {
		if game != "stardewvalley" {
			t.Fatalf("installed game = %q", game)
		}
		return []int{123, 456}
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := readReply(t, &out)
	if got.ModIDs == nil || len(*got.ModIDs) != 2 || (*got.ModIDs)[0] != 123 || (*got.ModIDs)[1] != 456 {
		t.Fatalf("installed reply = %+v", got.ModIDs)
	}
}

func TestActiveNexusConnectedOffWhenExtensionOff(t *testing.T) {
	listenControl(t, `{"extensionConnection":"off","lastProfile":{"stardew":"aaaaaaaaaaaaaaaa"}}`)
	if activeNexusConnected("stardewvalley") {
		t.Fatal("extension off should reply not-connected")
	}
}

func TestServeReportsConnectedWhenProfileHasNoMods(t *testing.T) {
	in := frame(t, request{Type: "installed", Game: "stardewvalley"})
	var out bytes.Buffer
	err := serveWithConnection(bytes.NewReader(in), &out, func(string) error { return nil },
		func(string) []int { return []int{} }, nil, func(string) bool { return true }, nil, func(string) []int { return []int{7} })
	if err != nil {
		t.Fatal(err)
	}
	got := readReply(t, &out)
	if !got.Connected || got.ModIDs == nil || len(*got.ModIDs) != 0 || !slices.Equal(got.BrokenIDs, []int{7}) {
		t.Fatalf("installed reply = %+v", got)
	}
}

func TestServeAnswersModProfiles(t *testing.T) {
	in := frame(t, request{Type: "mod", Game: "stardewvalley", ModID: 123})
	var out bytes.Buffer
	err := serve(bytes.NewReader(in), &out, func(string) error {
		t.Fatal("mod request opened a link")
		return nil
	}, nil, func(game string, modID int) (modInProfile, []modInProfile) {
		if game != "stardewvalley" || modID != 123 {
			t.Fatalf("mod request = %q/%d", game, modID)
		}
		version := "1.2.3"
		return modInProfile{Profile: "Default", Version: &version, FileID: 456}, []modInProfile{{Profile: "Co-op", Version: nil, FileID: 789}}
	})
	if err != nil {
		t.Fatal(err)
	}
	got := readReply(t, &out)
	if got.Open == nil || got.Open.Profile != "Default" || got.Open.Version == nil || *got.Open.Version != "1.2.3" ||
		got.Open.FileID != 456 ||
		len(got.Others) != 1 || got.Others[0].Profile != "Co-op" || got.Others[0].Version != nil || got.Others[0].FileID != 789 {
		t.Fatalf("mod reply = %+v", got)
	}
}

func TestInvoked(t *testing.T) {
	for _, c := range []struct {
		args []string
		want bool
	}{
		{[]string{ChromeOrigin}, true},
		{[]string{"/home/u/.mozilla/native-messaging-hosts/tech.rethunk.mortar.json", FirefoxID}, true},
		{[]string{"nxm://stardewvalley/mods/1/files/2"}, false},
		{nil, false},
	} {
		if got := Invoked(c.args); got != c.want {
			t.Errorf("Invoked(%q) = %v", c.args, got)
		}
	}
}

func TestNexusModProfilesReturnsPerProfileFileIDs(t *testing.T) {
	openID := "aaaaaaaaaaaaaaaa"
	dir := listenControl(t, `{"lastProfile":{"stardew":"`+openID+`"}}`)
	writeProfile := func(id, name string, hidden bool, fileID int, body []byte) {
		t.Helper()
		pdir := filepath.Join(dir, "profiles", "stardew", id)
		if err := os.MkdirAll(pdir, 0o700); err != nil {
			t.Fatal(err)
		}
		if body == nil {
			var err error
			body, err = json.Marshal(map[string]any{
				"name":   name,
				"hidden": hidden,
				"entries": []map[string]any{{
					"source": map[string]any{"kind": "nexus", "modId": 1915, "fileId": fileID},
					"mods":   []map[string]any{{"version": "2.0.0"}},
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(pdir, "profile.json"), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeProfile(openID, "Default", false, 111, nil)
	writeProfile("bbbbbbbbbbbbbbbb", "Co-op", false, 222, nil)
	writeProfile("cccccccccccccccc", "Shared", false, 111, nil)
	writeProfile("dddddddddddddddd", "Hidden", true, 333, nil)
	writeProfile("eeeeeeeeeeeeeeee", "Broken", false, 444, []byte("{"))
	openProfile, others := nexusModProfiles("stardewvalley", 1915)
	if openProfile.Profile != "Default" || openProfile.FileID != 111 {
		t.Fatalf("open = %+v", openProfile)
	}
	byName := map[string]int{}
	for _, p := range others {
		byName[p.Profile] = p.FileID
	}
	if byName["Co-op"] != 222 || byName["Shared"] != 111 {
		t.Fatalf("others = %+v", others)
	}
	if _, ok := byName["Hidden"]; ok {
		t.Fatalf("hidden profile listed: %+v", others)
	}
	if _, ok := byName["Broken"]; ok {
		t.Fatalf("damaged profile listed: %+v", others)
	}
}

func TestNexusModProfilesReportsRequiredByPinAndSkip(t *testing.T) {
	dir := listenControl(t, "")
	openID := "aaaaaaaaaaaaaaaa"
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"lastProfile":{"stardew":"`+openID+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	pdir := filepath.Join(dir, "profiles", "stardew", openID)
	if err := os.MkdirAll(pdir, 0o700); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"name": "Default",
		"entries": []map[string]any{
			{
				"pinned":      true,
				"skipVersion": "3.0.0",
				"skipSources": []string{"github"},
				"source":      map[string]any{"kind": "nexus", "modId": 1915, "fileId": 111},
				"mods":        []map[string]any{{"uniqueId": "Core.Lib", "name": "Core", "version": "2.0.0"}},
			},
			{
				"source": map[string]any{"kind": "nexus", "modId": 99, "fileId": 2},
				"mods": []map[string]any{{
					"uniqueId": "Farm.Pack", "name": "Farm pack", "needs": []string{"Core.Lib"},
				}},
			},
			{
				"disabled": []string{"Off.Pack"},
				"source":   map[string]any{"kind": "local"},
				"mods":     []map[string]any{{"uniqueId": "Off.Pack", "name": "Off", "needs": []string{"Core.Lib"}}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pdir, "profile.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	openProfile, _ := nexusModProfiles("stardewvalley", 1915)
	if !openProfile.Pinned || openProfile.SkipVersion != "3.0.0" || len(openProfile.SkipSources) != 1 || openProfile.SkipSources[0] != "github" {
		t.Fatalf("pin/skip = %+v", openProfile)
	}
	if len(openProfile.RequiredBy) != 1 || openProfile.RequiredBy[0] != "Farm.Pack" {
		t.Fatalf("requiredBy = %v", openProfile.RequiredBy)
	}
	if len(openProfile.RequiredByNames) != 1 || openProfile.RequiredByNames[0] != "Farm pack" {
		t.Fatalf("requiredByNames = %v", openProfile.RequiredByNames)
	}
}

func TestServeAnswersModUpdateAvailable(t *testing.T) {
	in := frame(t, request{Type: "mod", Game: "stardewvalley", ModID: 1915})
	var out bytes.Buffer
	err := serve(bytes.NewReader(in), &out, func(string) error {
		t.Fatal("mod request opened a link")
		return nil
	}, nil, func(string, int) (modInProfile, []modInProfile) {
		v := "1.0.0"
		fresh := "2.0.0"
		return modInProfile{Profile: "Default", Version: &v, UpdateAvailable: true},
			[]modInProfile{{Profile: "Co-op", Version: &fresh}}
	})
	if err != nil {
		t.Fatal(err)
	}
	got := readReply(t, &out)
	if got.Open == nil || !got.Open.UpdateAvailable || len(got.Others) != 1 || got.Others[0].UpdateAvailable {
		t.Fatalf("mod reply = %+v", got)
	}
}

func TestNexusModProfilesUpdateAvailableFromCache(t *testing.T) {
	dir := listenControl(t, "")
	openID := "aaaaaaaaaaaaaaaa"
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"lastProfile":{"stardew":"`+openID+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	write := func(id, name, version string, fileID int) {
		t.Helper()
		pdir := filepath.Join(dir, "profiles", "stardew", id)
		if err := os.MkdirAll(pdir, 0o700); err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal(map[string]any{
			"name": name,
			"entries": []map[string]any{{
				"source": map[string]any{"kind": "nexus", "modId": 1915, "fileId": fileID},
				"mods":   []map[string]any{{"version": version}},
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pdir, "profile.json"), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(openID, "Default", "1.0.0", 10)
	write("bbbbbbbbbbbbbbbb", "Co-op", "3.0.0", 30)
	cacheDir := filepath.Join(dir, "cache", "nexus")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cache, err := json.Marshal(map[string]any{
		"fetched": "2026-01-01T00:00:00Z",
		"value": map[string]any{
			"page": map[string]any{"version": "2.0.0"},
			"files": []map[string]any{
				{"fileId": 10, "version": "1.0.0", "replacedBy": 20},
				{"fileId": 20, "version": "2.0.0"},
				{"fileId": 30, "version": "3.0.0"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "details-v3-stardewvalley-1915.json"), cache, 0o600); err != nil {
		t.Fatal(err)
	}
	openProfile, others := nexusModProfiles("stardewvalley", 1915)
	if !openProfile.UpdateAvailable {
		t.Fatalf("open update = %+v", openProfile)
	}
	if len(others) != 1 || others[0].Profile != "Co-op" || others[0].UpdateAvailable {
		t.Fatalf("others = %+v", others)
	}
}

func TestServeAnswersUpdates(t *testing.T) {
	in := frame(t, request{Type: "updates", Game: "stardewvalley"})
	var out bytes.Buffer
	err := serveWithConnection(bytes.NewReader(in), &out, func(string) error {
		t.Fatal("updates request opened a link")
		return nil
	}, nil, nil, nil, func(game string) (string, []modUpdate) {
		if game != "stardewvalley" {
			t.Fatalf("updates game = %q", game)
		}
		return "Default", []modUpdate{
			{ModID: 2, Name: "Zed", Installed: "1.0.0", Latest: "2.0.0"},
			{ModID: 1, Name: "Alpha", Installed: "3.0.0", Latest: ""},
		}
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := readReply(t, &out)
	if got.Profile != "Default" || got.Updates == nil || len(*got.Updates) != 2 {
		t.Fatalf("updates reply = %+v", got)
	}
	gotUpdates := *got.Updates
	if gotUpdates[0].Name != "Alpha" || gotUpdates[0].ModID != 1 || gotUpdates[0].Latest != "" {
		t.Fatalf("sorted empty latest = %+v", gotUpdates[0])
	}
	if gotUpdates[1].Name != "Zed" || gotUpdates[1].Installed != "1.0.0" || gotUpdates[1].Latest != "2.0.0" {
		t.Fatalf("second update = %+v", gotUpdates[1])
	}
}

func TestActiveNexusUpdatesFromCache(t *testing.T) {
	dir := listenControl(t, "")
	openID := "aaaaaaaaaaaaaaaa"
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"lastProfile":{"stardew":"`+openID+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	pdir := filepath.Join(dir, "profiles", "stardew", openID)
	if err := os.MkdirAll(pdir, 0o700); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"name": "Default",
		"entries": []map[string]any{
			{
				"source": map[string]any{"kind": "nexus", "modId": 10, "fileId": 1},
				"mods":   []map[string]any{{"name": "Zed", "version": "1.0.0"}},
			},
			{
				"source": map[string]any{"kind": "nexus", "modId": 20, "fileId": 2},
				"mods":   []map[string]any{{"name": "Current", "version": "5.0.0"}},
			},
			{
				"source": map[string]any{"kind": "nexus", "modId": 30, "fileId": 3},
				"mods":   []map[string]any{{"name": "Alpha", "version": "0.1.0"}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pdir, "profile.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(dir, "cache", "nexus")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeCache := func(modID int, page string, files []map[string]any) {
		t.Helper()
		cache, err := json.Marshal(map[string]any{
			"fetched": "2026-01-02T03:04:05Z",
			"value": map[string]any{
				"page":  map[string]any{"version": page},
				"files": files,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(cacheDir, "details-v3-stardewvalley-"+strconv.Itoa(modID)+".json")
		if err := os.WriteFile(path, cache, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeCache(10, "2.0.0", []map[string]any{{"fileId": 1, "version": "1.0.0"}})
	writeCache(20, "5.0.0", []map[string]any{{"fileId": 2, "version": "5.0.0"}})
	writeCache(30, "", []map[string]any{
		{"fileId": 3, "version": "0.1.0", "replacedBy": 4},
		{"fileId": 4, "version": "0.2.0"},
	})
	name, rows := activeNexusUpdates("stardewvalley")
	if name != "Default" || len(rows) != 2 {
		t.Fatalf("updates = %q %+v", name, rows)
	}
	if rows[0].Name != "Alpha" || rows[0].ModID != 30 || rows[0].Installed != "0.1.0" || rows[0].Latest != "" {
		t.Fatalf("unknown latest = %+v", rows[0])
	}
	if rows[1].Name != "Zed" || rows[1].ModID != 10 || rows[1].Latest != "2.0.0" {
		t.Fatalf("named update = %+v", rows[1])
	}
	in := frame(t, request{Type: "updates", Game: "stardewvalley"})
	var out bytes.Buffer
	if err := Serve(bytes.NewReader(in), &out, func(string) error {
		t.Fatal("updates request opened a link")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	got := readReply(t, &out)
	if got.Profile != "Default" || got.Updates == nil || len(*got.Updates) != 2 || (*got.Updates)[0].Latest != "" || (*got.Updates)[1].Name != "Zed" {
		t.Fatalf("Serve updates = %+v", got)
	}
}

func TestNexusArchiveNamesGiveTheirModID(t *testing.T) {
	cases := map[string]int{
		"Random Lost Library Book Covers-15393-1-2-0-1726137756.zip": 15393,
		"Aspen-6754-0-0-53-1710866042.zip":                           6754,
		"Fish Helper-33167-1-2-0-1758836697.7z":                      33167,
		"EvenBetterArtisanGoodIcons.1.6.6.zip":                       0,
		"my-pack-2-1.zip":                                            0,
	}
	for name, want := range cases {
		if got := nexusArchiveModID(name); got != want {
			t.Errorf("%s: %d, want %d", name, got, want)
		}
	}
}
