package pack

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"

	"github.com/Rethunk-Tech/go-modmanagers/gale"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// GaleBaseURL is the server behind Gale's profile sync, which hands out the short codes.
const GaleBaseURL = "https://gale.kesomannen.com/api"

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
	_, ok := gale.CodeID(in.Text)
	return in.Path == "" && ok
}

// Parse downloads the code's profile and reads it like an .r2z.
func (g GaleCode) Parse(ctx context.Context, in Input) (Draft, error) {
	id, _ := gale.CodeID(in.Text)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	client := gale.Client{HTTP: g.HTTP, BaseURL: cmp.Or(g.URL, GaleBaseURL), UserAgent: source.UserAgentWithSite(""), MaxBytes: maxInput}
	b, err := client.FetchCode(ctx, id)
	if errors.Is(err, safefs.ErrTooLarge) {
		return Draft{}, errors.New("profile code is too large")
	}
	if errors.Is(err, httpkit.ErrNotFound) {
		return Draft{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("gale has no profile code %s", id))
	}
	if err != nil {
		return Draft{}, fmt.Errorf("gale profile code %s: %w", id, err)
	}
	return parseR2Zip(b)
}
