package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

var placeholder = regexp.MustCompile(`\{[a-zA-Z]+\}`)

func placeholders(msg string) []string {
	found := placeholder.FindAllString(msg, -1)
	slices.Sort(found)
	return slices.Compact(found)
}

func TestParsePO(t *testing.T) {
	t.Parallel()
	const po = `msgid ""
msgstr ""
"Language: de\n"

#: src/a.ts
msgid "Quit"
msgstr "Beenden"

#, fuzzy
msgid "Fuzzy"
msgstr "Unscharf"

msgid "Untranslated"
msgstr ""

msgid ""
"Line one:\n"
"\n"
"{path} \"quoted\"\\"
msgstr ""
"Zeile eins:\n"
"\n"
"{path} \"zitiert\"\\"

#~ msgid "Gone"
#~ msgstr "Weg"
`
	got, err := parsePO(strings.NewReader(po))
	if err != nil {
		t.Fatal(err)
	}
	want := catalog{
		"Quit":                             "Beenden",
		"Line one:\n\n{path} \"quoted\"\\": "Zeile eins:\n\n{path} \"zitiert\"\\",
	}
	if len(got) != len(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%q = %q, want %q", k, got[k], v)
		}
	}
	if _, err := parsePO(strings.NewReader("msgctxt \"x\"\n")); err == nil {
		t.Error("msgctxt accepted")
	}
}

func TestLocaleAndFormat(t *testing.T) {
	t.Parallel()
	cats := map[string]catalog{"en": {}, "de": {"Quit": "Beenden"}, "pt-BR": {}}
	cases := []struct{ setting, system, want string }{
		{"", "de-DE", "de"},
		{"", "pt-BR", "pt-BR"},
		{"", "pt-PT", "en"},
		{"", "fr-FR", "en"},
		{"", "", "en"},
		{"de", "fr-FR", "de"},
		{"fr", "de-AT", "de"},
	}
	for _, c := range cases {
		if got := resolve(cats, c.setting, c.system); got != c.want {
			t.Errorf("resolve(%q, %q) = %q, want %q", c.setting, c.system, got, c.want)
		}
	}
	for in, want := range map[string]string{
		"de_DE.UTF-8":    "de-DE",
		"pt_BR@euro":     "pt-BR",
		"en_US.UTF-8@x":  "en-US",
		"fr":             "fr",
		"C":              "",
		"POSIX":          "",
		"zh_Hant_TW.utf": "zh-Hant-TW",
	} {
		if got := posixTag(in); got != want {
			t.Errorf("posixTag(%q) = %q, want %q", in, got, want)
		}
	}
	if got := format("", "Stop {game}", []string{"game", "Stardew"}); got != "Stop Stardew" {
		t.Errorf("fallback = %q", got)
	}
	if got := format("{game} anhalten", "Stop {game}", []string{"game", "{path}", "path", "x"}); got != "{path} anhalten" {
		t.Errorf("translated = %q", got)
	}
}

// callSite is one i18n.T call: its msgid and the placeholder keys passed after it.
type callSite struct {
	pos   string
	msgid string
	keys  []string
}

// literal folds a string literal or a + chain of them.
func literal(e ast.Expr) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(e.Value)
		return s, err == nil
	case *ast.BinaryExpr:
		l, lok := literal(e.X)
		r, rok := literal(e.Y)
		return l + r, e.Op == token.ADD && lok && rok
	}
	return "", false
}

func callSites(t *testing.T) []callSite {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	var sites []callSite
	fset := token.NewFileSet()
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(name, ".") || slices.Contains([]string{"frontend", "web", "site", "tmp", "bin", "node_modules", "testdata"}, name)) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		src, err := fsx.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(src), "i18n.T(") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "T" {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "i18n" {
				return true
			}
			site := callSite{pos: fset.Position(call.Pos()).String()}
			if len(call.Args) == 0 {
				t.Errorf("%s: i18n.T without a msgid", site.pos)
				return true
			}
			msgid, ok := literal(call.Args[0])
			if !ok {
				t.Errorf("%s: the msgid is not a string literal", site.pos)
				return true
			}
			site.msgid = msgid
			for i := 1; i < len(call.Args); i += 2 {
				key, ok := literal(call.Args[i])
				if !ok {
					t.Errorf("%s: placeholder name %d is not a string literal", site.pos, i)
					continue
				}
				site.keys = append(site.keys, "{"+key+"}")
			}
			if len(call.Args)%2 == 0 {
				t.Errorf("%s: placeholder without a value", site.pos)
			}
			sites = append(sites, site)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return sites
}

func TestCatalogMatchesCallSites(t *testing.T) {
	t.Parallel()
	cats := catalogs()
	en, ok := cats["en"]
	if !ok {
		t.Fatal("no English catalog")
	}
	used := map[string]bool{}
	sites := callSites(t)
	if len(sites) == 0 {
		t.Fatal("found no i18n.T calls")
	}
	for _, s := range sites {
		used[s.msgid] = true
		if en[s.msgid] == "" {
			t.Errorf("%s: %q is missing from locales/en/native.po", s.pos, s.msgid)
		}
		keys := slices.Clone(s.keys)
		slices.Sort(keys)
		if want := placeholders(s.msgid); !slices.Equal(slices.Compact(keys), want) {
			t.Errorf("%s: passes %v for %q, which has %v", s.pos, keys, s.msgid, want)
		}
	}
	for msgid, msgstr := range en {
		if !used[msgid] {
			t.Errorf("locales/en/native.po has %q, which no i18n.T call uses", msgid)
		}
		if msgstr != msgid {
			t.Errorf("locales/en/native.po: %q has a different msgstr %q", msgid, msgstr)
		}
	}
	for locale, c := range cats {
		for msgid, msgstr := range c {
			if _, ok := en[msgid]; !ok {
				t.Errorf("%s: %q is not in the English catalog", locale, msgid)
			}
			if !slices.Equal(placeholders(msgid), placeholders(msgstr)) {
				t.Errorf("%s: %q changes the placeholders of %q", locale, msgstr, msgid)
			}
		}
	}
}
