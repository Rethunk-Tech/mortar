// Package opener is the one way Mortar hands an address to the system's URL handler. A page, a requirement, a changelog
// link, a collection or a share can all carry a link a stranger wrote, and the handler also launches other schemes'
// apps (steam://, file://, a custom scheme), so remote data never chooses the scheme: web addresses open, and the few
// other schemes Mortar itself needs are separate, argument-checked calls.
package opener

import (
	"errors"
	"log"
	"net/url"
	"strings"
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// maxURL bounds what is passed to the handler; longer is not a page address.
const maxURL = 2048

// Web checks that raw is an http or https address with a host and returns it for the handler. A scheme other than
// those, a missing host, a user name or password, control characters or spaces are refused.
func Web(raw string) (string, error) {
	if raw == "" || len(raw) > maxURL || strings.ContainsFunc(raw, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) {
		return "", refused(raw)
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
		return "", refused(raw)
	}
	return raw, nil
}

func refused(raw string) error {
	if len(raw) > 80 {
		raw = raw[:80] + "…"
	}
	log.Printf("opener: refused %q: only http and https addresses open", raw)
	return usererr.New(usererr.Invalid, "Mortar only opens http and https links.")
}

// SteamValidate is the steam://validate address of a Steam app, which asks Steam to verify the game's files. appID must
// be digits: it comes from the catalog, which is remote data.
func SteamValidate(appID string) (string, error) {
	if appID == "" || len(appID) > 12 || strings.TrimLeft(appID, "0123456789") != "" {
		return "", errors.New("not a Steam app id")
	}
	return "steam://validate/" + appID, nil
}

// Service is the opener the window calls. Open is the system handler (Wails' Browser.OpenURL).
type Service struct {
	Open func(string) error
}

// OpenWeb opens an http or https address in the default browser and refuses anything else.
func (s *Service) OpenWeb(raw string) error {
	u, err := Web(raw)
	if err != nil {
		return err
	}
	return s.Open(u)
}

// OpenSteamValidate asks Steam to verify an app's files. It exists apart from OpenWeb because steam:// is the one
// non-web scheme the window needs, and only for a numeric app id.
func (s *Service) OpenSteamValidate(appID string) error {
	u, err := SteamValidate(appID)
	if err != nil {
		return usererr.Wrap(usererr.Invalid, err)
	}
	return s.Open(u)
}
