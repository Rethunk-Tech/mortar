package source

import (
	"net/url"
	"strings"
	"time"
)

// Limits every searching driver shares.
const (
	PageSize       = 20
	RequestTimeout = 20 * time.Second
	MaxBody        = 16 << 20
)

// UserAgent names Mortar to a site.
func UserAgent(version string) string {
	if version == "" {
		return "Mortar"
	}
	return "Mortar/" + version
}

// FirstPage is the page number a search starts at.
const FirstPage = 1

// GitHubRepo reads "owner/repo" from a github.com repository address, or "" when raw is not one.
func GitHubRepo(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Hostname() != "github.com" && u.Hostname() != "www.github.com") {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	return parts[0] + "/" + strings.TrimSuffix(parts[1], ".git")
}
