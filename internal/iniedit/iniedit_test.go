package iniedit

import (
	"errors"
	"testing"
)

func TestSet(t *testing.T) {
	for _, c := range []struct{ name, in, section, want string }{
		{"rewrites case-insensitively", "[o]\r\nModsDisabled = 1\r\nx = 2\r\n", "", "[o]\r\nModsDisabled = 0\r\nx = 2\r\n"},
		{"ignores comments", "; modsdisabled = 1\n[o]\nmodsdisabled=1", "", "; modsdisabled = 1\n[o]\nmodsdisabled = 0"},
		{"appends when absent", "[o]\nx = 2", "", "[o]\nx = 2\nmodsdisabled = 0\n"},
		{"empty file", "", "", "modsdisabled = 0\n"},
		{"byte order mark on the first line", "\xef\xbb\xbfmodsdisabled = 1\n", "", "\xef\xbb\xbfmodsdisabled = 0\n"},
		{"byte order mark before a section", "\xef\xbb\xbf[Options]\r\nmodsdisabled = 1\r\n", "options", "\xef\xbb\xbf[Options]\r\nmodsdisabled = 0\r\n"},
		{"edits only inside its section", "[a]\nmodsdisabled = 1\n[b]\nmodsdisabled = 1\n", "B", "[a]\nmodsdisabled = 1\n[b]\nmodsdisabled = 0\n"},
		{"appends inside its section", "[a]\nx = 1\n\n[b]\ny = 2\n", "a", "[a]\nx = 1\nmodsdisabled = 0\n\n[b]\ny = 2\n"},
		{"appends inside the last section keeping CRLF", "[a]\r\nx = 1", "A", "[a]\r\nx = 1\r\nmodsdisabled = 0\r\n"},
		{"creates the section", "[a]\r\nx = 1\r\n", "b", "[a]\r\nx = 1\r\n[b]\r\nmodsdisabled = 0\r\n"},
		{"fills an empty section", "[a]\n[b]\ny = 2\n", "a", "[a]\nmodsdisabled = 0\n[b]\ny = 2\n"},
	} {
		got, err := Set(c.in, c.section, "modsdisabled", "0")
		if err != nil || got != c.want {
			t.Errorf("%s: got %q (%v) want %q", c.name, got, err, c.want)
		}
	}
}

func TestSetRefusesUTF16Untouched(t *testing.T) {
	for _, in := range []string{"\xff\xfem\x00o\x00", "\xfe\xff\x00m\x00o", "a\x00b\x00"} {
		got, err := Set(in, "", "k", "v")
		if !errors.Is(err, ErrUTF16) || got != in {
			t.Errorf("%q: got %q, %v", in, got, err)
		}
	}
}
