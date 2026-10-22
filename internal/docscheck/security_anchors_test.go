package docscheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// anchorRE matches a `file.go:LINE` anchor, and an immediately following `Symbol` when the row names one.
var anchorRE = regexp.MustCompile("`([\\w./-]+\\.go):(\\d+)`(?: `([\\w.]+)`)?")

// symbolAnchorRE matches a `file.go#Symbol` anchor, which names the symbol and leaves its line to the test, so the note
// does not drift when code above it changes. Symbol is a name, or Type.Method for a method.
var symbolAnchorRE = regexp.MustCompile("`([\\w./-]+\\.go)#([\\w.]+)`")

// declLine is the 1-based number of the first line of src that declares sym: a function, a method (Type.Method), a
// var, const or type, or a name opening a line inside a var or const block or a closure assignment. It is 0 when none.
func declLine(src, sym string) int {
	typ, name, isMethod := strings.Cut(sym, ".")
	if !isMethod {
		typ, name = "", sym
	}
	n := regexp.QuoteMeta(name)
	var patterns []*regexp.Regexp
	if typ != "" {
		patterns = append(patterns, regexp.MustCompile(`^func \((?:\w+ )?\*?`+regexp.QuoteMeta(typ)+`(?:\[[^\]]*\])?\)\s+`+n+`\b`))
	} else {
		patterns = append(patterns,
			regexp.MustCompile(`^func (?:\([^)]*\)\s+)?`+n+`[\[(]`),
			regexp.MustCompile(`^(?:var|const|type) `+n+`\b`),
			regexp.MustCompile(`^\s+`+n+`\s*(?::=|=|[A-Za-z\[*])`),
		)
	}
	for i, line := range strings.Split(src, "\n") {
		for _, p := range patterns {
			if p.MatchString(line) {
				return i + 1
			}
		}
	}
	return 0
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for ; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		if dir == filepath.Dir(dir) {
			t.Fatal("no go.mod above the test")
		}
	}
}

// resolve finds the file an anchor names: docs/security.md writes paths relative to the repository, to internal/, or by a
// unique trailing path. It answers the slash-separated path inside root.
func resolve(t *testing.T, root *os.Root, rel string) string {
	t.Helper()
	for _, candidate := range []string{rel, "internal/" + rel} {
		if info, err := root.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	var found []string
	err := fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Build output is skipped too: the gate rebuilds site/ and dist/ while this test walks.
		if d.IsDir() && slices.Contains([]string{"node_modules", ".git", ".claude", "tmp", "site", "dist", "bin"}, d.Name()) {
			return fs.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(path, "/"+rel) {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("anchor path %q matches %d files", rel, len(found))
	}
	return found[0]
}

// TestSecurityAnchorsPointAtTheirSymbol keeps docs/security.md's file:line index honest: each anchor must land on a line
// of its file, and when the row names the symbol it guards, that symbol must be on that line.
func TestSecurityAnchorsPointAtTheirSymbol(t *testing.T) {
	t.Parallel()
	root, err := os.OpenRoot(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	doc, err := root.ReadFile("docs/security.md")
	if err != nil {
		t.Fatal(err)
	}
	anchors := anchorRE.FindAllStringSubmatch(string(doc), -1)
	if len(anchors) == 0 {
		t.Fatal("no anchors found in docs/security.md")
	}
	for _, m := range anchors {
		rel, sym := m[1], m[3]
		n, err := strconv.Atoi(m[2])
		if err != nil {
			t.Fatal(err)
		}
		src, err := root.ReadFile(resolve(t, root, rel))
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(src), "\n")
		if n < 1 || n > len(lines) {
			t.Errorf("%s:%d is past the end of the file (%d lines)", rel, n, len(lines))
			continue
		}
		if sym == "" {
			continue
		}
		name := sym[strings.LastIndex(sym, ".")+1:]
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(lines[n-1]) {
			t.Errorf("%s:%d does not hold %s: %q", rel, n, sym, strings.TrimSpace(lines[n-1]))
		}
	}
	symbols := symbolAnchorRE.FindAllStringSubmatch(string(doc), -1)
	if len(symbols) == 0 {
		t.Fatal("no symbol anchors found in docs/security.md")
	}
	for _, m := range symbols {
		src, err := root.ReadFile(resolve(t, root, m[1]))
		if err != nil {
			t.Fatal(err)
		}
		if declLine(string(src), m[2]) == 0 {
			t.Errorf("%s declares no %s", m[1], m[2])
		}
	}
}

func TestDeclLineIsStrict(t *testing.T) {
	t.Parallel()
	src := "package p\n\nfunc plain(x int) {}\n\nfunc (c *Code) Parse() {}\n\nvar (\n\tlimit = 3\n)\n\nconst top = 1\n\ntype shape struct{}\n"
	for sym, want := range map[string]int{"plain": 3, "Code.Parse": 5, "limit": 8, "top": 11, "shape": 13, "missing": 0, "Other.Parse": 0, "Parse": 5} {
		if got := declLine(src, sym); got != want {
			t.Errorf("declLine(%q) = %d, want %d", sym, got, want)
		}
	}
}
