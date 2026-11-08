package winname

import "testing"

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
