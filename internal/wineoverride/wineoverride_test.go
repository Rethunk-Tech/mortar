package wineoverride

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestEnsure(t *testing.T) {
	fixture, err := safefs.ReadFile("testdata/user.reg")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"section present":   string(fixture),
		"section absent":    strings.Replace(string(fixture), "[Software\\\\Wine\\\\DllOverrides]", "[Software\\\\Wine\\\\Other]", 1),
		"stale value":       strings.Replace(string(fixture), `"dxgi"="native,builtin"`, `"winhttp"="builtin"`, 1),
		"crlf line endings": strings.ReplaceAll(string(fixture), "\n", "\r\n"),
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			reg := filepath.Join(t.TempDir(), "user.reg")
			if err := safefs.WriteFile(reg, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := Ensure(reg, "winhttp"); err != nil {
				t.Fatal(err)
			}
			once, _ := safefs.ReadFile(reg)
			if err := Ensure(reg, "winhttp"); err != nil {
				t.Fatal(err)
			}
			twice, _ := safefs.ReadFile(reg)
			if string(once) != string(twice) || strings.Count(string(once), `"winhttp"="native,builtin"`) != 1 {
				t.Fatalf("not idempotent or duplicated:\n%s", once)
			}
			if backup, _ := safefs.ReadFile(reg + ".mortar-backup"); string(backup) != content {
				t.Fatal("backup is not the original")
			}
			// Every other line survives, in order, with the file's own line ending.
			eol := "\n"
			if strings.Contains(content, "\r\n") {
				eol = "\r\n"
			}
			if strings.Contains(strings.ReplaceAll(string(once), eol, ""), "\n") {
				t.Fatal("line endings mixed")
			}
			rest := strings.Split(strings.Replace(string(once), `"winhttp"="native,builtin"`+eol, "", 1), eol)
			for l := range strings.SplitSeq(content, eol) {
				if l != "" && !strings.Contains(strings.Join(rest, eol), l) && !strings.HasPrefix(l, `"winhttp"=`) && !strings.HasPrefix(l, `[Software\\Wine\\Other]`) {
					t.Fatalf("line lost: %q", l)
				}
			}
		})
	}
}
