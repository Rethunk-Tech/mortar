package sandbox

import (
	"regexp"
	"strings"
)

var (
	// refusedRe is a log line saying a path was refused; missingRe one saying it was not there, which in the sandbox is
	// also what a hidden path looks like.
	refusedRe = regexp.MustCompile(`(?i)access to the path .* is denied|permission denied|read-only file system|\bEACCES\b|\bEROFS\b|errno 13|UnauthorizedAccess`)
	missingRe = regexp.MustCompile(`(?i)could not find (?:a part of )?the path|DirectoryNotFound|FileNotFound|no such file or directory|\bENOENT\b`)
	// pathRe is a path in a place the sandbox empties, as a Unix path or as Wine's Z: drive spells it.
	pathRe = regexp.MustCompile(`(?:Z:)?[\\/](?:home|root|mnt|media|run[\\/]media)(?:[\\/][^\s'"<>|]*)?`)
)

// Denials are the paths in logs that a sandboxed game most likely reached for and could not have: refused or missing,
// in a folder the sandbox empties, outside writable. A missing path counts only under home, since the read-only
// folders mounted elsewhere (a Steam library) legitimately lack files. Each path is listed once, in log order.
func Denials(logs []string, writable []string, home string) []string {
	var out []string
	seen := map[string]bool{}
	for _, log := range logs {
		for line := range strings.SplitSeq(log, "\n") {
			refused, missing := refusedRe.MatchString(line), missingRe.MatchString(line)
			if !refused && !missing {
				continue
			}
			for _, raw := range pathRe.FindAllString(line, -1) {
				p := strings.TrimRight(strings.ReplaceAll(strings.TrimPrefix(raw, "Z:"), `\`, "/"), ".,:;)")
				if seen[p] || within(p, writable) || !refused && !within(p, []string{home}) {
					continue
				}
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

func within(p string, dirs []string) bool {
	for _, d := range dirs {
		if d != "" && (p == d || strings.HasPrefix(p, strings.TrimRight(d, "/")+"/")) {
			return true
		}
	}
	return false
}
