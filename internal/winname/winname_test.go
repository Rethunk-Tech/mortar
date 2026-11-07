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
	for in, want := range map[string]string{"A:B": "A-B", "Main?. ": "Main-", "NUL": "NUL_", "nul.txt": "nul_.txt", "com1.tar.gz": "com1_.tar.gz", "": "", " . ": ""} {
		if got := Clean(in); got != want || (got != "" && !Valid(got)) {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFoldJoinsNamesACaseInsensitiveFileSystemKeepsAsOne(t *testing.T) {
	t.Parallel()
	for _, pair := range [][2]string{{"ς.dll", "σ.dll"}, {"ſ", "S"}, {"µ", "μ"}, {"A/B", "a/b"}} {
		if Fold(pair[0]) != Fold(pair[1]) {
			t.Errorf("%q and %q fold apart", pair[0], pair[1])
		}
	}
	if Fold("a") == Fold("b") {
		t.Error("distinct names fold together")
	}
}
