package docscheck

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

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

func TestDeclLineIsStrict(t *testing.T) {
	t.Parallel()
	src := "package p\n\nfunc plain(x int) {}\n\nfunc (c *Code) Parse() {}\n\nvar (\n\tlimit = 3\n)\n\nconst top = 1\n\ntype shape struct{}\n"
	for sym, want := range map[string]int{"plain": 3, "Code.Parse": 5, "limit": 8, "top": 11, "shape": 13, "missing": 0, "Other.Parse": 0, "Parse": 5} {
		if got := declLine(src, sym); got != want {
			t.Errorf("declLine(%q) = %d, want %d", sym, got, want)
		}
	}
}

var (
	testNameRE = regexp.MustCompile(`^func ((?:Test|Fuzz)\w+)\(`)
	// refRE matches a backticked test name, or the `(`package`)` label that closes a list of names.
	refRE      = regexp.MustCompile("`((?:Test|Fuzz)\\w+)`|\\(`([\\w./-]+)`[):]")
	linkRE     = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)
	headingRE  = regexp.MustCompile(`^#{1,6}\s+(.*?)\s*#*$`)
	mdLinkText = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	slugDrop   = regexp.MustCompile(`[^\p{L}\p{N} _-]`)
	matrixIDRE = regexp.MustCompile("`([a-z]+\\.[a-z][a-z0-9-]*)`")
)

type repoFiles struct {
	root  string
	fs    *os.Root
	files []string // slash-separated, relative to root
}

func scanRepo(t *testing.T) repoFiles {
	t.Helper()
	root := repoRoot(t)
	skip := []string{"node_modules", ".git", ".claude", "tmp", "site", "dist", "bin"}
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && slices.Contains(skip, d.Name()) {
			return fs.SkipDir
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(root, p)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	fsRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fsRoot.Close() })
	return repoFiles{root, fsRoot, files}
}

