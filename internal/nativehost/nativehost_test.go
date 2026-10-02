package nativehost

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"
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
		return modInProfile{Profile: "Default", Version: &version}, []modInProfile{{Profile: "Co-op", Version: nil}}
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
		len(got.Others) != 1 || got.Others[0].Profile != "Co-op" || got.Others[0].Version != nil {
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
