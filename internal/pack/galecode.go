package pack

import (
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// GaleBaseURL is the server behind Gale's profile sync, which hands out the short codes.
const GaleBaseURL = "https://gale.kesomannen.com/api"

// galeCodeID is the id Gale's server makes: six upper-case letters or digits.
var galeCodeID = regexp.MustCompile(`^[A-Z0-9]{6}$`)

// galeLegacyID is the id older Gale profiles have: a UUID in unpadded URL-safe base64, case-sensitive.
func galeLegacyID(text string) bool {
	b, err := base64.RawURLEncoding.DecodeString(text)
	return err == nil && len(text) == 22 && len(b) == 16
}

// galeLinks are the prefixes Gale puts before a sync id: its "Copy link" address and its own deep link.
var galeLinks = []string{"https://gale.kesomannen.com/api/desktop/profile/sync/clone/", "gale://profile/sync/clone/"}

// galeID is pasted text as the id Gale's server knows it, or "" when it is neither shape. A copied link is read for
// the id at its end.
func galeID(text string) string {
	text = strings.TrimSpace(text)
	for _, prefix := range galeLinks {
		if id, ok := strings.CutPrefix(text, prefix); ok {
			text = strings.TrimSuffix(id, "/")
			break
		}
	}
	if galeLegacyID(text) {
		return text
	}
	if up := strings.ToUpper(text); galeCodeID.MatchString(up) {
		return up
	}
	return ""
}

// GaleCode reads a Gale sync code, for example "E5RG8F", or an older profile's 22-character id. The server answers a download with a redirect to the profile's
// .r2z on Thunderstore, and needs no account to read it.
type GaleCode struct {
	HTTP *http.Client
	// URL is the sync server's API address; empty means the real one.
	URL string
}

// ID names the format.
func (GaleCode) ID() string { return "gale-code" }

// Detect accepts only a pasted text of exactly one of Gale's id shapes. A short code is read in either case: people
// retype codes, and the server stores them upper-case.
func (GaleCode) Detect(in Input) bool {
	return in.Path == "" && galeID(in.Text) != ""
}

// Parse downloads the code's profile and reads it like an .r2z.
func (g GaleCode) Parse(ctx context.Context, in Input) (Draft, error) {
	id := galeID(in.Text)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cmp.Or(g.URL, GaleBaseURL)+"/profile/"+id, nil)
	if err != nil {
		return Draft{}, err
	}
	req.Header.Set("User-Agent", source.UserAgentWithSite(""))
	resp, err := cmp.Or(g.HTTP, http.DefaultClient).Do(req)
	if err != nil {
		return Draft{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return Draft{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("gale has no profile code %s", id))
	}
	if resp.StatusCode != http.StatusOK {
		return Draft{}, usererr.Wrap(usererr.Network, fmt.Errorf("gale answered %s for profile code %s", resp.Status, id))
	}
	b, err := safefs.ReadCapped(resp.Body, maxInput)
	if errors.Is(err, safefs.ErrTooLarge) {
		return Draft{}, errors.New("profile code is too large")
	}
	if err != nil {
		return Draft{}, err
	}
	return parseR2Zip(b)
}
