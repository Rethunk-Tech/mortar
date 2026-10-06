package opener

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// The system handler is reached from three places only, each with its own guard: the opener (web addresses),
// nexussvc's sign-in page (opener.Web first) and datadir.Open (a local path checked by checkOpenable). Anything new
// that starts a handler has to be added here on purpose.
var handlerStarts = regexp.MustCompile(`Browser\.(OpenURL|OpenFile)\(|"xdg-open"|"explorer"|rundll32|ShellExecute|"gio",\s*"open"`)

func TestOnlyTheGuardedCallersStartTheSystemHandler(t *testing.T) {
	root := filepath.Join("..", "..")
	allowed := map[string]bool{
		"main.go":                         true, // the opener service's Open
		"internal/nexussvc/sso.go":        true, // opener.Web before Browser.OpenURL
		"internal/datadir/open.go":        true, // checkOpenable before xdg-open or explorer
		"internal/opener/callers_test.go": true,
	}
	var offenders []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "frontend") {
			return fs.SkipDir
		}
		rel := filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator)))
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || allowed[rel] {
			return nil
		}
		b, err := fsx.ReadFile(path)
		if err != nil {
			return err
		}
		if handlerStarts.Match(b) {
			offenders = append(offenders, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) != 0 {
		t.Fatalf("these start the system handler without the opener's guard: %v", offenders)
	}
}
