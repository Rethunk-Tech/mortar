package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// vortexCase is one example of Vortex's cyberpunk2077_ext_redux test fixtures (test/unit/mods.example.*.ts): the files
// of an archive and the copies its installer makes of them, or, for a prompt case, the copies made when the player
// goes ahead. testdata/cyberpunk/vortex-cases.json holds them as data.
type vortexCase struct {
	Kind     string            `json:"kind"`
	Name     string            `json:"name"`
	Outcome  string            `json:"outcome"`
	Files    []string          `json:"files"`
	Copies   [][2]string       `json:"copies"`
	ModName  string            `json:"modName"`
	Contents map[string]string `json:"contents"`
}

// vortexOnlyGaps lists the examples this port deliberately lays out differently, each with the reason.
var vortexOnlyGaps = map[string]string{
	"core.cyberscript/Core CyberScript installs without prompting when all required paths present":                     "Vortex counts a CyberScript ASI the archive does not hold as one of its files, so it takes a stray file as the ASI; here every file below CyberScript's folders is placed and nothing else",
	"core.cyberscript/Core CyberScript cancels installation without prompting when all required paths are not present": "the same count: Vortex refuses on arithmetic that includes the missing ASI, which is not a rule the archive can be checked against",
}

func TestCyberpunkLayoutsFollowVortexsExamples(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("testdata/cyberpunk/vortex-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []vortexCase
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	g := Game{Loaders: []string{"folder"}, Targets: []Target{{ID: TargetMods, Layout: LayoutCyberpunk}}}
	for _, c := range cases {
		id := c.Kind + "/" + c.Name
		if why, ok := vortexOnlyGaps[id]; ok {
			t.Logf("skipped %s: %s", id, why)
			continue
		}
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, f := range c.Files {
				// A few examples list a folder as a file, beside the files in it.
				if slices.ContainsFunc(c.Files, func(o string) bool { return strings.HasPrefix(o, f+"/") }) {
					continue
				}
				body := "x"
				if s, ok := c.Contents[f]; ok {
					body = s
				}
				testfs.WriteAt(t, filepath.Join(dir, filepath.FromSlash(f)), []byte(body))
			}
			a := Open(dir, "key")
			a.Name = cmpName(c.ModName)
			l, err := (cyberpunk{}).Layout(a, g, nil)
			switch c.Outcome {
			case "fail":
				if err == nil && !l.Unrecognised {
					t.Fatalf("laid out %v, want a refusal", render(l))
				}
				return
			case "prompt":
				// Vortex asks; the layout here is what going ahead gives, with a note where it asked.
				if err != nil {
					t.Fatalf("prompt case refused: %v", err)
				}
			default:
				if err != nil {
					t.Fatalf("refused: %v", err)
				}
			}
			var got [][2]string
			for _, f := range l.Files {
				got = append(got, [2]string{f.Src, f.Rel})
			}
			slices.SortFunc(got, func(a, b [2]string) int { return strings.Compare(a[0]+"\x00"+a[1], b[0]+"\x00"+b[1]) })
			// Vortex lists a copy of CyberScript's ASI whether or not the archive holds one.
			want := slices.DeleteFunc(slices.Clone(c.Copies), func(p [2]string) bool { return !slices.Contains(c.Files, p[0]) })
			slices.SortFunc(want, func(a, b [2]string) int { return strings.Compare(a[0]+"\x00"+a[1], b[0]+"\x00"+b[1]) })
			if !slices.Equal(got, want) {
				t.Fatalf("copies\n got %v\nwant %v", fmtPairs(got), fmtPairs(want))
			}
		})
	}
}

func cmpName(n string) string {
	if n != "" {
		return n
	}
	return "Fake Mod For Examples"
}

func fmtPairs(ps [][2]string) string {
	var out []string
	for _, p := range ps {
		out = append(out, fmt.Sprintf("%s -> %s", p[0], p[1]))
	}
	return strings.Join(out, "\n     ")
}

var _ = errors.Is
