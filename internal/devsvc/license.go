package devsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Licenses are the licenses New mod offers besides the template's own; none of their texts names a holder or year.
var Licenses = []string{"apache-2.0", "gpl-3.0", "mpl-2.0", "unlicense"}

const licensesAPI = "https://api.github.com/licenses/"

// applyLicense writes the chosen license's text over the template's LICENSE, from GitHub's licenses API.
func (s *Service) applyLicense(ctx context.Context, dir, key string) error {
	if key == "" {
		return nil
	}
	if !slices.Contains(Licenses, key) {
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a license New mod offers", key))
	}
	base := licensesAPI
	if s.LicensesURL != "" {
		base = s.LicensesURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+key, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := s.httpClient().Do(req)
	if err != nil {
		return usererr.Wrap(usererr.Network, err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return usererr.Wrap(usererr.Network, fmt.Errorf("the %s license text: %s", key, res.Status))
	}
	var license struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&license); err != nil || license.Body == "" {
		return usererr.Wrap(usererr.Network, errors.New("the license text could not be read"))
	}
	return fsx.WriteFile(filepath.Join(dir, "LICENSE"), []byte(license.Body), 0o644) // #nosec G306 -- a tracked text file
}

// SDKNeed is the SDK a template's global.json pins and how far .NET may roll forward from it.
type SDKNeed struct {
	Version     string `json:"version"`
	RollForward string `json:"rollForward"`
}

func readSDKNeed(dir string) SDKNeed {
	raw, err := fsx.ReadFile(filepath.Join(dir, "global.json"))
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
