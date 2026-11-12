package modreport

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/go-nexusmods"
)

const (
	maxGitHubBody = 6000
	// maxURL is the longest address Mortar's opener accepts (internal/opener maxURL).
	maxURL = 2048
)

var githubRepoPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

// validGitHubRepo mirrors manifest.ValidGitHubRepo: "." and "..", which the pattern admits, would move the URL's path.
func validGitHubRepo(repo string) bool {
	owner, name, _ := strings.Cut(repo, "/")
	return githubRepoPattern.MatchString(repo) && strings.Trim(owner, ".") != "" && strings.Trim(name, ".") != ""
}

// GitHubIssueURL opens a new issue on repo ("owner/name") with a prefilled title and the longest start of body that
// keeps the address within maxURL; cut says the body was shortened. The URL is empty for a repo that is not owner/name.
func GitHubIssueURL(repo, title, body string) (link string, cut bool) {
	repo = strings.TrimSpace(repo)
	if !validGitHubRepo(repo) {
		return "", false
	}
	base := "https://github.com/" + repo + "/issues/new?"
	at := func(n int) string {
		return base + "title=" + url.QueryEscape(title) + "&body=" + url.QueryEscape(strings.ToValidUTF8(body[:n], ""))
	}
	// Encoding grows each byte up to threefold, so the longest body that fits is found by halving.
	lo, hi := 0, min(len(body), maxGitHubBody)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if len(at(mid)) <= maxURL {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if len(at(lo)) > maxURL {
		return "", false
	}
	return at(lo), lo < len(body)
}

// NexusBugsURL is the mod's bug tracker tab on Nexus; domain is the v1 site segment.
func NexusBugsURL(domain string, modID int) string {
	if domain == "" || modID <= 0 {
		return ""
	}
	return nexusmods.ModURL(domain, modID) + "?tab=bugs"
}
