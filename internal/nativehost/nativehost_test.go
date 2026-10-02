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
		var n uint32
		_ = binary.Read(&out, binary.NativeEndian, &n)
		var r reply
		if err := json.Unmarshal(out.Next(int(n)), &r); err != nil {
			t.Fatal(err)
		}
		replies = append(replies, r)
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
	var n uint32
	if err := binary.Read(&out, binary.NativeEndian, &n); err != nil {
		t.Fatal(err)
	}
	var got reply
	if err := json.Unmarshal(out.Next(int(n)), &got); err != nil {
		t.Fatal(err)
	}
	if got.ModIDs == nil || len(*got.ModIDs) != 2 || (*got.ModIDs)[0] != 123 || (*got.ModIDs)[1] != 456 {
		t.Fatalf("installed reply = %+v", got.ModIDs)
	}
}

func TestServeReportsConnectedWhenProfileHasNoMods(t *testing.T) {
	in := frame(t, request{Type: "installed", Game: "stardewvalley"})
	var out bytes.Buffer
	err := serveWithConnection(bytes.NewReader(in), &out, func(string) error { return nil },
		func(string) []int { return []int{} }, nil, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	var n uint32
	if err := binary.Read(&out, binary.NativeEndian, &n); err != nil {
		t.Fatal(err)
	}
	var got reply
	if err := json.Unmarshal(out.Next(int(n)), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Connected || got.ModIDs == nil || len(*got.ModIDs) != 0 {
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
	var n uint32
	if err := binary.Read(&out, binary.NativeEndian, &n); err != nil {
		t.Fatal(err)
	}
	var got reply
	if err := json.Unmarshal(out.Next(int(n)), &got); err != nil {
		t.Fatal(err)
	}
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
	port := addr.Port
	if err := os.WriteFile(filepath.Join(dir, "control.json"), []byte(`{"port":`+strconv.Itoa(port)+`}`), 0o600); err != nil {
		t.Fatal(err)
	}
	openID := "aaaaaaaaaaaaaaaa"
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"lastProfile":{"stardew":"`+openID+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
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
