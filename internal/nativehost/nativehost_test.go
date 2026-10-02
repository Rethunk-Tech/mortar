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
