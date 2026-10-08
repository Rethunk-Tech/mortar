package jsonc

import (
	"encoding/json"
	"testing"
)

// FuzzClean checks that Clean never panics and that whatever it turns into valid JSON stays valid when cleaned again.
func FuzzClean(f *testing.F) {
	for _, s := range []string{
		"\xef\xbb\xbf{'a': 'b', c: [1,], /* x */ // y\n}",
		"\xff\xfe{\x00}\x00",
		"{a:\"\n\"}", "{'a':'\\", "/*", "{a:-Infinity}", "[NaN,undefined,]", "{\"a\\", "\xfe\xff\x00",
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		once := Clean(b)
		if json.Valid(once) && !json.Valid(Clean(once)) {
			t.Fatalf("cleaning valid output %q broke it", once)
		}
	})
}
