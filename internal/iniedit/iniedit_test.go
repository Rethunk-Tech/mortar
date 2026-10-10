package iniedit

import "testing"

func TestSet(t *testing.T) {
	for _, c := range []struct{ name, in, want string }{
		{"rewrites case-insensitively", "[o]\r\nModsDisabled = 1\r\nx = 2\r\n", "[o]\r\nModsDisabled = 0\r\nx = 2\r\n"},
		{"ignores comments", "; modsdisabled = 1\n[o]\nmodsdisabled=1", "; modsdisabled = 1\n[o]\nmodsdisabled = 0"},
		{"appends when absent", "[o]\nx = 2", "[o]\nx = 2\nmodsdisabled = 0\n"},
		{"empty file", "", "modsdisabled = 0\n"},
	} {
		if got := Set(c.in, "modsdisabled", "0"); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestSetMatchesAKeyOnAByteOrderMarkedFirstLine(t *testing.T) {
	got := Set("\xef\xbb\xbfmodsdisabled = 1\n", "modsdisabled", "0")
	if got != "\xef\xbb\xbfmodsdisabled = 0\n" {
		t.Fatalf("got %q: the old value stays first and wins in most parsers", got)
	}
}
