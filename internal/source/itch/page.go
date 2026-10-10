package itch

import (
	"net/url"
	"regexp"
	"strings"
)

// A page is named by the account and game slug of its address, "<user>.itch.io/<game>", written "user/game". A game
// on its own domain has no such address and cannot be named.
var (
	userPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,38}$`)
	slugPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,99}$`)
)

// ValidPage reports whether id is a page name as ParsePageURL returns it.
func ValidPage(id string) bool {
	user, game, ok := strings.Cut(id, "/")
	return ok && userPattern.MatchString(user) && slugPattern.MatchString(game)
}

// PageURL is the canonical address of a page, or "" for an id that is not one.
func PageURL(id string) string {
	if !ValidPage(id) {
		return ""
	}
	user, game, _ := strings.Cut(id, "/")
	return "https://" + user + ".itch.io/" + game
}

// ParsePageURL reads a pasted game page address: https only, host <user>.itch.io with no user name or port, and a path
// of the one game slug. It returns "user/game"; the address that opens is always rebuilt from it, so a query,
// fragment or look-alike host never reaches the browser.
func ParsePageURL(text string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(text))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return "", false
	}
	user, found := strings.CutSuffix(strings.ToLower(u.Hostname()), ".itch.io")
	if !found || user == "www" || !userPattern.MatchString(user) {
		return "", false
	}
	game := strings.TrimSuffix(u.EscapedPath(), "/")
	game, ok := strings.CutPrefix(game, "/")
	if !ok || !slugPattern.MatchString(game) {
		return "", false
	}
	return user + "/" + game, true
}
