package devsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Licenses are the licenses New mod offers besides the template's own; none of their texts names a holder or year.
var Licenses = []string{"apache-2.0", "gpl-3.0", "mpl-2.0", "unlicense"}

// applyLicense writes the chosen license's text over the template's LICENSE, from GitHub's licenses API.
func (s *Service) applyLicense(ctx context.Context, dir, key string) error {
	if key == "" {
		return nil
	}
	if !slices.Contains(Licenses, key) {
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a license New mod offers", key))
	}
	body, err := (&gh.Client{HTTP: s.httpClient(), APIBase: strings.TrimRight(s.LicensesURL, "/")}).LicenseText(ctx, key)
	if err != nil {
		return licenseTextErr(key, err)
	}
	return safefs.WriteFile(filepath.Join(dir, "LICENSE"), []byte(body), 0o644)
}

func licenseTextErr(key string, err error) error {
	if errors.Is(err, httpkit.ErrNotFound) {
		return usererr.Wrap(usererr.Network, fmt.Errorf("the %s license text: %w", key, err))
	}
	if _, ok := errors.AsType[*httpkit.StatusError](err); ok {
		return usererr.Wrap(usererr.Network, fmt.Errorf("the %s license text: %w", key, err))
	}
	if _, ok := errors.AsType[*httpkit.RateLimitError](err); ok {
		return usererr.Wrap(usererr.Network, fmt.Errorf("the %s license text: %w", key, err))
	}
	var ue *url.Error
	if errors.As(err, &ue) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return usererr.Wrap(usererr.Network, fmt.Errorf("the %s license text: %w", key, err))
	}
	return usererr.Wrap(usererr.Network, errors.New("the license text could not be read"))
}

// SDKNeed is the SDK a template's global.json pins and how far .NET may roll forward from it.
type SDKNeed struct {
	Version     string `json:"version"`
	RollForward string `json:"rollForward"`
}

func readSDKNeed(dir string) SDKNeed {
	raw, err := safefs.ReadFile(filepath.Join(dir, "global.json"))
	if err != nil {
		return SDKNeed{}
	}
	var g struct {
		SDK SDKNeed `json:"sdk"`
	}
	if json.Unmarshal(raw, &g) != nil || !sdkVersion(g.SDK.Version) {
		return SDKNeed{}
	}
	return g.SDK
}

func sdkVersion(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if _, err := strconv.Atoi(strings.SplitN(p, "-", 2)[0]); err != nil {
			return false
		}
	}
	return true
}
