package devsvc

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-nexusmods"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const nexusName = "Nexus Mods"

// nexusSignedIn is an error that asks for a sign-in when no Nexus Mods account is signed in.
func nexusSignedIn() error {
	if !nexussvc.HasSession() {
		return usererr.Wrap(usererr.Permission, errors.New("sign in to Nexus Mods in Settings, under Accounts"))
	}
	return nil
}

func nexusPage(gameID string, ids NexusIDs) string {
	modID, err := strconv.Atoi(ids.ModID)
	if err != nil {
		return ""
	}
	g, _ := components.Game(gameID)
	return nexusmods.ModURL(g.NexusDomain(), modID) + "?tab=files"
}

// uploadNexus sends the package as the next version of the mod file ids.FileID and adds changelog under the mod page.
func (s *Service) uploadNexus(ctx context.Context, path, name string, ids NexusIDs, version, changelog string) error {
	c := nexussvc.WithSession(nexusmods.New("Mortar", s.Version))
	c.HTTP = s.httpClient()
	c.BaseURL = s.nexusAPI
	err := nexusmods.NewUploader(c).Upload(ctx, nexusmods.Release{
		Path: path, ModFileID: ids.FileID, Name: name, Version: version, ModID: ids.ModID, Changelog: changelog,
	})
	return siteError(nexusName, err)
}

// siteError is a site's upload failure as a user-facing one: refused credentials ask for new ones, a busy site is
// retried later, and a failing site is a network error.
func siteError(site string, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := errors.AsType[*nexusmods.ChangelogError](err); ok {
		return errors.New(site + ": the file is up, but its changelog was not added; do not upload again, add the changelog on the mod page")
	}
	if rl, ok := errors.AsType[*httpkit.RateLimitError](err); ok {
		return &source.BusyError{Source: site, Reset: rl.Reset}
	}
	if errors.Is(err, nexusmods.ErrUnauthorized) {
		return usererr.Wrap(usererr.Permission, errors.New(site+" refused the credentials: enter them again"))
	}
	if errors.Is(err, nexusmods.ErrUploadPending) {
		return errors.New(site + " is still processing the upload; try again in a few minutes")
	}
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok {
		switch {
		case se.Code == http.StatusUnauthorized || se.Code == http.StatusForbidden:
			return usererr.Wrap(usererr.Permission, errors.New(site+" refused the credentials, or the account has no rights to that mod or project: enter them again or check the id"))
		case se.Code >= http.StatusInternalServerError:
			return usererr.Wrap(usererr.Network, err)
		}
		return usererr.Wrap(usererr.Invalid, err)
	}
	return err
}
