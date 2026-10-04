package winname

import "testing"

func TestValidAndClean(t *testing.T) {
	for _, bad := range []string{"foo.", "foo ", "a?b", "a|b", "a\x01b", "CON", "nul.txt", "com1", "CONIN$", ""} {
		if Valid(bad) {
			t.Errorf("Valid(%q) = true", bad)
		}
	}
	for _, ok := range []string{"foo", "a.b", "Stardew Valley (Game)", ".hidden", "com10"} {
		if !Valid(ok) {
			t.Errorf("Valid(%q) = false", ok)
		}
	}
	for in, want := range map[string]string{"A:B": "A-B", "Main?. ": "Main-", "NUL": "NUL_", "": "", " . ": ""} {
		if got := Clean(in); got != want || (got != "" && !Valid(got)) {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}
