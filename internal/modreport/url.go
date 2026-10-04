package modreport

import (
	"net/url"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/nexus"
)

const maxGitHubBody = 6000

// GitHubIssueURL opens a new issue on repo ("owner/name") with a prefilled title and body truncated to maxGitHubBody.
func GitHubIssueURL(repo, title, body string) string {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return ""
	}
	if len(body) > maxGitHubBody {
		body = body[:maxGitHubBody]
	}
	q := url.Values{
		"title": {title},
		"body":  {body},
	}
	return "https://github.com/" + repo + "/issues/new?" + q.Encode()
}

// NexusBugsURL is the mod's bug tracker tab on Nexus; domain is the v1 site segment (e.g. stardewvalley).
func NexusBugsURL(domain string, modID int) string {
	if domain == "" || modID <= 0 {
		return ""
	}
	return nexus.ModURL(domain, modID) + "?tab=bugs"
}
