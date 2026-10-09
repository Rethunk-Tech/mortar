package docscheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// anchorRE matches a `file.go:LINE` anchor, and an immediately following `Symbol` when the row names one.
var anchorRE = regexp.MustCompile("`([\\w./-]+\\.go):(\\d+)`(?: `([\\w.]+)`)?")

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
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "tmp") {
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
}
