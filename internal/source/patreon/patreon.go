// Package patreon is the source of mods their authors post to patrons. Patreon offers a patron no API for a post's
// files, and its terms bar automated access to patron content, so Mortar only names a post: the player opens it, saves
// the file in the browser and adds that file, which then remembers the post id. Nothing is searched, fetched or
// checked for updates.
package patreon

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// ID is the catalog id and the profile entry kind.
const ID = "patreon"

// maxIDDigits bounds a post id (idPattern holds the same number); Patreon's are far shorter.
const maxIDDigits = 12

// Driver is the Patreon source. It implements no Searcher, so Browse never lists it.
type Driver struct{}

var _ = source.Register(Driver{})

func (Driver) ID() string { return ID }

func (Driver) Name() string { return "Patreon" }

// Modes says the player finishes the download in the browser.
func (Driver) Modes() []source.Acquire { return []source.Acquire{source.Handoff} }

// Hosts is the site's web host, so a pasted post address traces back to it.
func (Driver) Hosts() []string { return []string{"patreon.com"} }

// ModPageURL is the post's page; id is a post id as ParsePostURL returns it.
func (Driver) ModPageURL(_, id string) string { return PostURL(id) }

var (
	idPattern   = regexp.MustCompile(`^[0-9]{1,12}$`)
	slugPattern = regexp.MustCompile(`^[A-Za-z0-9_.~-]+$`)
)

// ValidID reports whether id is a post id.
func ValidID(id string) bool { return len(id) <= maxIDDigits && idPattern.MatchString(id) }

// PostURL is the canonical address of a post, or "" for an id that is not one.
func PostURL(id string) string {
	if !ValidID(id) {
		return ""
	}
	return "https://www.patreon.com/posts/" + id
}

// ParsePostURL reads a pasted post address: https only, host patreon.com or www.patreon.com, no user name or port, and
// a path of /posts/<id> or /posts/<slug>-<id>. It returns the id; the address that opens is always rebuilt from it.
func ParsePostURL(text string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(text))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return "", false
	}
	if host := strings.ToLower(u.Hostname()); host != "patreon.com" && host != "www.patreon.com" {
		return "", false
	}
	path := strings.TrimSuffix(u.EscapedPath(), "/")
	last, ok := strings.CutPrefix(path, "/posts/")
	if !ok || last == "" || strings.Contains(last, "/") {
		return "", false
	}
	id := last
	if slug, tail, found := strings.CutLast(last, "-"); found {
		if slug == "" || !slugPattern.MatchString(slug) {
			return "", false
		}
		id = tail
	}
	if !ValidID(id) {
		return "", false
	}
	return id, true
}
