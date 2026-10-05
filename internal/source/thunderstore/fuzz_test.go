package thunderstore

import (
	"encoding/json"
	"strings"
	"testing"
)

func FuzzParseLink(f *testing.F) {
	f.Add("ror2mm://v1/install/thunderstore.io/Ns/Name/1.2.3/")
	f.Add("ror2mm://v1/install/thunderstore.io/../x/1.0.0/")
	f.Add("ror2mm://v1/install/thunderstore.io/a/b/c/")
	f.Add("")
	f.Fuzz(func(t *testing.T, raw string) {
		ref, err := ParseLink(raw)
		if err != nil {
			return
		}
		for _, s := range []string{ref.Namespace, ref.Name} {
			if !part.MatchString(s) {
				t.Fatalf("bad part %q from %q", s, raw)
			}
		}
		if !versionRe.MatchString(ref.Version) || strings.Contains(ref.Namespace+ref.Name, "/") {
			t.Fatalf("bad ref %+v from %q", ref, raw)
		}
	})
}

func FuzzIndexChunkDecode(f *testing.F) {
	f.Add([]byte(`[]`))
	f.Add([]byte(`[{"name":"a","owner":"b","versions":[{"version_number":"1.0.0"}]}]`))
	f.Add([]byte(`{"x":`))
	f.Fuzz(func(t *testing.T, b []byte) {
		var pk []pkg
		_ = json.Unmarshal(b, &pk)
	})
}
