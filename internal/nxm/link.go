// Package nxm reads nxm:// links and registers Mortar as the system handler of the scheme.
package nxm

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/nexus"
)

// Link is an accepted nxm:// download link.
type Link struct {
	ModID   int    `json:"modId"`
	FileID  int    `json:"fileId"`
	Key     string `json:"key"`
	Expires int64  `json:"expires"`
	UserID  int    `json:"userId"`
}

// Why a link was refused. The frontend words each in the user's language.
const (
	ReasonForm     = "form"
	ReasonGame     = "game"
	ReasonExpired  = "expired"
	ReasonUser     = "user"
	ReasonSignedIn = "signedIn"
)

// RejectError says why a link was refused.
type RejectError struct{ Reason string }

func (e *RejectError) Error() string { return "nxm link refused: " + e.Reason }

func reject(reason string) error { return &RejectError{Reason: reason} }

// IsLink reports whether an argument is an nxm:// link, whatever its content.
func IsLink(arg string) bool { return strings.HasPrefix(strings.ToLower(arg), "nxm://") }

// Parse accepts nxm://stardewvalley/mods/<mod id>/files/<file id>?key=&expires=&user_id= only: numeric ids, a key,
// not yet expired at now, and user_id equal to userID, the signed-in account. Anything else is a *RejectError.
func Parse(raw string, userID int, now time.Time) (Link, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "nxm" || u.User != nil || u.Port() != "" || u.Fragment != "" {
		return Link{}, reject(ReasonForm)
	}
	if u.Host != nexus.Game {
		return Link{}, reject(ReasonGame)
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "mods" || parts[2] != "files" {
		return Link{}, reject(ReasonForm)
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(q) != 3 {
		return Link{}, reject(ReasonForm)
	}
	var l Link
	if l.ModID, err = number(parts[1]); err != nil {
		return Link{}, err
	}
	if l.FileID, err = number(parts[3]); err != nil {
		return Link{}, err
	}
	expires, err := number(q.Get("expires"))
	if err != nil {
		return Link{}, err
	}
	if l.UserID, err = number(q.Get("user_id")); err != nil {
		return Link{}, err
	}
	l.Expires, l.Key = int64(expires), q.Get("key")
	if l.Key == "" || len(q["key"]) != 1 {
		return Link{}, reject(ReasonForm)
	}
	if userID == 0 {
		return Link{}, reject(ReasonSignedIn)
	}
	if l.UserID != userID {
		return Link{}, reject(ReasonUser)
	}
	if l.Expires <= now.Unix() {
		return Link{}, reject(ReasonExpired)
	}
	return l, nil
}

// number reads a positive decimal integer, no sign, spaces or leading junk.
func number(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 || strconv.Itoa(n) != s {
		return 0, reject(ReasonForm)
	}
	return n, nil
}
