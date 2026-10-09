package nativehost

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
)

func frame(t *testing.T, v any) []byte {
	t.Helper()
	if r, ok := v.(request); ok && r.Protocol == nil {
		r.Protocol = new(Protocol)
		v = r
	}
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
	testfs.DataHome(t)
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
			// Answers the hello that controlwire.Live sends with an empty result.
			_, _ = bufio.NewReader(c).ReadBytes('\n')
			_, _ = c.Write([]byte("{}\n"))
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

func writeProfileJSON(t *testing.T, dir, id string, profile map[string]any) {
	t.Helper()
	pdir := filepath.Join(dir, "profiles", "stardew", id)
	if err := os.MkdirAll(pdir, 0o700); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pdir, "profile.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestServeHandsEachLinkOverAndReplies(t *testing.T) {
	in := append(frame(t, request{Link: "nxm://a"}), frame(t, request{Link: "nxm://bad"})...)
	var got []string
	var out bytes.Buffer
	err := ServeFrom(nil, bytes.NewReader(in), &out, func(link string) error {
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
	in := frame(t, request{Type: "installed", Source: "nexus", SourceGameKey: "stardewvalley"})
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

func TestActiveNexusStateOffWhenExtensionOff(t *testing.T) {
	listenControl(t, `{"global":{"extensionConnection":"off","lastProfile":{"stardew":"aaaaaaaaaaaaaaaa"}}}`)
	if st, _ := activeNexusState("stardewvalley"); st != stateOff {
		t.Fatalf("extension off state = %q", st)
	}
}

func TestActiveNexusStateNotRunningThenNoProfileThenReady(t *testing.T) {
	testfs.DataHome(t)
	if st, _ := activeNexusState("stardewvalley"); st != stateNotRunning {
		t.Fatalf("closed Mortar state = %q", st)
	}
	dir := listenControl(t, `{}`)
	if st, _ := activeNexusState("stardewvalley"); st != stateNoProfile {
		t.Fatalf("no profile state = %q", st)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"global":{"lastProfile":{"stardew":"p1"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	pdir := filepath.Join(dir, "profiles", "stardew", "p1")
	if err := os.MkdirAll(pdir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pdir, "profile.json"), []byte(`{"name":"Farm"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if st, name := activeNexusState("stardewvalley"); st != stateReady || name != "Farm" {
		t.Fatalf("ready state = %q %q", st, name)
	}
}

func TestServeOffAnswersNothingAndModCarriesProblemsAndRequirements(t *testing.T) {
	h := handlers{
		open:      func(string) error { return nil },
		installed: func(string) []int { t.Fatal("off read installed"); return nil },
		im: func(string, int) (modInProfile, []modInProfile) {
			return modInProfile{Profile: "Farm", PageVersion: "2.0"}, nil
		},
		problems:     func(string, int) []modProblem { return []modProblem{{Kind: "x", Text: "bad"}} },
		requirements: func(string, int) []requirementItem { return []requirementItem{{Name: "SMAPI", External: true}} },
	}
	for _, st := range []string{stateOff, stateReady, stateNoProfile} {
		h.state = func(string) (string, string) { return st, "Farm" }
		var out bytes.Buffer
		in := append(frame(t, request{Type: "installed", Source: "nexus", SourceGameKey: "stardewvalley"}), frame(t, request{Type: "mod", Source: "nexus", SourceGameKey: "stardewvalley", ModID: 1})...)
		if st == stateOff {
			h.installed = func(string) []int { return nil }
		}
		if err := serveHandlers(bytes.NewReader(in), &out, h); err != nil {
			t.Fatal(err)
		}
		installed, im := readReply(t, &out), readReply(t, &out)
		if installed.State != st || installed.Connected != (st == stateReady) {
			t.Fatalf("%s installed reply = %+v", st, installed)
		}
		wantMod := st == stateReady
		if (im.Open != nil) != (st != stateOff) || (len(im.Problems) == 1) != wantMod || (len(im.Requirements) == 1) != wantMod {
			t.Fatalf("%s mod reply = %+v", st, im)
		}
		if st == stateReady && im.Open.PageVersion != "2.0" {
			t.Fatalf("page version = %+v", im.Open)
		}
	}
}

func TestServeReportsConnectedWhenProfileHasNoMods(t *testing.T) {
	in := frame(t, request{Type: "installed", Source: "nexus", SourceGameKey: "stardewvalley"})
	var out bytes.Buffer
	err := serveHandlers(bytes.NewReader(in), &out, handlers{
		open:      func(string) error { return nil },
		installed: func(string) []int { return []int{} },
		state:     func(string) (string, string) { return stateReady, "Main" },
		broken:    func(string) []int { return []int{7} },
	})
	if err != nil {
		t.Fatal(err)
	}
	got := readReply(t, &out)
	if !got.Connected || got.ModIDs == nil || len(*got.ModIDs) != 0 || !slices.Equal(got.BrokenIDs, []int{7}) {
		t.Fatalf("installed reply = %+v", got)
	}
}

func TestServeAnswersModProfiles(t *testing.T) {
	in := frame(t, request{Type: "mod", Source: "nexus", SourceGameKey: "stardewvalley", ModID: 123})
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
	dir := listenControl(t, `{"global":{"lastProfile":{"stardew":"`+openID+`"}}}`)
	writeProfile := func(id, name string, fileID int, body []byte) {
		t.Helper()
		pdir := filepath.Join(dir, "profiles", "stardew", id)
		if err := os.MkdirAll(pdir, 0o700); err != nil {
			t.Fatal(err)
		}
		if body == nil {
			var err error
			body, err = json.Marshal(map[string]any{
				"name": name,
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
	writeProfile(openID, "Default", 111, nil)
	writeProfile("bbbbbbbbbbbbbbbb", "Co-op", 222, nil)
	writeProfile("cccccccccccccccc", "Shared", 111, nil)
	writeProfile("eeeeeeeeeeeeeeee", "Broken", 444, []byte("{"))
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
	if _, ok := byName["Broken"]; ok {
		t.Fatalf("damaged profile listed: %+v", others)
	}
}

func TestNexusModProfilesReportsRequiredByPinAndSkip(t *testing.T) {
	openID := "aaaaaaaaaaaaaaaa"
	dir := listenControl(t, `{"global":{"lastProfile":{"stardew":"`+openID+`"}}}`)
	writeProfileJSON(t, dir, openID, map[string]any{
		"name": "Default",
		"entries": []map[string]any{
			{
				"pinned":      true,
				"skipVersion": "3.0.0",
				"skipSources": []string{"github"},
				"source":      map[string]any{"kind": "nexus", "modId": 1915, "fileId": 111},
				"mods":        []map[string]any{{"id": "smapi:Core.Lib", "name": "Core", "version": "2.0.0"}},
			},
			{
				"source": map[string]any{"kind": "nexus", "modId": 99, "fileId": 2},
				"mods": []map[string]any{{
					"id": "smapi:Farm.Pack", "name": "Farm pack", "needs": []string{"smapi:Core.Lib"},
				}},
			},
			{
				"disabled": []string{"smapi:Off.Pack"},
				"source":   map[string]any{"kind": "local"},
				"mods":     []map[string]any{{"id": "smapi:Off.Pack", "name": "Off", "needs": []string{"smapi:Core.Lib"}}},
			},
		},
	})
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
	in := frame(t, request{Type: "mod", Source: "nexus", SourceGameKey: "stardewvalley", ModID: 1915})
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
	openID := "aaaaaaaaaaaaaaaa"
	dir := listenControl(t, `{"global":{"lastProfile":{"stardew":"`+openID+`"}}}`)
	write := func(id, name, version string, fileID int) {
		t.Helper()
		writeProfileJSON(t, dir, id, map[string]any{
			"name": name,
			"entries": []map[string]any{{
				"source": map[string]any{"kind": "nexus", "modId": 1915, "fileId": fileID},
				"mods":   []map[string]any{{"version": version}},
			}},
		})
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
	if err := os.WriteFile(filepath.Join(dir, "cache", filepath.FromSlash(nexussvc.DetailsName("stardewvalley", 1915))), cache, 0o600); err != nil {
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
	in := frame(t, request{Type: "updates", Source: "nexus", SourceGameKey: "stardewvalley"})
	var out bytes.Buffer
	err := serveHandlers(bytes.NewReader(in), &out, handlers{
		open: func(string) error {
			t.Fatal("updates request opened a link")
			return nil
		},
		updates: func(game string) (string, []modUpdate) {
			if game != "stardewvalley" {
				t.Fatalf("updates game = %q", game)
			}
			return "Default", []modUpdate{
				{ModID: 2, Name: "Zed", Installed: "1.0.0", Latest: "2.0.0"},
				{ModID: 1, Name: "Alpha", Installed: "3.0.0", Latest: ""},
			}
		},
	})
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
	openID := "aaaaaaaaaaaaaaaa"
	dir := listenControl(t, `{"global":{"lastProfile":{"stardew":"`+openID+`"}}}`)
	writeProfileJSON(t, dir, openID, map[string]any{
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
		path := filepath.Join(dir, "cache", filepath.FromSlash(nexussvc.DetailsName("stardewvalley", modID)))
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
	in := frame(t, request{Type: "updates", Source: "nexus", SourceGameKey: "stardewvalley"})
	var out bytes.Buffer
	if err := ServeFrom(nil, bytes.NewReader(in), &out, func(string) error {
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

func TestRecordContactWritesOncePerMinute(t *testing.T) {
	testfs.DataHome(t)
	contactWrote = Contact{}
	if !LastContact().LastSeen.IsZero() {
		t.Fatal("contact before any message")
	}
	first := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	recordContact("Firefox", "", first)
	recordContact("Chrome", "", first.Add(30*time.Second))
	got := LastContact()
	if got.Browser != "Firefox" || !got.LastSeen.Equal(first) {
		t.Fatalf("throttled contact = %+v", got)
	}
	recordContact("Chrome", "", first.Add(2*time.Minute))
	if got := LastContact(); got.Browser != "Chrome" {
		t.Fatalf("later contact = %+v", got)
	}
	recordContact("Chrome", ExtensionTooOld, first.Add(2*time.Minute+time.Second))
	if got := LastContact(); got.Mismatch != ExtensionTooOld {
		t.Fatalf("a protocol mismatch is recorded at once: %+v", got)
	}
}

func TestServeRefusesAnExtensionProtocolOutOfRange(t *testing.T) {
	in := append(frame(t, map[string]any{"link": "nxm://unversioned"}),
		frame(t, map[string]any{"protocol": MinProtocol - 1, "link": "nxm://zero"})...)
	in = append(in, frame(t, map[string]any{"protocol": MaxProtocol + 1, "type": "installed", "source": "nexus", "sourceGameKey": "stardewvalley"})...)
	in = append(in, frame(t, map[string]any{"protocol": Protocol, "link": "nxm://ok"})...)
	var out bytes.Buffer
	var opened, mismatches []string
	err := serveHandlers(bytes.NewReader(in), &out, handlers{
		contact: func(mismatch string) { mismatches = append(mismatches, mismatch) },
		open:    func(link string) error { opened = append(opened, link); return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{ExtensionTooOld, ExtensionTooOld, ExtensionTooNew, ""} {
		got := readReply(t, &out)
		if got.Protocol != Protocol || got.ProtocolError != want || (want == "") != got.OK {
			t.Fatalf("reply %+v, want protocolError %q", got, want)
		}
	}
	if !slices.Equal(opened, []string{"nxm://ok"}) {
		t.Fatalf("opened %v", opened)
	}
	if !slices.Equal(mismatches, []string{ExtensionTooOld, ExtensionTooOld, ExtensionTooNew, ""}) {
		t.Fatalf("contacts %v", mismatches)
	}
}

func TestRepliesNameTheGamesBySource(t *testing.T) {
	var out bytes.Buffer
	err := serveHandlers(bytes.NewReader(frame(t, request{Protocol: new(Protocol), Type: "installed", Source: "nexus", SourceGameKey: "stardewvalley"})), &out, handlers{
		installed: func(key string) []int { return []int{len(key)} },
		im:        func(string, int) (modInProfile, []modInProfile) { return modInProfile{}, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	got := readReply(t, &out)
	if got.Protocol != 2 || got.ModIDs == nil || (*got.ModIDs)[0] != len("stardewvalley") {
		t.Fatalf("reply = %+v", got)
	}
	i := slices.IndexFunc(got.Games, func(g hostGame) bool { return g.ID == "stardew" })
	if i < 0 || got.Games[i].Sources["nexus"] != "stardewvalley" || got.Games[i].Name == "" {
		t.Fatalf("games = %+v", got.Games)
	}
}

func serve(r io.Reader, w io.Writer, open func(link string) error, installed func(game string) []int, im func(game string, modID int) (modInProfile, []modInProfile), problem ...func(game string, modID int) []modProblem) error {
	h := handlers{open: open, installed: installed, im: im}
	if len(problem) > 0 {
		h.problems = problem[0]
	}
	return serveHandlers(r, w, h)
}

func TestManifestAllowsStoreOrigins(t *testing.T) {
	old := storeChromeIDs
	storeChromeIDs = []string{"abcdefghijklmnopabcdefghijklmnop"}
	t.Cleanup(func() { storeChromeIDs = old })
	b, err := Manifest("/usr/bin/mortar", false)
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		AllowedOrigins []string `json:"allowed_origins"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	want := []string{ChromeOrigin, "chrome-extension://abcdefghijklmnopabcdefghijklmnop/"}
	if !slices.Equal(m.AllowedOrigins, want) {
		t.Fatalf("allowed_origins = %q, want %q", m.AllowedOrigins, want)
	}
	if !Invoked([]string{want[1]}) {
		t.Fatal("store origin must start the host")
	}
}

func TestServeAnswersThunderstorePackages(t *testing.T) {
	var queued []string
	h := handlers{
		packages: func(key string) (string, string, []string) {
			if key != "riskofrain2" {
				t.Fatalf("community = %q", key)
			}
			return stateReady, "Run", []string{"Me-Mod"}
		},
		installPackage: func(key, pkg string) error {
			queued = append(queued, key+"/"+pkg)
			return nil
		},
	}
	in := append(frame(t, request{Type: "installedPackages", Source: "thunderstore", SourceGameKey: "riskofrain2"}),
		frame(t, request{Type: "installPackage", Source: "thunderstore", SourceGameKey: "riskofrain2", Package: "Me-Other"})...)
	var out bytes.Buffer
	if err := serveHandlers(bytes.NewReader(in), &out, h); err != nil {
		t.Fatal(err)
	}
	installed, install := readReply(t, &out), readReply(t, &out)
	if installed.Packages == nil || !slices.Equal(*installed.Packages, []string{"Me-Mod"}) || !installed.Connected {
		t.Errorf("installedPackages reply = %+v", installed)
	}
	if !install.OK || !slices.Equal(queued, []string{"riskofrain2/Me-Other"}) {
		t.Errorf("installPackage reply = %+v, queued %v", install, queued)
	}
}

func TestMortarRunningIgnoresAStalePortAnotherProgramReused(t *testing.T) {
	t.Parallel()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	port, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("not a TCP listener")
	}
	dir := t.TempDir()
	b, err := json.Marshal(controlwire.Discovery{Port: port.Port, Token: "stale"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, controlwire.FileName), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if mortarRunning(dir) {
		t.Fatal("a port that only accepts connections counted as a running Mortar")
	}
}
