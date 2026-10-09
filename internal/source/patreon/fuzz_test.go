package patreon

import (
	"strings"
	"testing"
)

// FuzzParsePostURL holds that no text yields an id that is not digits, and that an accepted id rebuilds an address on
// patreon.com.
func FuzzParsePostURL(f *testing.F) {
	f.Add("https://www.patreon.com/posts/cool-mod-12345678")
	f.Add("https://patreon.com/posts/1")
	f.Add("https://www.patreon.com/posts/%2e%2e-1")
	f.Add("https://user:pw@www.patreon.com/posts/x-1")
	f.Fuzz(func(t *testing.T, text string) {
		id, ok := ParsePostURL(text)
		if !ok {
			return
		}
		if !ValidID(id) || !strings.HasPrefix(PostURL(id), "https://www.patreon.com/posts/") {
			t.Fatalf("%q gave %q", text, id)
		}
	})
}
