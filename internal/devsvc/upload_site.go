package devsvc

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// maxSiteReply is the most of a site's answer Mortar reads: every answer it parses is a short JSON object.
const maxSiteReply = 1 << 20

// siteError is the error for a site's refusal, nil under 400. A refused credential (401, or 403 when the account lacks
// rights to the target) asks for a new one, and a busy site (429) is retried later.
func siteError(site, what string, res *http.Response, reply []byte) error {
	switch {
	case res.StatusCode < http.StatusBadRequest:
		return nil
	case res.StatusCode == http.StatusUnauthorized:
		return usererr.Wrap(usererr.Permission, fmt.Errorf("%s refused the credentials: enter them again", site))
	case res.StatusCode == http.StatusForbidden:
		return usererr.Wrap(usererr.Permission, fmt.Errorf("%s refused the credentials, or the account has no rights to that mod or project: enter them again or check the id", site))
	case res.StatusCode == http.StatusTooManyRequests:
		return source.Busy(site, res)
	}
	text := strings.TrimSpace(string(reply))
	if len(text) > 300 {
		text = text[:300]
	}
	return fmt.Errorf("%s %s: %s: %s", site, what, res.Status, text)
}

// storageURL is a URL a site handed out to upload bytes to. It carries no credential of Mortar's, but the file must not
// cross the internet in the clear: only https is followed, or plain http when the API itself was reached that way (tests).
func storageURL(raw, apiBase string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("the site gave an unusable upload address")
	}
	if api, err := url.Parse(apiBase); u.Scheme != "https" && (err != nil || api.Scheme != "http" || u.Host != api.Host) {
		return "", fmt.Errorf("the site gave an upload address that is not https")
	}
	return raw, nil
}
