package cli

import (
	"os"
	"regexp"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// guiSentences reads the GUI's kind-to-sentence table out of its source, keyed by wire kind; "" holds the default.
func guiSentences(t *testing.T) map[string]string {
	t.Helper()
	src, err := os.ReadFile("../../frontend/src/toasts/report.ts")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	re := regexp.MustCompile("(?:case '(\\w+)':|default:)\\s*return i18n\\._\\(\\s*msg`([^`]*)`")
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		out[m[1]] = m[2]
	}
	return out
}

func TestSentenceMatchesGUIForEveryKind(t *testing.T) {
	gui := guiSentences(t)
	for _, k := range usererr.Kinds {
		want, ok := gui[string(k)]
		if k == usererr.Unknown {
			want, ok = gui[""], true
		}
		if !ok {
			t.Errorf("the GUI has no sentence for %q", k)
			continue
		}
		if got := Sentence(k); got != want {
			t.Errorf("Sentence(%q) = %q, GUI says %q", k, got, want)
		}
	}
	if Sentence("brand_new") != gui[""] {
		t.Errorf("an unlisted kind should read as the default sentence")
	}
}
