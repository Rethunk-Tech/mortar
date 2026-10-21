package opener

import (
	"bytes"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// The system handler is reached from three places only, each with its own guard: the opener (web addresses),
// nexussvc's sign-in page (opener.Web first) and datadir.Open (a local path checked by checkOpenable). Anything new
// that starts a handler has to be added here on purpose.
var handlerStarts = regexp.MustCompile(`Browser\.(OpenURL|OpenFile)\(|"xdg-open"|"explorer"|rundll32|ShellExecute|"gio",\s*"open"`)

// handlerWords holds a word every handlerStarts match contains; a file with none is skipped before the regexp, which
// the race detector slows twentyfold over the repo's Go source.
var handlerWords = [][]byte{[]byte("Browser."), []byte("xdg-open"), []byte("explorer"), []byte("rundll32"), []byte("ShellExecute"), []byte(`"gio"`)}

func rel0(root, path string) string {
	return filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator)))
}

func TestOnlyTheGuardedCallersStartTheSystemHandler(t *testing.T) {
	root := filepath.Join("..", "..")
	allowed := map[string]bool{
		"main.go":                          true, // the opener service's Open
		"internal/nexussvc/sso.go":         true, // opener.Web before Browser.OpenURL
		"wailshost.go":                     true, // the host adapter; its only caller, nexussvc, runs opener.Web first
		"internal/datadir/open.go":         true, // checkOpenable before xdg-open or explorer
		"internal/lan/firewall_windows.go": true, // elevates Mortar's own executable with a fixed argument
		"webview2_windows.go":              true, // opener.Web on Microsoft's fixed download address, before any window exists
		"internal/game/epiclink.go":        true, // the Epic launcher's fixed scheme around an AppName checked as a plain name
		"internal/game/launcherlink.go":    true, // Heroic's fixed scheme around a runner from a two-name list and an AppName checked as a plain name; the arguments are query-escaped
		"internal/opener/callers_test.go":  true,
	}
	var offenders []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// tmp holds gitignored scratch (tens of thousands of files, none of it shipped code); walking it costs seconds
		// and its churn keeps go test from caching this package.
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "frontend" || rel0(root, path) == "tmp") {
			return fs.SkipDir
		}
		rel := rel0(root, path)
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || allowed[rel] {
			return nil
		}
		b, err := fsx.ReadFile(path)
		if err != nil {
			return err
		}
		if slices.ContainsFunc(handlerWords, func(w []byte) bool { return bytes.Contains(b, w) }) && handlerStarts.Match(b) {
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
