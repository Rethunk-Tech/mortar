package configsvc

import (
	"strings"
	"testing"
)

// FuzzCfgSetKeepsOtherLines holds that rewriting one setting with its own value never panics, never moves or
// changes any other line, and leaves a line already written as "Key = value" byte for byte as it was.
func FuzzCfgSetKeepsOtherLines(f *testing.F) {
	f.Add("## Settings file was created by plugin P v1.0\n## Plugin GUID: a.b\n\n[General]\n\n## Hi\n# Setting type: Boolean\n# Default value: true\nEnabled = true\n")
	f.Add("[A]\nKey=1\r\nOther =  two\n")
	f.Add("[x]\n# Setting type: Int32\n# Acceptable value range: From 1 to 5\nN = 3\n=\n[\n")
	f.Fuzz(func(t *testing.T, text string) {
		d := parseCfg(text)
		norm := strings.ReplaceAll(text, "\r\n", "\n")
		for _, listed := range d.entries {
			// A repeated key edits its first line, which is the entry set targets.
			e, _ := d.find(listed.section, listed.Key)
			out, err := d.set(e.section, e.Key, e.Value)
			if err != nil {
				continue
			}
			got, want := strings.Split(out, "\n"), strings.Split(norm, "\n")
			if len(got) != len(want) {
				t.Fatalf("line count %d, want %d", len(got), len(want))
			}
			for i := range got {
				if i != e.line && got[i] != want[i] {
					t.Fatalf("line %d changed: %q -> %q", i, want[i], got[i])
				}
			}
			// Validation canonicalises a value ("truE" becomes "true"), which is a change; anything it keeps must not move.
			if canon, _ := validate(e.Entry, e.Value); canon == e.Value && want[e.line] == e.Key+" = "+e.Value && out != norm {
				t.Fatalf("an unchanged value rewrote the file:\n%q\n%q", norm, out)
			}
		}
	})
}
