package pack

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// GaleBaseURL is the server behind Gale's profile sync, which hands out the short codes.
const GaleBaseURL = "https://gale.kesomannen.com/api"

// galeCodeID is the id Gale's server makes: six upper-case letters or digits.
var galeCodeID = regexp.MustCompile(`^[A-Z0-9]{6}$`)

// GaleCode reads a Gale sync code, for example "E5RG8F". The server answers a download with a redirect to the profile's
// .r2z on Thunderstore, and needs no account to read it.
type GaleCode struct {
	HTTP *http.Client
	// URL is the sync server's API address; empty means the real one.
	URL string
}

// ID names the format.
func (GaleCode) ID() string { return "gale-code" }

// Detect accepts only a pasted text of exactly Gale's code shape.
func (GaleCode) Detect(in Input) bool {
	return in.Path == "" && galeCodeID.MatchString(strings.TrimSpace(in.Text))
}

// Parse downloads the code's profile and reads it like an .r2z.
func (g GaleCode) Parse(ctx context.Context, in Input) (Draft, error) {
	id := strings.TrimSpace(in.Text)
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
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxInput+1))
	if err != nil {
		return Draft{}, err
	}
	if len(b) > maxInput {
		return Draft{}, errors.New("profile code is too large")
	}
	return parseR2Zip(b)
}
