// Package tsupload publishes a package zip to Thunderstore with a team's service account token kept in the keyring.
package tsupload

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Keyring is the keyring service a team's token is kept under, named by the team: the template's own publish reads
// the same item.
const Keyring = "thunderstore"

// Site is Thunderstore's address.
const Site = "https://thunderstore.io/"

// Client refuses a redirect to another host or scheme, which would carry the token along. Its nil Transport is
// http.DefaultTransport at request time, the one that names Mortar to every site.
var Client = httpkit.WithRedirect(nil, httpkit.SameOrigin)

// Community is the game's Thunderstore community key, empty when it has none.
func Community(gameID string) string {
	g, _ := components.Game(gameID)
	src, _ := g.Source("thunderstore")
	return src.Key
}

// SetToken keeps team's service account token.
func SetToken(team, token string) error {
	if team == "" || token == "" {
		return usererr.Wrap(usererr.Invalid, errors.New("the team and the token are needed"))
	}
	return secret.SetIn(Keyring, team, token)
}

// HasToken reports whether team has a token kept.
func HasToken(team string) bool {
	token, err := secret.GetIn(Keyring, team)
	return err == nil && token != ""
}

// ForgetToken removes team's kept token.
func ForgetToken(team string) error {
	return secret.DeleteIn(Keyring, team)
}

// Upload sends the zip at zipPath with team's kept token and publishes it under team in the community, returning the
// package's page. A missing token is a Permission error, as is one Thunderstore refuses.
func Upload(ctx context.Context, client *http.Client, site, zipPath, team, community string, categories []string) (string, error) {
	token, err := secret.GetIn(Keyring, team)
	if err != nil || token == "" {
		return "", usererr.Wrap(usererr.Permission, errors.New("add a Thunderstore service account token first"))
	}
	up := thunderstore.NewUploader(client, site, token)
	uuid, err := up.Upload(ctx, zipPath)
	if err != nil {
		return "", refusedToken(err)
	}
	page, err := up.Submit(ctx, uuid, team, community, categories)
	return page, refusedToken(err)
}

func refusedToken(err error) error {
	var status *httpkit.StatusError
	if errors.As(err, &status) && (status.Code == http.StatusUnauthorized || status.Code == http.StatusForbidden) {
		return usererr.Wrap(usererr.Permission, fmt.Errorf("thunderstore refused the token: %w", err))
	}
	return err
}