func (r repoFiles) read(t *testing.T, rel string) string {
	t.Helper()
	b, err := r.fs.ReadFile(rel)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func (r repoFiles) has(rel string) bool { return slices.Contains(r.files, rel) }

// resolve finds the file an anchor names: the docs write paths relative to the repository, to internal/, or by a unique
// trailing path.
func (r repoFiles) resolve(rel string) (string, string) {
	for _, c := range []string{rel, "internal/" + rel} {
		if r.has(c) {
			return c, ""
		}
	}
	var found []string
	for _, f := range r.files {
		if strings.HasSuffix(f, "/"+rel) {
			found = append(found, f)
		}
	}
	switch len(found) {
	case 1:
		return found[0], ""
	case 0:
		return "", "no such file"
	}
	return "", "matches several files"
}

func docFiles(t *testing.T, r repoFiles) []string {
	t.Helper()
	docs := []string{"HUMANS.md", "AGENTS.md"}
	for _, f := range r.files {
		if strings.HasPrefix(f, "docs/") && strings.HasSuffix(f, ".md") && strings.Count(f, "/") == 1 {
			docs = append(docs, f)
		}
	}
	sort.Strings(docs)
	return docs
}

func report(t *testing.T, problems []string) {
	t.Helper()
	if len(problems) > 0 {
		t.Errorf("%d bad references:\n%s", len(problems), strings.Join(problems, "\n"))
	}
}

// goTestDirs maps each Test/Fuzz function to the directories whose *_test.go files declare it.
func goTestDirs(t *testing.T, r repoFiles) map[string][]string {
	t.Helper()
	dirs := map[string][]string{}
	for _, f := range r.files {
		if !strings.HasSuffix(f, "_test.go") {
			continue
		}
		for line := range strings.SplitSeq(r.read(t, f), "\n") {
			if m := testNameRE.FindStringSubmatch(line); m != nil && !slices.Contains(dirs[m[1]], path.Dir(f)) {
				dirs[m[1]] = append(dirs[m[1]], path.Dir(f))
			}
		}
	}
	return dirs
}

// TestDocsNameRealTests: a backticked Test or Fuzz name exists, and when a `(package)` label closes the list of names it
// sits in, each name is declared in that package's directory.
func TestDocsNameRealTests(t *testing.T) {
	t.Parallel()
	r := scanRepo(t)
	dirs := goTestDirs(t, r)
	var problems []string
	for _, doc := range docFiles(t, r) {
		for i, line := range strings.Split(r.read(t, doc), "\n") {
			for cell := range strings.SplitSeq(line, "|") {
				var pending []string
				for _, m := range refRE.FindAllStringSubmatch(cell, -1) {
					if m[1] != "" {
						pending = append(pending, m[1])
						if _, ok := dirs[m[1]]; !ok {
							problems = append(problems, doc+":"+strconv.Itoa(i+1)+": "+m[1]+" is declared in no *_test.go")
						}
						continue
					}
					label := strings.TrimSuffix(m[2], "_test.go")
					if label != m[2] {
						label = path.Dir(m[2])
					}
					for _, name := range pending {
						if got := dirs[name]; got != nil && !slices.ContainsFunc(got, func(d string) bool { return d == label || strings.HasSuffix(d, "/"+label) }) {
							problems = append(problems, doc+":"+strconv.Itoa(i+1)+": "+name+" is labelled ("+m[2]+") but declared in "+strings.Join(got, ", "))
						}
					}
					pending = nil
				}
			}
		}
	}
	report(t, problems)
}

// TestDocsSymbolAnchorsResolve: every `file.go#Symbol` names a file that declares the symbol.
func TestDocsSymbolAnchorsResolve(t *testing.T) {
	t.Parallel()
	r := scanRepo(t)
	var problems []string
	seen := 0
	for _, doc := range docFiles(t, r) {
		for i, line := range strings.Split(r.read(t, doc), "\n") {
			for _, m := range symbolAnchorRE.FindAllStringSubmatch(line, -1) {
				seen++
				where := doc + ":" + strconv.Itoa(i+1) + ": " + m[1] + "#" + m[2]
				file, why := r.resolve(m[1])
				switch {
				case why != "":
					problems = append(problems, where+": "+why)
				case declLine(r.read(t, file), m[2]) == 0:
					problems = append(problems, where+": "+file+" declares no "+m[2])
				}
			}
		}
	}
	if seen == 0 {
		t.Fatal("no symbol anchors found in the docs")
	}
	report(t, problems)
}

// TestBepInExMatrixStepsExistInTheRegress: each step id the matrix cites is one scripts/regress-bepinex.sh reports.
func TestBepInExMatrixStepsExistInTheRegress(t *testing.T) {
	t.Parallel()
	r := scanRepo(t)
	script := r.read(t, "scripts/regress-bepinex.sh")
	var problems []string
	for i, line := range strings.Split(r.read(t, "docs/bepinex-test-matrix.md"), "\n") {
		if !strings.Contains(line, "launch (") && !strings.Contains(line, "regress") {
			continue
		}
		for _, m := range matrixIDRE.FindAllStringSubmatch(line, -1) {
			prefix, ext, _ := strings.Cut(m[1], ".")
			if slices.Contains([]string{"tsv", "json", "log", "txt"}, ext) {
				continue
			}
			// A step reported in a loop is written once as prefix.$var.
			if !regexp.MustCompile(`[\s"':]`+regexp.QuoteMeta(m[1])+`[\s"':]`).MatchString(script) && !strings.Contains(script, prefix+".$") {
				problems = append(problems, "docs/bepinex-test-matrix.md:"+strconv.Itoa(i+1)+": step "+m[1]+" is not in scripts/regress-bepinex.sh")
			}
		}
	}
	report(t, problems)
}

// slugs are the anchors GitHub gives a markdown file's headings: lowercase, punctuation dropped, spaces to hyphens, a
// repeated heading numbered.
func slugs(src string) map[string]bool {
	out := map[string]bool{}
	count := map[string]int{}
	fenced := false
	for line := range strings.SplitSeq(src, "\n") {
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
		}
		m := headingRE.FindStringSubmatch(line)
		if fenced || m == nil {
			continue
		}
		text := mdLinkText.ReplaceAllString(m[1], "$1")
		text = strings.NewReplacer("`", "", "*", "").Replace(text)
		slug := strings.ReplaceAll(slugDrop.ReplaceAllString(strings.ToLower(text), ""), " ", "-")
		if n := count[slug]; n > 0 {
			out[slug+"-"+strconv.Itoa(n)] = true
		} else {
			out[slug] = true
		}
		count[slug]++
	}
	return out
}

// TestDocsLinksReachTheirHeading: a markdown link to a file or `#anchor` lands on a file and heading that exist.
func TestDocsLinksReachTheirHeading(t *testing.T) {
	t.Parallel()
	r := scanRepo(t)
	var problems []string
	for _, doc := range docFiles(t, r) {
		fenced := false
		for i, line := range strings.Split(r.read(t, doc), "\n") {
			if strings.HasPrefix(line, "```") {
				fenced = !fenced
			}
			if fenced {
				continue
			}
			for _, m := range linkRE.FindAllStringSubmatch(line, -1) {
				target, anchor, _ := strings.Cut(m[1], "#")
				if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
					continue
				}
				file := doc
				if target != "" {
					file = path.Join(path.Dir(doc), target)
				}
				where := doc + ":" + strconv.Itoa(i+1) + ": " + m[1]
				switch {
				case !r.has(file):
					problems = append(problems, where+": no such file")
				case anchor != "" && strings.HasSuffix(file, ".md") && !slugs(r.read(t, file))[anchor]:
					problems = append(problems, where+": no heading "+anchor)
				}
			}
		}
	}
	report(t, problems)
}
