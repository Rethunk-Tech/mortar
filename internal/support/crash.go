package support

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/Rethunk-Tech/go-safefs"
)

const (
	crashLogName  = "crash.log"
	crashSeenName = "crash.seen"
	prevLogName   = "mortar.prev.log"
	logTailLines  = 40
)

// lastRunCrashed is set by DetectLastRunCrashed at process start so LastRunCrashed
// stays true for this run without re-reading files the frontend might call twice.
var lastRunCrashed atomic.Bool

// DetectLastRunCrashed reports whether the previous process ended unexpectedly:
// crash.log larger than the byte count in crash.seen, or mortar.prev.log whose
// has no clean-shutdown record. crash.seen is written after the
// check so a given growth is reported once.
func DetectLastRunCrashed(dataDir string) bool {
	crashPath := filepath.Join(dataDir, crashLogName)
	grew := crashLogGrew(crashPath, filepath.Join(dataDir, crashSeenName))
	writeCrashSeen(filepath.Join(dataDir, crashSeenName), crashPath)
	crashed := grew || prevLogUnclean(filepath.Join(dataDir, prevLogName))
	lastRunCrashed.Store(crashed)
	return crashed
}

func crashLogGrew(crashPath, seenPath string) bool {
	info, err := os.Stat(crashPath)
	if err != nil {
		return false
	}
	seen, ok := readSeenSize(seenPath)
	if !ok {
		return info.Size() > 0
	}
	if info.Size() < seen {
		return false
	}
	return info.Size() > seen
}

func readSeenSize(seenPath string) (int64, bool) {
	b, err := safefs.ReadFile(seenPath)
	if err != nil {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func writeCrashSeen(seenPath, crashPath string) {
	var size int64
	if info, err := os.Stat(crashPath); err == nil {
		size = info.Size()
	}
	_ = safefs.WriteFileAtomic(seenPath, []byte(strconv.FormatInt(size, 10)+"\n"), 0o600)
}

// maxLogLine is the longest log line prevLogUnclean reads.
const maxLogLine = 4 << 20

func prevLogUnclean(prevPath string) bool {
	f, err := safefs.Open(prevPath)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	// Late lines (a service logging as it stops) can follow the record, so it is looked for anywhere in the log.
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, maxLogLine)
	for sc.Scan() {
		line := sc.Text()
		// slog text handler: msg=shutdown clean=true
		if strings.Contains(line, "msg=shutdown") && strings.Contains(line, "clean=true") {
			return false
		}
	}
	// A line too long to scan hides the record: say nothing rather than report a crash that did not happen.
	return sc.Err() == nil
}

// LastRunCrashed reports whether the previous Mortar process ended unexpectedly, once per run: a reloaded or
// rebuilt window must not show the notice again.
func (s *Service) LastRunCrashed() bool {
	return lastRunCrashed.Swap(false)
}

func logTailSections(dataDir, home string, budget int) string {
	if dataDir == "" || budget <= 0 {
		return ""
	}
	var bodies []string
	var names []string
	for _, name := range []string{prevLogName, crashLogName} {
		b, err := safefs.ReadFile(filepath.Join(dataDir, name))
		if err != nil {
			continue
		}
		names = append(names, name)
		bodies = append(bodies, hideHomeIn(lastLines(string(b), logTailLines), home))
	}
	if len(names) == 0 {
		return ""
	}
	for {
		out := formatTailSections(names, bodies)
		if len(out) <= budget {
			return out
		}
		i := longestIndex(bodies)
		if i < 0 || bodies[i] == "" {
			return ""
		}
		bodies[i] = shrinkTail(bodies[i])
	}
}

func formatTailSections(names, bodies []string) string {
	var b strings.Builder
	for i, name := range names {
		b.WriteString("\n**")
		b.WriteString(name)
		b.WriteString("**\n```\n")
		b.WriteString(bodies[i])
		if bodies[i] != "" && !strings.HasSuffix(bodies[i], "\n") {
			b.WriteByte('\n')
		}
		b.WriteString("```\n")
	}
	return b.String()
}

func longestIndex(bodies []string) int {
	best, n := -1, 0
	for i, s := range bodies {
		if len(s) >= n {
			best, n = i, len(s)
		}
	}
	return best
}

// shrinkTail drops the oldest line of a log tail, or the older half of a single long line.
func shrinkTail(s string) string {
	s = strings.TrimRight(s, "\n")
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[i+1:]
	}
	if len(s) > 1 {
		return s[len(s)/2:]
	}
	return ""
}

// loggedURL is an address in free text. A logged request carries its secrets in the address itself: the nxm key and
// expiry, a signed CDN link's user id, digest and expiry, an API key or an access token in the query, a password in
// the user info.
var loggedURL = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[^\s"'<>]+`)

var (
	queryValue = regexp.MustCompile(`([?&;][^=&;\s#]*=)[^&;\s#]*`)
	userInfo   = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9+.-]*://)[^/@?#]*@`)
	bearer     = regexp.MustCompile(`(?i)((?:authorization|bearer|apikey|api-key)["']?\s*[:=]?\s*(?:bearer\s+)?)[A-Za-z0-9._~+/=-]{8,}`)
)

// scrubSecrets blanks the value of every query parameter and the user info of every address in text, and any
// bearer token or API key written as a header.
func scrubSecrets(text string) string {
	text = loggedURL.ReplaceAllStringFunc(text, func(u string) string {
		u = userInfo.ReplaceAllString(u, "${1}REDACTED@")
		return queryValue.ReplaceAllString(u, "${1}REDACTED")
	})
	return bearer.ReplaceAllString(text, "${1}REDACTED")
}

// hideHomeIn replaces every occurrence of the home folder in free text such as a log, where paths appear mid-line.
func hideHomeIn(text, home string) string {
	text = scrubSecrets(text)
	for _, h := range []string{home, filepath.ToSlash(home), filepath.FromSlash(home)} {
		if h = strings.TrimRight(h, `/\`); h != "" {
			text = strings.ReplaceAll(text, h, "~")
		}
	}
	return text
}
