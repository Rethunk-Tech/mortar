package manifest

import (
	"slices"
	"testing"
)

// FuzzParse feeds SMAPI manifest.json files as mods ship them: BOMs, comments, trailing commas and odd key casing.
func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"\xef\xbb\xbf{\n  // comment\n  \"Name\": \"Mod\", \"Author\": \"A\", \"Version\": \"1.2.3\", \"UniqueID\": \"A.Mod\",\n  \"EntryDll\": \"Mod.dll\", \"UpdateKeys\": [\"Nexus:1234@sub\", \"GitHub:owner/repo\"],\n  \"Dependencies\": [{\"UniqueID\": \"B.Lib\", \"MinimumVersion\": \"2.0\", \"IsRequired\": \"false\"},],\n}",
		`{"uniqueid":"C.Pack","version":{"MajorVersion":1,"MinorVersion":0,"PatchVersion":0},"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"},"DeleteOldVersion":true}`,
		`{"UniqueID":"","UpdateKeys":"Nexus:1"}`,
		`{"UniqueID":"x","UpdateKeys":["GitHub:../..","GitHub:o/r?x","Nexus: 99999999999999999999"]}`,
		`/* unterminated`,
		`[]`,
		"{'UniqueID': 'A', Name: \"x\\\n\"}",
		"\xff\xfe{\x00\"\x00U\x00",
		"{\u201cUniqueID\u201d: \u201cA\u201d}",
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		m, err := Parse(b)
		if err != nil {
			return
		}
		if m.UniqueID == "" {
			t.Fatal("accepted a manifest without a UniqueID")
		}
		_, _ = m.ModID(), m.ContentPackForID()
		for _, d := range m.Dependencies {
			if d.UniqueID == "" {
				t.Fatalf("dependency without a UniqueID: %+v", d)
			}
			_ = d.Dep()
		}
		for _, k := range m.UpdateKeys {
			_, _ = NexusUpdateKey(k)
			if repo, ok := GitHubUpdateKey(k); ok && !ValidGitHubRepo(repo) {
				t.Fatalf("update key %q gives repo %q, which would move the GitHub URL's path", k, repo)
			}
		}
		// The cache hands out clones: a caller editing its copy must not change the next parse.
		again, err := Parse(b)
		if err != nil || !slices.Equal(again.UpdateKeys, m.UpdateKeys) || !slices.Equal(again.Dependencies, m.Dependencies) {
			t.Fatalf("second parse differs: %v", err)
		}
		if len(m.UpdateKeys) > 0 {
			m.UpdateKeys[0] = "changed"
			if third, _ := Parse(b); third.UpdateKeys[0] == "changed" {
				t.Fatal("parse cache shares its slices with callers")
			}
		}
	})
}
